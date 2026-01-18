package main

import (
	"bytes"
	"context"
	"encoding/json"
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

	// Apply middleware
	r.Use(middleware.Recovery())
	r.Use(middleware.RequestID())
	r.Use(middleware.Logger())
	r.Use(middleware.Metrics())
	r.Use(middleware.RequestSizeLimit(cfg.RequestMaxBytes))

	// Health check endpoints
	r.GET("/health", func(cGin *gin.Context) {
		cGin.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"time":   time.Now().UTC().Format(time.RFC3339),
		})
	})

	r.GET("/health/ready", func(cGin *gin.Context) {
		// Check if storage is accessible
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

	// Metrics endpoints
	r.GET("/metrics", func(cGin *gin.Context) {
		m := metrics.Get()
		cGin.Data(http.StatusOK, "text/plain; charset=utf-8", []byte(m.GetPrometheusMetrics()))
	})

	r.GET("/v1/stats", func(cGin *gin.Context) {
		m := metrics.Get()
		cGin.JSON(http.StatusOK, m.GetStats())
	})

	// Provider management endpoints
	r.GET("/v1/config/provider", func(cGin *gin.Context) {
		currentProvider := semanticEngine.GetCurrentProvider()
		cGin.JSON(http.StatusOK, gin.H{
			"provider":            currentProvider,
			"available_providers": []string{"openai", "mistral", "claude"},
		})
	})

	r.POST("/v1/config/provider", func(cGin *gin.Context) {
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

	// Cache management endpoints
	r.GET("/v1/cache", func(cGin *gin.Context) {
		ctx := cGin.Request.Context()
		keys, err := store.GetAllKeys(ctx, "")
		if err != nil {
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list cache entries"})
			return
		}

		// Filter out embedding and prompt keys
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

	r.DELETE("/v1/cache", func(cGin *gin.Context) {
		ctx := cGin.Request.Context()
		err := c.Clear(ctx)
		if err != nil {
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to clear cache"})
			return
		}

		logging.Info().Msg("Cache cleared")
		cGin.JSON(http.StatusOK, gin.H{
			"message": "Cache cleared successfully",
		})
	})

	r.DELETE("/v1/cache/:key", func(cGin *gin.Context) {
		ctx := cGin.Request.Context()
		key := cGin.Param("key")

		err := c.Delete(ctx, key)
		if err != nil {
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete cache entry"})
			return
		}

		// Also delete associated embedding and prompt
		_ = store.Delete(ctx, "emb:"+key)
		_ = store.Delete(ctx, "prompt:"+key)

		logging.Info().Str("key", key).Msg("Cache entry deleted")
		cGin.JSON(http.StatusOK, gin.H{
			"message": "Cache entry deleted successfully",
			"key":     key,
		})
	})

	// Main chat completion endpoint
	r.POST("/v1/chat/completions", func(cGin *gin.Context) {
		var req ChatCompletionRequest
		requestID, _ := cGin.Get("request_id")

		// Read the body
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
				cGin.Data(http.StatusOK, "application/json", cachedResp)
				return
			}
		}

		logging.Info().
			Str("request_id", requestID.(string)).
			Str("provider", semanticEngine.GetCurrentProvider()).
			Msg("Cache MISS - forwarding to provider")

		// 2. Forward to the current provider
		respBody, statusCode, err := semanticEngine.ForwardChatCompletion(ctx, bodyBytes)
		if err != nil {
			logging.Error().
				Str("request_id", requestID.(string)).
				Err(err).
				Msg("Provider call failed")
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call provider: " + err.Error()})
			return
		}

		// 3. Cache Response & Embedding
		if statusCode == http.StatusOK {
			key := cache.GenerateKey(prompt)

			// Save Response
			if err := c.Set(ctx, key, respBody, cfg.CacheTTL); err != nil {
				logging.Warn().
					Str("request_id", requestID.(string)).
					Err(err).
					Msg("Failed to cache response")
			}

			// Save Prompt for Verification
			if err := store.Set(ctx, "prompt:"+key, []byte(prompt)); err != nil {
				logging.Warn().
					Str("request_id", requestID.(string)).
					Err(err).
					Msg("Failed to save prompt")
			}

			// Save Embedding and add to ANN index
			embedding, err := semanticEngine.GetProvider().Embed(ctx, prompt)
			if err == nil {
				embBytes := semantic.Float32ToBytes(embedding)
				if err := store.Set(ctx, "emb:"+key, embBytes); err != nil {
					logging.Warn().
						Str("request_id", requestID.(string)).
						Err(err).
						Msg("Failed to save embedding")
				} else {
					// Add to ANN index
					semanticEngine.AddToIndex("emb:"+key, embedding)
				}
			} else {
				logging.Warn().
					Str("request_id", requestID.(string)).
					Err(err).
					Msg("Failed to generate embedding")
			}
		}

		middleware.SetCacheHeaders(cGin, &middleware.CacheHeadersConfig{
			CacheHit: false,
			Score:    score,
			Provider: semanticEngine.GetCurrentProvider(),
		})
		cGin.Data(statusCode, "application/json", respBody)
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
