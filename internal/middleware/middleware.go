package middleware

import (
	"crypto/subtle"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/messkan/PromptCache/internal/logging"
	"github.com/messkan/PromptCache/internal/metrics"
)

// Auth validates the Bearer token in the Authorization header.
// If token is empty, auth is disabled and all requests pass through.
func Auth(token string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if token == "" {
			c.Next()
			return
		}

		authHeader := c.GetHeader("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}

		provided := authHeader[len("Bearer "):]
		if subtle.ConstantTimeCompare([]byte(provided), []byte(token)) != 1 {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "Unauthorized",
			})
			return
		}

		c.Next()
	}
}

// RequestID adds a unique request ID to each request
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// Logger logs request details using structured logging
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()

		requestID, _ := c.Get("request_id")

		if raw != "" {
			path = path + "?" + raw
		}

		logging.Info().
			Str("request_id", requestID.(string)).
			Str("method", method).
			Str("path", path).
			Int("status", statusCode).
			Dur("latency", latency).
			Str("client_ip", clientIP).
			Msg("Request completed")
	}
}

// Metrics records request metrics
func Metrics() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		m := metrics.Get()
		m.RecordRequest()

		c.Next()

		latency := time.Since(start)
		m.RecordLatency(latency)

		if c.Writer.Status() >= 400 {
			m.RecordFailedRequest()
		}
	}
}

// RequestSizeLimit limits the request body size
func RequestSizeLimit(maxBytes int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.ContentLength > maxBytes {
			c.AbortWithStatusJSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "Request body too large",
			})
			return
		}
		
		c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxBytes)
		c.Next()
	}
}

// Recovery recovers from panics and logs them
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")
				
				logging.Error().
					Str("request_id", requestID.(string)).
					Interface("error", err).
					Msg("Panic recovered")

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
			}
		}()
		
		c.Next()
	}
}

// CacheHeaders adds cache-related headers to responses
type CacheHeadersConfig struct {
	CacheHit   bool
	Score      float32
	CacheKey   string
	Provider   string
}

// SetCacheHeaders sets cache-related response headers
func SetCacheHeaders(c *gin.Context, cfg *CacheHeadersConfig) {
	if cfg.CacheHit {
		c.Header("X-Cache", "HIT")
	} else {
		c.Header("X-Cache", "MISS")
	}
	
	if cfg.Score > 0 {
		c.Header("X-Similarity-Score", formatFloat32(cfg.Score))
	}
	
	if cfg.CacheKey != "" {
		c.Header("X-Cache-Key", cfg.CacheKey)
	}
	
	if cfg.Provider != "" {
		c.Header("X-Provider", cfg.Provider)
	}
}

func formatFloat32(f float32) string {
	// Simple formatting to 4 decimal places
	intPart := int(f)
	fracPart := int((f - float32(intPart)) * 10000)
	
	result := make([]byte, 0, 8)
	if intPart == 0 {
		result = append(result, '0')
	} else {
		result = appendInt(result, intPart)
	}
	result = append(result, '.')
	
	// Pad with zeros
	if fracPart < 10 {
		result = append(result, '0', '0', '0')
	} else if fracPart < 100 {
		result = append(result, '0', '0')
	} else if fracPart < 1000 {
		result = append(result, '0')
	}
	result = appendInt(result, fracPart)
	
	return string(result)
}

func appendInt(b []byte, i int) []byte {
	if i == 0 {
		return append(b, '0')
	}
	
	var tmp [20]byte
	pos := len(tmp)
	for i > 0 {
		pos--
		tmp[pos] = byte('0' + i%10)
		i /= 10
	}
	return append(b, tmp[pos:]...)
}
