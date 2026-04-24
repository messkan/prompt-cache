package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/messkan/PromptCache/internal/cache"
	"github.com/messkan/PromptCache/internal/config"
	"github.com/messkan/PromptCache/internal/logging"
	"github.com/messkan/PromptCache/internal/metrics"
	"github.com/messkan/PromptCache/internal/middleware"
	"github.com/messkan/PromptCache/internal/semantic"
	"github.com/messkan/PromptCache/internal/storage"
)

type ChatCompletionRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func main() {
	// Load configuration
	cfg := config.Load()

	// Initialize structured logging
	logLevel := os.Getenv("LOG_LEVEL")
	if logLevel == "" {
		logLevel = "info"
	}
	logging.Init(logLevel)

	logging.Info().
		Str("port", cfg.Port).
		Str("storage_path", cfg.StoragePath).
		Dur("cache_ttl", cfg.CacheTTL).
		Int("max_entries", cfg.MaxEntries).
		Float32("high_threshold", cfg.HighThreshold).
		Float32("low_threshold", cfg.LowThreshold).
		Msg("Starting PromptCache server")

	// Initialize Storage
	store, err := storage.NewBadgerStore(cfg.StoragePath)
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to initialize BadgerDB")
	}

	// Initialize Semantic Engine with provider from environment
	provider, err := semantic.NewProvider()
	if err != nil {
		logging.Fatal().Err(err).Msg("Failed to initialize embedding provider")
	}

	// Load semantic configuration
	semanticConfig := &semantic.Config{
		HighThreshold:          cfg.HighThreshold,
		LowThreshold:           cfg.LowThreshold,
		EnableGrayZoneVerifier: cfg.EnableGrayZoneVerifier,
		EmbeddingDimension:     1536, // Default for OpenAI
		UseANNIndex:            true,
	}

	logging.Info().
		Float32("high_threshold", semanticConfig.HighThreshold).
		Float32("low_threshold", semanticConfig.LowThreshold).
		Bool("gray_zone_verifier", semanticConfig.EnableGrayZoneVerifier).
		Msg("Cache configuration loaded")

	semanticEngine := semantic.NewSemanticEngine(provider, store, provider, semanticConfig)

	// Initialize Cache with configuration
	cacheConfig := &cache.Config{
		TTL:             cfg.CacheTTL,
		MaxEntries:      cfg.MaxEntries,
		CleanupInterval: 1 * time.Hour,
	}
	c := cache.NewCacheWithConfig(store, cacheConfig)

	// Set Gin mode
	if os.Getenv("GIN_MODE") == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()

	// Apply global middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Metrics())
	r.Use(middleware.RequestSizeLimit(cfg.RequestMaxBytes))

	// ── Unprotected endpoints ────────────────────────────────────────────────

	// Health check endpoints (always public)
	r.GET("/health", func(cGin *gin.Context) {
		cGin.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.GET("/health/ready", func(cGin *gin.Context) {
		ctx := cGin.Request.Context()
		_, err := store.Count(ctx)
		if err != nil {
			cGin.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "storage not accessible",
			})
			return
		}
		cGin.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})

	r.GET("/health/live", func(cGin *gin.Context) {
		cGin.JSON(http.StatusOK, gin.H{
			"status": "alive",
		})
	})

	// Main inference endpoint (public — not protected by auth)
	r.POST("/v1/chat/completions", func(cGin *gin.Context) {
		var req ChatCompletionRequest
		requestID, _ := cGin.Get("request_id")

		bodyBytes, err := io.ReadAll(cGin.Request.Body)
		if err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}
		cGin.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		if err := json.Unmarshal(bodyBytes, &req); err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		// Extract prompt (last user message)
		prompt := ""
		for i := len(req.Messages) - 1; i >= 0; i-- {
			if req.Messages[i].Role == "user" {
				prompt = req.Messages[i].Content
				break
			}
		}

		if prompt == "" {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "No user prompt found"})
			return
		}

		ctx := cGin.Request.Context()

		// 1. Check Semantic Cache
		similarKey, score, err := semanticEngine.FindSimilar(ctx, prompt)
		if err != nil {
			logging.Warn().
				Str("request_id", requestID.(string)).
				Err(err).
				Msg("Semantic search error")
		}

		if similarKey != "" {
			logging.Info().
				Str("request_id", requestID.(string)).
				Float32("score", score).
				Str("key", similarKey).
				Msg("Cache HIT")

			actualKey := strings.TrimPrefix(similarKey, "emb:")
			cachedResp, found, err := c.Get(ctx, actualKey)
			if err == nil && found {
				middleware.SetCacheHeaders(cGin, &middleware.CacheHeadersConfig{
					CacheHit: true,
					Score:    score,
					CacheKey: actualKey,
					Provider: semanticEngine.GetCurrentProvider(),
				})

				if req.Stream {
					// Synthesize OpenAI-compatible SSE stream from the cached JSON response
					var cached struct {
						ID      string `json:"id"`
						Model   string `json:"model"`
						Created int64  `json:"created"`
						Choices []struct {
							Message struct {
								Content string `json:"content"`
							} `json:"message"`
						} `json:"choices"`
					}
					cGin.Header("Content-Type", "text/event-stream")
					cGin.Header("Cache-Control", "no-cache")
					cGin.Header("Connection", "keep-alive")
					cGin.Header("X-Accel-Buffering", "no")

					if json.Unmarshal(cachedResp, &cached) == nil && len(cached.Choices) > 0 {
						id := cached.ID
						if id == "" {
							shortKey := actualKey
							if len(shortKey) > 8 {
								shortKey = shortKey[:8]
							}
							id = fmt.Sprintf("chatcmpl-cached-%s", shortKey)
						}
						model := cached.Model
						if model == "" {
							model = req.Model
						}
						created := cached.Created
						if created == 0 {
							created = time.Now().Unix()
						}
						content := cached.Choices[0].Message.Content

						emitSSE := func(v interface{}) {
							data, _ := json.Marshal(v)
							_, _ = cGin.Writer.Write([]byte("data: " + string(data) + "\n\n"))
							cGin.Writer.Flush()
						}
						emitSSE(map[string]interface{}{
							"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
							"choices": []map[string]interface{}{{"index": 0, "delta": map[string]string{"role": "assistant"}, "finish_reason": nil}},
						})
						emitSSE(map[string]interface{}{
							"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
							"choices": []map[string]interface{}{{"index": 0, "delta": map[string]string{"content": content}, "finish_reason": nil}},
						})
						emitSSE(map[string]interface{}{
							"id": id, "object": "chat.completion.chunk", "created": created, "model": model,
							"choices": []map[string]interface{}{{"index": 0, "delta": map[string]interface{}{}, "finish_reason": "stop"}},
						})
						_, _ = cGin.Writer.Write([]byte("data: [DONE]\n\n"))
						cGin.Writer.Flush()
					} else {
						// Fallback: emit cached bytes as a single content chunk so clients still get the response
						shortKey := actualKey
						if len(shortKey) > 8 {
							shortKey = shortKey[:8]
						}
						id := fmt.Sprintf("chatcmpl-cached-%s", shortKey)
						created := time.Now().Unix()
						emitSSE := func(v interface{}) {
							data, _ := json.Marshal(v)
							_, _ = cGin.Writer.Write([]byte("data: " + string(data) + "\n\n"))
							cGin.Writer.Flush()
						}
						emitSSE(map[string]interface{}{
							"id": id, "object": "chat.completion.chunk", "created": created, "model": req.Model,
							"choices": []map[string]interface{}{{"index": 0, "delta": map[string]string{"role": "assistant", "content": string(cachedResp)}, "finish_reason": "stop"}},
						})
						_, _ = cGin.Writer.Write([]byte("data: [DONE]\n\n"))
						cGin.Writer.Flush()
					}
					return
				}

				cGin.Data(http.StatusOK, "application/json", cachedResp)
				return
			}
		}

		logging.Info().
			Str("request_id", requestID.(string)).
			Str("provider", semanticEngine.GetCurrentProvider()).
			Bool("stream", req.Stream).
			Msg("Cache MISS - forwarding to provider")

		var respBody []byte
		var statusCode int

		if req.Stream {
			// Stream the response directly to the client and get buffered body for caching
			respBody, statusCode, err = semanticEngine.ForwardStreamingChatCompletion(ctx, bodyBytes, cGin.Writer)
		} else {
			respBody, statusCode, err = semanticEngine.ForwardChatCompletion(ctx, bodyBytes)
		}

		if err != nil {
			logging.Error().
				Str("request_id", requestID.(string)).
				Err(err).
				Msg("Provider call failed")
			if req.Stream {
				// Headers may already be flushed; emit an SSE error event so the client sees something
				errPayload, _ := json.Marshal(gin.H{"error": "Failed to call provider: " + err.Error()})
				_, _ = cGin.Writer.Write([]byte("data: " + string(errPayload) + "\n\n"))
				_, _ = cGin.Writer.Write([]byte("data: [DONE]\n\n"))
				cGin.Writer.Flush()
			} else {
				cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call provider: " + err.Error()})
			}
			return
		}

		// 3. Cache Response & Embedding (for both streaming and non-streaming)
		if statusCode == http.StatusOK && len(respBody) > 0 {
			key := cache.GenerateKey(prompt)

			if err := c.Set(ctx, key, respBody, cfg.CacheTTL); err != nil {
				logging.Warn().Str("request_id", requestID.(string)).Err(err).Msg("Failed to cache response")
			}

			if err := store.Set(ctx, "prompt:"+key, []byte(prompt)); err != nil {
				logging.Warn().Str("request_id", requestID.(string)).Err(err).Msg("Failed to save prompt")
			}

			embedding, err := semanticEngine.GetProvider().Embed(ctx, prompt)
			if err == nil {
				embBytes := semantic.Float32ToBytes(embedding)
				if err := store.Set(ctx, "emb:"+key, embBytes); err != nil {
					logging.Warn().Str("request_id", requestID.(string)).Err(err).Msg("Failed to save embedding")
				} else {
					semanticEngine.AddToIndex("emb:"+key, embedding)
				}
			} else {
				logging.Warn().Str("request_id", requestID.(string)).Err(err).Msg("Failed to generate embedding")
			}
		}

		if !req.Stream {
			middleware.SetCacheHeaders(cGin, &middleware.CacheHeadersConfig{
				CacheHit: false,
				Score:    score,
				Provider: semanticEngine.GetCurrentProvider(),
			})
			cGin.Data(statusCode, "application/json", respBody)
		}
	})

	// ── Protected management endpoints (auth-gated) ───────────────────────────
	if cfg.AuthToken == "" {
		logging.Warn().Msg("API_AUTH_TOKEN is not set — management endpoints (/metrics, /v1/stats, /v1/config, /v1/cache, /v1/cache/warm) are UNPROTECTED")
	} else {
		logging.Info().Msg("Auth enabled — management endpoints require Bearer token")
	}
	protected := r.Group("/")
	protected.Use(middleware.Auth(cfg.AuthToken))

	// Metrics
	protected.GET("/metrics", func(cGin *gin.Context) {
		m := metrics.Get()
		cGin.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(m.GetPrometheusMetrics()))
	})

	protected.GET("/v1/stats", func(cGin *gin.Context) {
		m := metrics.Get()
		cGin.JSON(http.StatusOK, m.GetStats())
	})

	// Configuration endpoints
	protected.GET("/v1/config", func(cGin *gin.Context) {
		semCfg := semanticEngine.GetConfig()
		cGin.JSON(http.StatusOK, gin.H{
			"provider":                   semanticEngine.GetCurrentProvider(),
			"available_providers":        []string{"openai", "mistral", "claude"},
			"high_threshold":             semCfg["high_threshold"],
			"low_threshold":              semCfg["low_threshold"],
			"enable_gray_zone_verifier":  semCfg["enable_gray_zone_verifier"],
			"cache_ttl_hours":            cfg.CacheTTL.Hours(),
			"cache_max_entries":          cfg.MaxEntries,
		})
	})

	protected.PATCH("/v1/config", func(cGin *gin.Context) {
		var body struct {
			HighThreshold          *float32 `json:"high_threshold"`
			LowThreshold           *float32 `json:"low_threshold"`
			EnableGrayZoneVerifier *bool    `json:"enable_gray_zone_verifier"`
		}
		if err := cGin.ShouldBindJSON(&body); err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}

		current := semanticEngine.GetConfig()
		high := current["high_threshold"].(float32)
		low := current["low_threshold"].(float32)
		if body.HighThreshold != nil {
			high = *body.HighThreshold
		}
		if body.LowThreshold != nil {
			low = *body.LowThreshold
		}

		if err := semanticEngine.UpdateThresholds(high, low, body.EnableGrayZoneVerifier); err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		updated := semanticEngine.GetConfig()
		logging.Info().
			Float64("high_threshold", float64(high)).
			Float64("low_threshold", float64(low)).
			Msg("Config thresholds updated")
		cGin.JSON(http.StatusOK, gin.H{
			"provider":                   semanticEngine.GetCurrentProvider(),
			"available_providers":        []string{"openai", "mistral", "claude"},
			"high_threshold":             updated["high_threshold"],
			"low_threshold":              updated["low_threshold"],
			"enable_gray_zone_verifier":  updated["enable_gray_zone_verifier"],
			"cache_ttl_hours":            cfg.CacheTTL.Hours(),
			"cache_max_entries":          cfg.MaxEntries,
		})
	})

	// Provider switching (GET info is also available at /v1/config)
	protected.POST("/v1/config/provider", func(cGin *gin.Context) {
		var req struct {
			Provider string `json:"provider" binding:"required"`
		}
		if err := cGin.ShouldBindJSON(&req); err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "provider field is required"})
			return
		}
		if err := semanticEngine.SetProvider(req.Provider); err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		logging.Info().Str("provider", req.Provider).Msg("Provider switched")
		cGin.JSON(http.StatusOK, gin.H{
			"message":  "Provider updated successfully",
			"provider": req.Provider,
		})
	})

	// Cache management
	protected.GET("/v1/cache", func(cGin *gin.Context) {
		ctx := cGin.Request.Context()
		keys, err := store.GetAllKeys(ctx, "")
		if err != nil {
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list cache entries"})
			return
		}
		var cacheKeys []string
		for _, k := range keys {
			if !strings.HasPrefix(k, "emb:") && !strings.HasPrefix(k, "prompt:") {
				cacheKeys = append(cacheKeys, k)
			}
		}
		cGin.JSON(http.StatusOK, gin.H{
			"count": len(cacheKeys),
			"keys":  cacheKeys,
		})
	})

	protected.DELETE("/v1/cache", func(cGin *gin.Context) {
		ctx := cGin.Request.Context()
		if err := c.Clear(ctx); err != nil {
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear cache"})
			return
		}
		logging.Info().Msg("Cache cleared")
		cGin.JSON(http.StatusOK, gin.H{"message": "Cache cleared successfully"})
	})

	protected.DELETE("/v1/cache/:key", func(cGin *gin.Context) {
		ctx := cGin.Request.Context()
		key := cGin.Param("key")
		if err := c.Delete(ctx, key); err != nil {
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete cache entry"})
			return
		}
		_ = store.Delete(ctx, "emb:"+key)
		_ = store.Delete(ctx, "prompt:"+key)
		logging.Info().Str("key", key).Msg("Cache entry deleted")
		cGin.JSON(http.StatusOK, gin.H{
			"message": "Cache entry deleted successfully",
			"key":     key,
		})
	})

	// Cache warming — bulk pre-populate the cache from historical prompt/response pairs
	protected.POST("/v1/cache/warm", func(cGin *gin.Context) {
		var body struct {
			Entries []struct {
				Prompt   string          `json:"prompt"`
				Response json.RawMessage `json:"response"`
			} `json:"entries"`
		}
		if err := cGin.ShouldBindJSON(&body); err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body"})
			return
		}
		if len(body.Entries) == 0 {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "entries array must not be empty"})
			return
		}

		ctx := cGin.Request.Context()
		provider := semanticEngine.GetProvider()

		type warmResult struct {
			Key    string `json:"key,omitempty"`
			Status string `json:"status"`
			Error  string `json:"error,omitempty"`
		}

		results := make([]warmResult, 0, len(body.Entries))
		processed, failed := 0, 0

		for _, entry := range body.Entries {
			if entry.Prompt == "" {
				failed++
				results = append(results, warmResult{Status: "error", Error: "prompt is required"})
				continue
			}

			key := cache.GenerateKey(entry.Prompt)

			if err := c.Set(ctx, key, []byte(entry.Response), cfg.CacheTTL); err != nil {
				failed++
				results = append(results, warmResult{Key: key, Status: "error", Error: "failed to store response: " + err.Error()})
				continue
			}
			_ = store.Set(ctx, "prompt:"+key, []byte(entry.Prompt))

			embedding, err := provider.Embed(ctx, entry.Prompt)
			if err != nil {
				// Roll back the cached response if embedding fails
				_ = c.Delete(ctx, key)
				_ = store.Delete(ctx, "prompt:"+key)
				failed++
				results = append(results, warmResult{Key: key, Status: "error", Error: "embedding failed: " + err.Error()})
				continue
			}

			embBytes := semantic.Float32ToBytes(embedding)
			_ = store.Set(ctx, "emb:"+key, embBytes)
			semanticEngine.AddToIndex("emb:"+key, embedding)

			processed++
			results = append(results, warmResult{Key: key, Status: "ok"})
		}

		logging.Info().Int("processed", processed).Int("failed", failed).Msg("Cache warming completed")
		cGin.JSON(http.StatusOK, gin.H{
			"processed": processed,
			"failed":    failed,
			"entries":   results,
		})
	})

	// Create HTTP server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	// Start server in goroutine
	go func() {
		logging.Info().Str("port", cfg.Port).Msg("PromptCache server starting")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logging.Fatal().Err(err).Msg("Server failed to start")
		}
	}()

	// Wait for interrupt signal for graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logging.Info().Msg("Shutting down server...")

	// Give outstanding requests 30 seconds to complete
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Stop cache cleanup goroutine
	c.Stop()

	// Sync storage before shutdown
	if err := store.Sync(); err != nil {
		logging.Warn().Err(err).Msg("Failed to sync storage")
	}

	// Shutdown server
	if err := srv.Shutdown(ctx); err != nil {
		logging.Error().Err(err).Msg("Server forced to shutdown")
	}

	// Close storage
	store.Close()

	logging.Info().Msg("Server exited gracefully")
}
