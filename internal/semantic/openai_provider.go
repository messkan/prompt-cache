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

type OpenAIProvider struct {
	apiKey              string
	client              *http.Client
	verificationModel   string
}

func NewOpenAIProvider() *OpenAIProvider {
	return NewOpenAIProviderWithModel("gpt-4o-mini")
}

func NewOpenAIProviderWithModel(verificationModel string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:            os.Getenv("OPENAI_API_KEY"),
		client:            &http.Client{},
		verificationModel: verificationModel,
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
	reqBody := EmbeddingRequest{
		Input: text,
		Model: "text-embedding-3-small",
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/embeddings", bytes.NewBuffer(jsonBody))
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
		return nil, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var embeddingResp EmbeddingResponse
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

func (p *OpenAIProvider) CheckSimilarity(ctx context.Context, prompt1, prompt2 string) (bool, error) {
	systemPrompt := "You are a semantic judge. Determine if the two user prompts have the exact same intent and meaning. Answer only with 'YES' or 'NO'."
	userPrompt := fmt.Sprintf("Prompt 1: %s\nPrompt 2: %s", prompt1, prompt2)

	reqBody := VerificationRequest{
		Model: p.verificationModel,
		Messages: []Message{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: userPrompt},
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.openai.com/v1/chat/completions", bytes.NewBuffer(jsonBody))
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
		return false, fmt.Errorf("OpenAI API error: %s", string(body))
	}

	var verResp VerificationResponse
	if err := json.NewDecoder(resp.Body).Decode(&verResp); err != nil {
		return false, err
	}

	if len(verResp.Choices) == 0 {
		return false, fmt.Errorf("no choices returned")
	}

	content := verResp.Choices[0].Message.Content
	return content == "YES", nil
}
