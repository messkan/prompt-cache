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

type MistralProvider struct {
	apiKey string
	client *http.Client
}

func NewMistralProvider() *MistralProvider {
	return &MistralProvider{
		apiKey: os.Getenv("MISTRAL_API_KEY"),
		client: &http.Client{},
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
	reqBody := MistralEmbeddingRequest{
		Input:          []string{text},
		Model:          "mistral-embed",
		EncodingFormat: "float",
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/embeddings", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Mistral API error: %s", string(body))
	}

	var embeddingResp MistralEmbeddingResponse
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

type MistralChatRequest struct {
	Model    string          `json:"model"`
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

func (p *MistralProvider) CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error) {
	systemPrompt := "You are a semantic judge. Determine if the two user prompts have the exact same intent and meaning. Answer only with 'YES' or 'NO'."
	userPrompt := fmt.Sprintf("Prompt 1: %s\nPrompt 2: %s", prompt1, prompt2)

	reqBody := MistralChatRequest{
		Model: "mistral-small-latest",
		Messages: []MistralMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.mistral.ai/v1/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return false, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("Mistral API error: %s", string(body))
	}

	var chatResp MistralChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return false, err
	}

	if len(chatResp.Choices) == 0 {
		return false, fmt.Errorf("no choices returned")
	}

	content := chatResp.Choices[0].Message.Content
	return content == "YES", nil
}
