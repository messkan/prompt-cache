package semantic

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	internalhttp "github.com/messkan/PromptCache/internal/http"
	"github.com/messkan/PromptCache/internal/metrics"
)

// streamingChunk is the common OpenAI SSE chunk structure used for buffering.
type streamingChunk struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index        int    `json:"index"`
		Delta        delta  `json:"delta"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
}

type delta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`
}

// pipeSSEAndBuffer reads an OpenAI-compatible SSE stream from r, writes each raw
// line to w (flushing after every event boundary), and returns a buffered
// non-streaming chat.completion JSON for caching.
func pipeSSEAndBuffer(r io.Reader, w http.ResponseWriter) ([]byte, error) {
	flusher, canFlush := w.(http.Flusher)

	var (
		fullContent strings.Builder
		chunkID     string
		chunkModel  string
		chunkCreated int64
	)

	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := scanner.Text()

		// Write line verbatim to client
		_, _ = w.Write([]byte(line + "\n"))

		if line == "" {
			// SSE event boundary — flush to client
			if canFlush {
				flusher.Flush()
			}
			continue
		}

		if strings.HasPrefix(line, "data: ") {
			data := line[len("data: "):]
			if data == "[DONE]" {
				// Trailing newline already omitted; write final blank line
				_, _ = w.Write([]byte("\n"))
				if canFlush {
					flusher.Flush()
				}
				break
			}
			var chunk streamingChunk
			if err := json.Unmarshal([]byte(data), &chunk); err == nil {
				if chunkID == "" {
					chunkID = chunk.ID
					chunkModel = chunk.Model
					chunkCreated = chunk.Created
				}
				for _, c := range chunk.Choices {
					fullContent.WriteString(c.Delta.Content)
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// Assemble buffered non-streaming response for caching
	resp := OpenAIChatCompletionResponse{
		ID:      chunkID,
		Object:  "chat.completion",
		Created: chunkCreated,
		Model:   chunkModel,
		Choices: []OpenAIChoice{
			{
				Index: 0,
				Message: OpenAIMessage{
					Role:    "assistant",
					Content: fullContent.String(),
				},
				FinishReason: "stop",
			},
		},
	}
	return json.Marshal(resp)
}

// setSSEHeaders sets Server-Sent Events headers on the response writer.
func setSSEHeaders(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
}

// ProviderConfig holds configuration for providers
type ProviderConfig struct {
	HTTPTimeout       time.Duration
	HTTPMaxRetries    int
	HTTPRetryBaseWait time.Duration

	// OpenAI settings
	OpenAIEmbedModel  string
	OpenAIVerifyModel string

	// Mistral settings
	MistralEmbedModel  string
	MistralVerifyModel string

	// Claude settings
	ClaudeModel       string
	ClaudeVerifyModel string
	VoyageEmbedModel  string
}

// DefaultProviderConfig returns default provider configuration
func DefaultProviderConfig() *ProviderConfig {
	return &ProviderConfig{
		HTTPTimeout:        30 * time.Second,
		HTTPMaxRetries:     3,
		HTTPRetryBaseWait:  500 * time.Millisecond,
		OpenAIEmbedModel:   getEnvOrDefault("OPENAI_EMBED_MODEL", "text-embedding-3-small"),
		OpenAIVerifyModel:  getEnvOrDefault("OPENAI_VERIFY_MODEL", "gpt-4o-mini"),
		MistralEmbedModel:  getEnvOrDefault("MISTRAL_EMBED_MODEL", "mistral-embed"),
		MistralVerifyModel: getEnvOrDefault("MISTRAL_VERIFY_MODEL", "mistral-small-latest"),
		ClaudeModel:        getEnvOrDefault("CLAUDE_MODEL", "claude-3-opus-20240229"),
		ClaudeVerifyModel:  getEnvOrDefault("CLAUDE_VERIFY_MODEL", "claude-3-haiku-20240307"),
		VoyageEmbedModel:   getEnvOrDefault("VOYAGE_EMBED_MODEL", "voyage-3"),
	}
}

func getEnvOrDefault(key, defaultVal string) string {
	if val := os.Getenv(key); val != "" {
		return val
	}
	return defaultVal
}

// OpenAIProvider implementation
type OpenAIProvider struct {
	apiKey     string
	client     *internalhttp.RetryableClient
	embedModel string
	chatModel  string
}

func NewOpenAIProvider() *OpenAIProvider {
	return NewOpenAIProviderWithConfig(DefaultProviderConfig())
}

func NewOpenAIProviderWithConfig(cfg *ProviderConfig) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey: os.Getenv("OPENAI_API_KEY"),
		client: internalhttp.NewRetryableClient(&internalhttp.ClientConfig{
			Timeout:       cfg.HTTPTimeout,
			MaxRetries:    cfg.HTTPMaxRetries,
			RetryBaseWait: cfg.HTTPRetryBaseWait,
		}),
		embedModel: cfg.OpenAIEmbedModel,
		chatModel:  cfg.OpenAIVerifyModel,
	}
}

type EmbeddingRequest struct {
	Input string `json:"input"`
	Model string `json:"model"`
}

type EmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (p *OpenAIProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	reqBody := EmbeddingRequest{
		Input: text,
		Model: p.embedModel,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return nil, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		m.RecordProviderError()
		return nil, err
	}

	if len(embeddingResp.Data) == 0 {
		m.RecordProviderError()
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert float64 to float32
	res := make([]float32, len(embeddingResp.Data[0].Embedding))
	for i, v := range embeddingResp.Data[0].Embedding {
		res[i] = float32(v)
	}

	return res, nil
}

type VerificationRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type VerificationResponse struct {
	Choices []struct {
		Message Message `json:"message"`
	} `json:"choices"`
}

func (p *OpenAIProvider) ForwardChatCompletion(ctx context.Context, requestBody []byte) ([]byte, int, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.RecordProviderError()
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode != http.StatusOK {
		m.RecordProviderError()
	}

	return respBody, resp.StatusCode, nil
}

func (p *OpenAIProvider) ForwardStreamingChatCompletion(ctx context.Context, requestBody []byte, w http.ResponseWriter) ([]byte, int, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	// Ensure stream: true in the forwarded request
	var reqMap map[string]interface{}
	if err := json.Unmarshal(requestBody, &reqMap); err != nil {
		m.RecordProviderError()
		return nil, http.StatusBadRequest, err
	}
	reqMap["stream"] = true
	modifiedBody, err := json.Marshal(reqMap)
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(modifiedBody))
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return nil, resp.StatusCode, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	setSSEHeaders(w)

	buffered, err := pipeSSEAndBuffer(resp.Body, w)
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusOK, err
	}

	return buffered, http.StatusOK, nil
}

func (p *OpenAIProvider) CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	systemPrompt := "You are a semantic judge. Determine if the two user prompts have the exact same intent and meaning. Answer only with 'YES' or 'NO'."
	userPrompt := fmt.Sprintf("Prompt 1: %s\nPrompt 2: %s", prompt1, prompt2)

	reqBody := VerificationRequest{
		Model: p.chatModel,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		m.RecordProviderError()
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		m.RecordProviderError()
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return false, fmt.Errorf("OpenAI API error (%d): %s", resp.StatusCode, string(body))
	}

	var verResp VerificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&verResp); err != nil {
		m.RecordProviderError()
		return false, err
	}

	if len(verResp.Choices) == 0 {
		m.RecordProviderError()
		return false, fmt.Errorf("no choices returned")
	}

	content := strings.TrimSpace(strings.ToUpper(verResp.Choices[0].Message.Content))
	return content == "YES", nil
}

// MistralProvider implementation
type MistralProvider struct {
	apiKey      string
	client      *internalhttp.RetryableClient
	embedModel  string
	verifyModel string
}

func NewMistralProvider() *MistralProvider {
	return NewMistralProviderWithConfig(DefaultProviderConfig())
}

func NewMistralProviderWithConfig(cfg *ProviderConfig) *MistralProvider {
	return &MistralProvider{
		apiKey: os.Getenv("MISTRAL_API_KEY"),
		client: internalhttp.NewRetryableClient(&internalhttp.ClientConfig{
			Timeout:       cfg.HTTPTimeout,
			MaxRetries:    cfg.HTTPMaxRetries,
			RetryBaseWait: cfg.HTTPRetryBaseWait,
		}),
		embedModel:  cfg.MistralEmbedModel,
		verifyModel: cfg.MistralVerifyModel,
	}
}

type MistralEmbeddingRequest struct {
	Input          []string `json:"input"`
	Model          string   `json:"model"`
	EncodingFormat string   `json:"encoding_format"`
}

type MistralEmbeddingResponse struct {
	Data []struct {
		Embedding []float64 `json:"embedding"`
	} `json:"data"`
}

func (p *MistralProvider) Embed(ctx context.Context, text string) ([]float32, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	reqBody := MistralEmbeddingRequest{
		Input:          []string{text},
		Model:          p.embedModel,
		EncodingFormat: "float",
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return nil, fmt.Errorf("Mistral API error (%d): %s", resp.StatusCode, string(body))
	}

	var embeddingResp MistralEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		m.RecordProviderError()
		return nil, err
	}

	if len(embeddingResp.Data) == 0 {
		m.RecordProviderError()
		return nil, fmt.Errorf("no embedding data returned")
	}

	// Convert float64 to float32
	res := make([]float32, len(embeddingResp.Data[0].Embedding))
	for i, v := range embeddingResp.Data[0].Embedding {
		res[i] = float32(v)
	}

	return res, nil
}

type MistralChatRequest struct {
	Model    string           `json:"model"`
	Messages []MistralMessage `json:"messages"`
}

type MistralMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type MistralChatResponse struct {
	Choices []struct {
		Message MistralMessage `json:"message"`
	} `json:"choices"`
}

func (p *MistralProvider) ForwardChatCompletion(ctx context.Context, requestBody []byte) ([]byte, int, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(requestBody))
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.RecordProviderError()
		return nil, resp.StatusCode, err
	}

	if resp.StatusCode != http.StatusOK {
		m.RecordProviderError()
	}

	return respBody, resp.StatusCode, nil
}

func (p *MistralProvider) ForwardStreamingChatCompletion(ctx context.Context, requestBody []byte, w http.ResponseWriter) ([]byte, int, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	// Ensure stream: true in the forwarded request
	var reqMap map[string]interface{}
	if err := json.Unmarshal(requestBody, &reqMap); err != nil {
		m.RecordProviderError()
		return nil, http.StatusBadRequest, err
	}
	reqMap["stream"] = true
	modifiedBody, err := json.Marshal(reqMap)
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(modifiedBody))
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return nil, resp.StatusCode, fmt.Errorf("Mistral API error (%d): %s", resp.StatusCode, string(body))
	}

	setSSEHeaders(w)

	buffered, err := pipeSSEAndBuffer(resp.Body, w)
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusOK, err
	}

	return buffered, http.StatusOK, nil
}

func (p *MistralProvider) CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	systemPrompt := "You are a semantic judge. Determine if the two user prompts have the exact same intent and meaning. Answer only with 'YES' or 'NO'."
	userPrompt := fmt.Sprintf("Prompt 1: %s\nPrompt 2: %s", prompt1, prompt2)

	reqBody := MistralChatRequest{
		Model: p.verifyModel,
		Messages: []MistralMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		m.RecordProviderError()
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		m.RecordProviderError()
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return false, fmt.Errorf("Mistral API error (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp MistralChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		m.RecordProviderError()
		return false, err
	}

	if len(chatResp.Choices) == 0 {
		m.RecordProviderError()
		return false, fmt.Errorf("no choices returned")
	}

	content := strings.TrimSpace(strings.ToUpper(chatResp.Choices[0].Message.Content))
	return content == "YES", nil
}

// ClaudeProvider implementation
type ClaudeProvider struct {
	apiKey       string
	client       *internalhttp.RetryableClient
	chatModel    string
	verifyModel  string
	voyageModel  string
}

func NewClaudeProvider() *ClaudeProvider {
	return NewClaudeProviderWithConfig(DefaultProviderConfig())
}

func NewClaudeProviderWithConfig(cfg *ProviderConfig) *ClaudeProvider {
	return &ClaudeProvider{
		apiKey: os.Getenv("ANTHROPIC_API_KEY"),
		client: internalhttp.NewRetryableClient(&internalhttp.ClientConfig{
			Timeout:       cfg.HTTPTimeout,
			MaxRetries:    cfg.HTTPMaxRetries,
			RetryBaseWait: cfg.HTTPRetryBaseWait,
		}),
		chatModel:   cfg.ClaudeModel,
		verifyModel: cfg.ClaudeVerifyModel,
		voyageModel: cfg.VoyageEmbedModel,
	}
}

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
	Usage   any            `json:"usage"`
}

type OpenAIChoice struct {
	Index        int           `json:"index"`
	Message      OpenAIMessage `json:"message"`
	FinishReason string        `json:"finish_reason"`
}

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
	m := metrics.Get()
	m.RecordProviderCall()

	voyageAPIKey := os.Getenv("VOYAGE_API_KEY")
	if voyageAPIKey == "" {
		m.RecordProviderError()
		return nil, fmt.Errorf("VOYAGE_API_KEY not set - required for Claude provider embeddings")
	}

	reqBody := VoyageEmbeddingRequest{
		Input: []string{text},
		Model: p.voyageModel,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.voyageai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+voyageAPIKey)

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return nil, fmt.Errorf("Voyage AI API error (%d): %s", resp.StatusCode, string(body))
	}

	var embeddingResp VoyageEmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		m.RecordProviderError()
		return nil, err
	}

	if len(embeddingResp.Data) == 0 {
		m.RecordProviderError()
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
	m := metrics.Get()
	m.RecordProviderCall()

	// 1. Unmarshal the incoming OpenAI-compatible request
	var openAIReq OpenAIChatCompletionRequest
	if err := json.Unmarshal(requestBody, &openAIReq); err != nil {
		m.RecordProviderError()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	if openAIReq.Stream {
		m.RecordProviderError()
		return nil, http.StatusNotImplemented, fmt.Errorf("streaming is not supported for claude provider yet")
	}

	// 2. Translate to Claude's request format
	claudeReq := ClaudeChatRequest{
		Model:     p.chatModel,
		MaxTokens: 1024,
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
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal claude request: %w", err)
	}

	// 4. Send the request to Claude's API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(claudeBody))
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, err
	}
	defer resp.Body.Close()

	claudeRespBody, err := io.ReadAll(resp.Body)
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, err
	}

	if resp.StatusCode != http.StatusOK {
		m.RecordProviderError()
		return claudeRespBody, resp.StatusCode, fmt.Errorf("claude API error: %s", string(claudeRespBody))
	}

	// 5. Unmarshal Claude's response
	var claudeResp ClaudeChatResponse
	if err := json.Unmarshal(claudeRespBody, &claudeResp); err != nil {
		m.RecordProviderError()
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
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal final response: %w", err)
	}

	return finalRespBody, http.StatusOK, nil
}

func (p *ClaudeProvider) ForwardStreamingChatCompletion(ctx context.Context, requestBody []byte, w http.ResponseWriter) ([]byte, int, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	// 1. Parse incoming OpenAI-compatible request
	var openAIReq OpenAIChatCompletionRequest
	if err := json.Unmarshal(requestBody, &openAIReq); err != nil {
		m.RecordProviderError()
		return nil, http.StatusBadRequest, fmt.Errorf("failed to unmarshal request body: %w", err)
	}

	// 2. Translate to Claude request format with stream: true
	claudeReq := ClaudeChatRequest{
		Model:     p.chatModel,
		MaxTokens: 1024,
		Stream:    true,
	}
	var claudeMessages []ClaudeMessage
	for _, msg := range openAIReq.Messages {
		if msg.Role == "system" {
			claudeReq.System = msg.Content
		} else {
			claudeMessages = append(claudeMessages, ClaudeMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}
	claudeReq.Messages = claudeMessages

	claudeBody, err := json.Marshal(claudeReq)
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, fmt.Errorf("failed to marshal claude request: %w", err)
	}

	// 3. Call Claude streaming API
	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(claudeBody))
	if err != nil {
		m.RecordProviderError()
		return nil, http.StatusInternalServerError, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return nil, 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return nil, resp.StatusCode, fmt.Errorf("claude API error (%d): %s", resp.StatusCode, string(body))
	}

	// 4. Set SSE headers on the client response
	setSSEHeaders(w)
	flusher, canFlush := w.(http.Flusher)

	// Claude SSE event structures
	type claudeEventData struct {
		Type  string `json:"type"`
		Index int    `json:"index"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
		Message struct {
			ID    string `json:"id"`
			Model string `json:"model"`
		} `json:"message"`
	}

	// Generate an ID and timestamp for the translated OpenAI events
	msgID := fmt.Sprintf("chatcmpl-claude-%d", time.Now().UnixNano())
	created := time.Now().Unix()
	model := p.chatModel

	// 5. Emit initial role-delta event so clients know the stream has started
	roleDelta := map[string]interface{}{
		"id": msgID, "object": "chat.completion.chunk", "created": created, "model": model,
		"choices": []map[string]interface{}{
			{"index": 0, "delta": map[string]string{"role": "assistant"}, "finish_reason": nil},
		},
	}
	roleData, _ := json.Marshal(roleDelta)
	_, _ = w.Write([]byte("data: " + string(roleData) + "\n\n"))
	if canFlush {
		flusher.Flush()
	}

	// 6. Stream Claude events, translate to OpenAI SSE format
	var fullContent strings.Builder
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := line[len("data: "):]

		var evt claudeEventData
		if err := json.Unmarshal([]byte(data), &evt); err != nil {
			continue
		}

		switch evt.Type {
		case "message_start":
			if evt.Message.ID != "" {
				msgID = evt.Message.ID
			}
			if evt.Message.Model != "" {
				model = evt.Message.Model
			}

		case "content_block_delta":
			if evt.Delta.Type == "text_delta" && evt.Delta.Text != "" {
				fullContent.WriteString(evt.Delta.Text)

				contentChunk := map[string]interface{}{
					"id": msgID, "object": "chat.completion.chunk", "created": created, "model": model,
					"choices": []map[string]interface{}{
						{"index": 0, "delta": map[string]string{"content": evt.Delta.Text}, "finish_reason": nil},
					},
				}
				chunkData, _ := json.Marshal(contentChunk)
				_, _ = w.Write([]byte("data: " + string(chunkData) + "\n\n"))
				if canFlush {
					flusher.Flush()
				}
			}

		case "message_stop":
			// Emit stop chunk
			stopChunk := map[string]interface{}{
				"id": msgID, "object": "chat.completion.chunk", "created": created, "model": model,
				"choices": []map[string]interface{}{
					{"index": 0, "delta": map[string]interface{}{}, "finish_reason": "stop"},
				},
			}
			stopData, _ := json.Marshal(stopChunk)
			_, _ = w.Write([]byte("data: " + string(stopData) + "\n\n"))
			_, _ = w.Write([]byte("data: [DONE]\n\n"))
			if canFlush {
				flusher.Flush()
			}
		}
	}

	if err := scanner.Err(); err != nil {
		m.RecordProviderError()
		return nil, http.StatusOK, err
	}

	// 7. Assemble buffered response for caching
	buffered := OpenAIChatCompletionResponse{
		ID:      msgID,
		Object:  "chat.completion",
		Created: created,
		Model:   model,
		Choices: []OpenAIChoice{
			{
				Index:        0,
				Message:      OpenAIMessage{Role: "assistant", Content: fullContent.String()},
				FinishReason: "stop",
			},
		},
	}
	finalBody, err := json.Marshal(buffered)
	if err != nil {
		return nil, http.StatusOK, err
	}
	return finalBody, http.StatusOK, nil
}

func (p *ClaudeProvider) CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error) {
	m := metrics.Get()
	m.RecordProviderCall()

	systemPrompt := "You are a semantic judge. Determine if the two user prompts have the exact same intent and meaning. Answer only with 'YES' or 'NO'."
	userPrompt := fmt.Sprintf("Prompt 1: %s\nPrompt 2: %s", prompt1, prompt2)

	reqBody := ClaudeChatRequest{
		Model:     p.verifyModel,
		MaxTokens: 10,
		System:    systemPrompt,
		Messages: []ClaudeMessage{
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		m.RecordProviderError()
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.anthropic.com/v1/messages", bytes.NewBuffer(jsonBody))
	if err != nil {
		m.RecordProviderError()
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", p.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(req)
	if err != nil {
		m.RecordProviderError()
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		m.RecordProviderError()
		return false, fmt.Errorf("Claude API error (%d): %s", resp.StatusCode, string(body))
	}

	var chatResp ClaudeChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		m.RecordProviderError()
		return false, err
	}

	if len(chatResp.Content) == 0 {
		m.RecordProviderError()
		return false, fmt.Errorf("no content returned")
	}

	content := strings.TrimSpace(strings.ToUpper(chatResp.Content[0].Text))
	return content == "YES", nil
}
