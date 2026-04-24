package semantic

import (
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPipeSSEAndBuffer_AssemblesContent(t *testing.T) {
	stream := strings.Join([]string{
		`data: {"id":"x","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"role":"assistant"},"finish_reason":""}]}`,
		``,
		`data: {"id":"x","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"Hello "},"finish_reason":""}]}`,
		``,
		`data: {"id":"x","object":"chat.completion.chunk","created":1,"model":"m","choices":[{"index":0,"delta":{"content":"world"},"finish_reason":""}]}`,
		``,
		`data: [DONE]`,
		``,
	}, "\n")

	w := httptest.NewRecorder()
	buffered, err := pipeSSEAndBuffer(strings.NewReader(stream), w)
	if err != nil {
		t.Fatalf("pipeSSEAndBuffer: %v", err)
	}

	var resp OpenAIChatCompletionResponse
	if err := json.Unmarshal(buffered, &resp); err != nil {
		t.Fatalf("unmarshal buffered: %v", err)
	}
	if resp.ID != "x" || resp.Model != "m" || resp.Created != 1 {
		t.Errorf("unexpected metadata: %+v", resp)
	}
	if len(resp.Choices) != 1 {
		t.Fatalf("expected 1 choice, got %d", len(resp.Choices))
	}
	if resp.Choices[0].Message.Content != "Hello world" {
		t.Errorf("expected 'Hello world', got %q", resp.Choices[0].Message.Content)
	}
	if !strings.Contains(w.Body.String(), "Hello ") || !strings.Contains(w.Body.String(), "[DONE]") {
		t.Errorf("client stream missing content: %s", w.Body.String())
	}
}

func TestPipeSSEAndBuffer_HandlesMalformedChunks(t *testing.T) {
	stream := "data: not-json\n\ndata: [DONE]\n\n"
	w := httptest.NewRecorder()
	buffered, err := pipeSSEAndBuffer(strings.NewReader(stream), w)
	if err != nil {
		t.Fatalf("should not error on malformed: %v", err)
	}
	var resp OpenAIChatCompletionResponse
	if err := json.Unmarshal(buffered, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Choices[0].Message.Content != "" {
		t.Errorf("expected empty content, got %q", resp.Choices[0].Message.Content)
	}
}
