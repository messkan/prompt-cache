package main

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/messkan/PromptCache/internal/cache"
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
	// Initialize Storage
	store, err := storage.NewBadgerStore("./badger_data")
	if err != nil {
		log.Fatalf("Failed to initialize BadgerDB: %v", err)
	}
	defer store.Close()

	// Initialize Semantic Engine
	openaiProvider := semantic.NewOpenAIProvider()
	semanticEngine := semantic.NewSemanticEngine(openaiProvider, store, openaiProvider, 0.95, 0.80)

	// Initialize Cache
	c := cache.NewCache(store)

	r := gin.Default()

	r.POST("/v1/chat/completions", func(cGin *gin.Context) {
		var req ChatCompletionRequest
		// We need to read the body but also keep it for forwarding
		bodyBytes, err := io.ReadAll(cGin.Request.Body)
		if err != nil {
			cGin.JSON(http.StatusBadRequest, gin.H{"error": "Failed to read request body"})
			return
		}
		// Restore body for binding
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
			log.Printf("Semantic search error: %v", err)
		}

		if similarKey != "" {
			log.Printf("🔥 Cache HIT! Score: %f, Key: %s", score, similarKey)
			// The key in semantic storage has "emb:" prefix, but cache storage does not.
			actualKey := strings.TrimPrefix(similarKey, "emb:")
			cachedResp, found, err := c.Get(ctx, actualKey)
			if err == nil && found {
				cGin.Data(http.StatusOK, "application/json", cachedResp)
				return
			}
		}

		log.Println("💨 Cache MISS. Forwarding to OpenAI...")

		// 2. Forward to OpenAI
		apiKey := os.Getenv("OPENAI_API_KEY")
		openAIReq, _ := http.NewRequest("POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(bodyBytes))
		openAIReq.Header.Set("Content-Type", "application/json")
		openAIReq.Header.Set("Authorization", "Bearer "+apiKey)

		client := &http.Client{}
		resp, err := client.Do(openAIReq)
		if err != nil {
			log.Printf("Failed to call OpenAI: %v", err)
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to call OpenAI: " + err.Error()})
			return
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			log.Printf("Failed to read OpenAI response: %v", err)
			cGin.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read OpenAI response: " + err.Error()})
			return
		}

		// 3. Cache Response & Embedding
		if resp.StatusCode == http.StatusOK {
			key := cache.GenerateKey(prompt)

			// Save Response
			if err := c.Set(ctx, key, respBody, 24*time.Hour); err != nil {
				log.Printf("Failed to cache response: %v", err)
			}

			// Save Prompt for Verification
			if err := store.Set(ctx, "prompt:"+key, []byte(prompt)); err != nil {
				log.Printf("Failed to save prompt: %v", err)
			}

			// Save Embedding
			embedding, err := openaiProvider.Embed(ctx, prompt)
			if err == nil {
				embBytes := semantic.Float32ToBytes(embedding)
				if err := store.Set(ctx, "emb:"+key, embBytes); err != nil {
					log.Printf("Failed to save embedding: %v", err)
				}
			} else {
				log.Printf("Failed to generate embedding: %v", err)
			}
		}

		cGin.Data(resp.StatusCode, "application/json", respBody)
	})

	log.Println("🚀 PromptCache Server running on :8080")
	r.Run(":8080")
}
