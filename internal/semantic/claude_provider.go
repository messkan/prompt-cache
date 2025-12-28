package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
)

type ClaudeProvider struct {
	apiKey string
	client *http.Client
}

func NewClaudeProvider() *ClaudeProvider {
	return &ClaudeProvider{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		client: &http.Client{},
	}
}

// Claude doesn't have a native embeddings API, so we use Voyage AI which is recommended by Anthropic
// Alternatively, you can use OpenAI embeddings or other providers
type VoyageEmbeddingRequest struct {
	Input []string `json:"input"`
	Model string   `json:"model"`
}

type VoyageEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (p *ClaudeProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	// Using Voyage AI for embeddings (recommended by Anthropic)
	voyageAPIKey := os.Getenv("VOYAGE_API_KEY")
	if voyageAPIKey == "" {
		return nil, fmt.Errorf("VOYAGE_API_KEY not set - required for Claude provider embeddings")
	}

	reqBody := VoyageEmbeddingRequest{
		Input: []string{text},
		Model: "voyage-3",
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.voyageai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+voyageAPIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Voyage AI API error: %s", string(body))
	}

	var embeddingResp VoyageEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}

	if len(embeddingResp.Data) == 0 {
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert float64 to float32
	res := make([]float32, len(embeddingResp.Data[0].Embedding))
	for i, v := range embeddingResp.Data[0].Embedding {
		res[i] = float32(v)
	}

	return res, nil
}

type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ClaudeChatRequest struct {
	Model     string          `json:"model"`
	MaxTokens int             `json:"max_tokens"`
	System    string          `json:"system,omitempty"`
	Messages  []ClaudeMessage `json:"messages"`
}

type ClaudeChatResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

func (p *ClaudeProvider) CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error) {
	systemPrompt := "You are a semantic judge. Determine if the two user prompts have the exact same intent and meaning. Answer only with 'YES' or 'NO'."
	userPrompt := fmt.Sprintf("Prompt 1: %s\nPrompt 2: %s", prompt1, prompt2)

	reqBody := ClaudeChatRequest{
		Model:     "claude-3-haiku-20240307",
		MaxTokens: 10,
		System:    systemPrompt,
		Messages: []ClaudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("Claude API error: %s", string(body))
	}

	var chatResp ClaudeChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return false, err
	}

	if len(chatResp.Content) == 0 {
		return false, fmt.Errorf("no content returned")
	}

	content := chatResp.Content[0].Text
	return content == "YES", nil
}
