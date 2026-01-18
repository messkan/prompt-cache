package semantic

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	internalhttp "github.com/messkan/PromptCache/internal/http"
)

// Mock HTTP RoundTripper for testing
type MockRoundTripper struct {
	Response *http.Response
	Err      error
}

func (m *MockRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if m.Err != nil {
		return nil, m.Err
	}
	return m.Response, nil
}

// Helper to create a mock retryable client
func newMockClient(response *http.Response) *internalhttp.RetryableClient {
	return internalhttp.NewRetryableClientWithHTTPClient(&http.Client{
		Transport: &MockRoundTripper{
			Response: response,
		},
	})
}

func TestMistralProvider_Embed(t *testing.T) {
	mockResponse := MistralEmbeddingResponse{
		Data: []struct {
			Embedding []float64 `json:"embedding"`
		}{
			{Embedding: []float64{0.1, 0.2, 0.3, 0.4}},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &MistralProvider{
		apiKey:      "test-key",
		embedModel:  "mistral-embed",
		verifyModel: "mistral-small-latest",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	embedding, err := provider.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embedding) != 4 {
		t.Errorf("Expected 4 dimensions, got %d", len(embedding))
	}

	if embedding[0] != 0.1 {
		t.Errorf("Expected first value 0.1, got %f", embedding[0])
	}
}

func TestMistralProvider_CheckSimilarity_Match(t *testing.T) {
	mockResponse := MistralChatResponse{
		Choices: []struct {
			Message MistralMessage `json:"message"`
		}{
			{Message: MistralMessage{Role: "assistant", Content: "YES"}},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &MistralProvider{
		apiKey:      "test-key",
		embedModel:  "mistral-embed",
		verifyModel: "mistral-small-latest",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	match, err := provider.CheckSimilarity(context.Background(), "prompt1", "prompt2")
	if err != nil {
		t.Fatalf("CheckSimilarity failed: %v", err)
	}

	if !match {
		t.Errorf("Expected match=true, got false")
	}
}

func TestMistralProvider_CheckSimilarity_NoMatch(t *testing.T) {
	mockResponse := MistralChatResponse{
		Choices: []struct {
			Message MistralMessage `json:"message"`
		}{
			{Message: MistralMessage{Role: "assistant", Content: "NO"}},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &MistralProvider{
		apiKey:      "test-key",
		embedModel:  "mistral-embed",
		verifyModel: "mistral-small-latest",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	match, err := provider.CheckSimilarity(context.Background(), "prompt1", "prompt2")
	if err != nil {
		t.Fatalf("CheckSimilarity failed: %v", err)
	}

	if match {
		t.Errorf("Expected match=false, got true")
	}
}

func TestClaudeProvider_Embed(t *testing.T) {
	mockResponse := VoyageEmbeddingResponse{
		Data: []struct {
			Embedding []float64 `json:"embedding"`
		}{
			{Embedding: []float64{0.5, 0.6, 0.7, 0.8}},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	// Set environment variable for test
	t.Setenv("VOYAGE_API_KEY", "test-voyage-key")

	provider := &ClaudeProvider{
		apiKey:      "test-key",
		chatModel:   "claude-3-opus-20240229",
		verifyModel: "claude-3-haiku-20240307",
		voyageModel: "voyage-3",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	embedding, err := provider.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embedding) != 4 {
		t.Errorf("Expected 4 dimensions, got %d", len(embedding))
	}

	if embedding[0] != 0.5 {
		t.Errorf("Expected first value 0.5, got %f", embedding[0])
	}
}

func TestClaudeProvider_Embed_NoVoyageKey(t *testing.T) {
	// Ensure VOYAGE_API_KEY is not set
	t.Setenv("VOYAGE_API_KEY", "")
	
	provider := &ClaudeProvider{
		apiKey:      "test-key",
		chatModel:   "claude-3-opus-20240229",
		verifyModel: "claude-3-haiku-20240307",
		voyageModel: "voyage-3",
		client:      newMockClient(&http.Response{StatusCode: http.StatusOK}),
	}

	_, err := provider.Embed(context.Background(), "test text")
	if err == nil {
		t.Fatal("Expected error when VOYAGE_API_KEY is not set")
	}

	if !strings.Contains(err.Error(), "VOYAGE_API_KEY") {
		t.Errorf("Expected error message about VOYAGE_API_KEY, got: %v", err)
	}
}

func TestClaudeProvider_CheckSimilarity_Match(t *testing.T) {
	mockResponse := ClaudeChatResponse{
		Content: []struct {
			Text string `json:"text"`
		}{
			{Text: "YES"},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &ClaudeProvider{
		apiKey:      "test-key",
		chatModel:   "claude-3-opus-20240229",
		verifyModel: "claude-3-haiku-20240307",
		voyageModel: "voyage-3",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	match, err := provider.CheckSimilarity(context.Background(), "prompt1", "prompt2")
	if err != nil {
		t.Fatalf("CheckSimilarity failed: %v", err)
	}

	if !match {
		t.Errorf("Expected match=true, got false")
	}
}

func TestClaudeProvider_CheckSimilarity_NoMatch(t *testing.T) {
	mockResponse := ClaudeChatResponse{
		Content: []struct {
			Text string `json:"text"`
		}{
			{Text: "NO"},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &ClaudeProvider{
		apiKey:      "test-key",
		chatModel:   "claude-3-opus-20240229",
		verifyModel: "claude-3-haiku-20240307",
		voyageModel: "voyage-3",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	match, err := provider.CheckSimilarity(context.Background(), "prompt1", "prompt2")
	if err != nil {
		t.Fatalf("CheckSimilarity failed: %v", err)
	}

	if match {
		t.Errorf("Expected match=false, got true")
	}
}

func TestOpenAIProvider_Embed(t *testing.T) {
	mockResponse := EmbeddingResponse{
		Data: []struct {
			Embedding []float64 `json:"embedding"`
		}{
			{Embedding: []float64{0.9, 0.8, 0.7, 0.6}},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &OpenAIProvider{
		apiKey:     "test-key",
		embedModel: "text-embedding-3-small",
		chatModel:  "gpt-4o-mini",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	embedding, err := provider.Embed(context.Background(), "test text")
	if err != nil {
		t.Fatalf("Embed failed: %v", err)
	}

	if len(embedding) != 4 {
		t.Errorf("Expected 4 dimensions, got %d", len(embedding))
	}

	if embedding[0] != 0.9 {
		t.Errorf("Expected first value 0.9, got %f", embedding[0])
	}
}

func TestOpenAIProvider_CheckSimilarity_Match(t *testing.T) {
	mockResponse := VerificationResponse{
		Choices: []struct {
			Message Message `json:"message"`
		}{
			{Message: Message{Role: "assistant", Content: "YES"}},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &OpenAIProvider{
		apiKey:     "test-key",
		embedModel: "text-embedding-3-small",
		chatModel:  "gpt-4o-mini",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	match, err := provider.CheckSimilarity(context.Background(), "prompt1", "prompt2")
	if err != nil {
		t.Fatalf("CheckSimilarity failed: %v", err)
	}

	if !match {
		t.Errorf("Expected match=true, got false")
	}
}

func TestOpenAIProvider_CheckSimilarity_NoMatch(t *testing.T) {
	mockResponse := VerificationResponse{
		Choices: []struct {
			Message Message `json:"message"`
		}{
			{Message: Message{Role: "assistant", Content: "NO"}},
		},
	}

	jsonBytes, _ := json.Marshal(mockResponse)

	provider := &OpenAIProvider{
		apiKey:     "test-key",
		embedModel: "text-embedding-3-small",
		chatModel:  "gpt-4o-mini",
		client: newMockClient(&http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
		}),
	}

	match, err := provider.CheckSimilarity(context.Background(), "prompt1", "prompt2")
	if err != nil {
		t.Fatalf("CheckSimilarity failed: %v", err)
	}

	if match {
		t.Errorf("Expected match=false, got true")
	}
}

func TestCheckSimilarity_CaseInsensitive(t *testing.T) {
	// Test that YES matching is case-insensitive
	testCases := []struct {
		content string
		want    bool
	}{
		{"YES", true},
		{"Yes", true},
		{"yes", true},
		{" YES ", true},
		{"NO", false},
		{"No", false},
		{"no", false},
		{"Maybe", false},
	}

	for _, tc := range testCases {
		t.Run(tc.content, func(t *testing.T) {
			mockResponse := VerificationResponse{
				Choices: []struct {
					Message Message `json:"message"`
				}{
					{Message: Message{Role: "assistant", Content: tc.content}},
				},
			}

			jsonBytes, _ := json.Marshal(mockResponse)

			provider := &OpenAIProvider{
				apiKey:     "test-key",
				embedModel: "text-embedding-3-small",
				chatModel:  "gpt-4o-mini",
				client: newMockClient(&http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(jsonBytes)),
				}),
			}

			match, err := provider.CheckSimilarity(context.Background(), "prompt1", "prompt2")
			if err != nil {
				t.Fatalf("CheckSimilarity failed: %v", err)
			}

			if match != tc.want {
				t.Errorf("For content %q: got match=%v, want %v", tc.content, match, tc.want)
			}
		})
	}
}
