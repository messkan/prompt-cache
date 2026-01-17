package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// OpenAI-compatible structures for translation
type OpenAIChatCompletionRequest struct {
	Model    string          `json:"model"`
	Messages []OpenAIMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type OpenAIMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type OpenAIChatCompletionResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []OpenAIChoice `json:"choices"`
	Usage   any            `json:"usage"` // Keep it flexible
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

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
	Stream    bool            `json:"stream"`
}

type ClaudeChatResponse struct {
	ID         string `json:"id"`
	Model      string `json:"model"`
	StopReason string `json:"stop_reason"`
	Content    []struct {
		Text string `json:"text"`
	} `json:"content"`
	Usage struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

func (p *ClaudeProvider) ForwardChatCompletion(ctx context.Context, requestBody []byte) ([]byte, int, error) {
	// 1. Unmarshal the incoming OpenAI-compatible request
	var openAIReq OpenAIChatCompletionRequest
	if err := json.Unmarshal(requestBody, &openAIReq); err != nil {
		return nil, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	if openAIReq.Stream {
		return nil, http.StatusNotImplemented, fmt.Errorf("streaming is not supported for claude provider yet")
	}

	// 2. Translate to Claude's request format
	claudeReq := ClaudeChatRequest{
		Model:     "claude-3-opus-20240229", // Or map from openAIReq.Model
		MaxTokens: 1024,                    // Claude requires MaxTokens
		Stream:    openAIReq.Stream,
	}

	// Separate system prompt from messages
	var messages []ClaudeMessage
	for _, msg := range openAIReq.Messages {
		if msg.Role == "system" {
			claudeReq.System = msg.Content
		} else {
			messages = append(messages, ClaudeMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	claudeReq.Messages = messages

	// 3. Marshal the new Claude request
	claudeBody, err := json.Marshal(claudeReq)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal claude request: %w", err)
	}

	// 4. Send the request to Claude's API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(claudeBody))
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	claudeRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}

	if resp.StatusCode != http.StatusOK {
		return claudeRespBody, resp.StatusCode, fmt.Errorf("claude API error: %s", string(claudeRespBody))
	}

	// 5. Unmarshal Claude's response
	var claudeResp ClaudeChatResponse
	if err := json.Unmarshal(claudeRespBody, &claudeResp); err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to unmarshal claude response: %w", err)
	}

	// 6. Translate back to OpenAI's response format
	openAIResp := OpenAIChatCompletionResponse{
		ID:      claudeResp.ID,
		Object:  "chat.completion",
		Created: time.Now().Unix(),
		Model:   claudeResp.Model,
		Usage:   claudeResp.Usage,
	}

	if len(claudeResp.Content) > 0 {
		openAIResp.Choices = []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: claudeResp.Content[0].Text,
				},
				FinishReason: claudeResp.StopReason,
			},
		}
	}

	// 7. Marshal the final OpenAI-compatible response
	finalRespBody, err := json.Marshal(openAIResp)
	if err != nil {
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal final response: %w", err)
	}

	return finalRespBody, http.StatusOK, nil
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
