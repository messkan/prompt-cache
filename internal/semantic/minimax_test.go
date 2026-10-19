package semantic

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	appconfig "github.com/messkan/PromptCache/internal/config"
)

func TestMiniMaxVerifierModelContracts(t *testing.T) {
	for _, tc := range []struct{ model, thinking string }{
		{"MiniMax-M3.1-Flash", "adaptive"},
		{"MiniMax-M3", "disabled"},
		{"MiniMax-M2.7", ""},
		{"custom-model", ""},
	} {
		t.Run(tc.model, func(t *testing.T) {
			client, transport := newCapturingMockClient(&http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"choices":[{"message":{"role":"assistant","content":"YES","reasoning_details":[{"type":"reasoning.text","text":"Compare the intent."}]}}]}`)),
			})
			provider := &MiniMaxProvider{apiKey: "test-key", client: client, verifyModel: tc.model, baseURL: "https://api.minimaxi.com/v1"}
			match, err := provider.CheckSimilarity(context.Background(), "hello", "hi")
			if err != nil || !match {
				t.Fatalf("match=%v, err=%v", match, err)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(transport.Request.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["model"] != tc.model || body["reasoning_split"] != true {
				t.Fatalf("unexpected request: %#v", body)
			}
			if tc.thinking == "" {
				if _, ok := body["thinking"]; ok {
					t.Fatal("custom and legacy models must keep API thinking defaults")
				}
			} else if body["thinking"].(map[string]interface{})["type"] != tc.thinking {
				t.Fatalf("unexpected thinking: %#v", body["thinking"])
			}
			if _, ok := body["reasoning_effort"]; ok {
				t.Fatal("verifier must not inject an effort override")
			}
			if transport.Request.URL.String() != "https://api.minimaxi.com/v1/chat/completions" {
				t.Fatal("regional endpoint was not preserved")
			}
		})
	}
}

func TestMiniMaxStreamingErrorsAreNotCacheable(t *testing.T) {
	for _, tc := range []struct{ contentType, body string }{
		{"application/json", `{"base_resp":{"status_code":1008,"status_msg":"balance error"}}`},
		{"text/event-stream", "data: {\"base_resp\":{\"status_code\":1008,\"status_msg\":\"balance error\"}}\n\n"},
	} {
		t.Run(tc.contentType, func(t *testing.T) {
			provider := &MiniMaxProvider{baseURL: "https://api.minimax.io/v1", client: newMockClient(&http.Response{
				StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{tc.contentType}}, Body: io.NopCloser(strings.NewReader(tc.body)),
			})}
			buffered, _, err := provider.ForwardStreamingChatCompletion(context.Background(), []byte(`{"model":"MiniMax-M3.1-Flash","messages":[]}`), httptest.NewRecorder())
			if err == nil || buffered != nil {
				t.Fatal("API errors must never become cacheable completions")
			}
		})
	}
}

func TestMiniMaxDefaultsAndOverrides(t *testing.T) {
	t.Setenv("MINIMAX_VERIFY_MODEL", "")
	if DefaultProviderConfig().MiniMaxVerifyModel != "MiniMax-M3.1-Flash" || appconfig.Load().MiniMaxVerifyModel != "MiniMax-M3.1-Flash" {
		t.Fatal("default configurations disagree")
	}
	t.Setenv("MINIMAX_VERIFY_MODEL", "custom-model")
	t.Setenv("MINIMAX_BASE_URL", "https://api.minimaxi.com/v1")
	provider := NewMiniMaxProvider()
	if provider.verifyModel != "custom-model" || provider.baseURL != "https://api.minimaxi.com/v1" {
		t.Fatal("explicit overrides were lost")
	}
	t.Setenv("OPENAI_API_KEY", "")
	if _, err := NewMiniMaxProvider().Embed(context.Background(), "test"); err == nil {
		t.Fatal("missing embedding credential must fail locally")
	}
}
