package main

import (
	"encoding/json"
	"testing"
)

func TestMessageContentUnmarshalString(t *testing.T) {
	var content MessageContent
	if err := json.Unmarshal([]byte(`"hello world"`), &content); err != nil {
		t.Fatalf("unmarshal string content failed: %v", err)
	}

	if content.Text != "hello world" {
		t.Errorf("expected text %q, got %q", "hello world", content.Text)
	}
	if content.Parts != nil {
		t.Errorf("expected no parts for string content, got %d", len(content.Parts))
	}
	if content.CacheText() != "hello world" {
		t.Errorf("expected cache text %q, got %q", "hello world", content.CacheText())
	}
}

func TestMessageContentUnmarshalParts(t *testing.T) {
	raw := `[
		{"type": "text", "text": "describe this"},
		{"type": "image_url", "image_url": {"url": "https://example.com/a.png"}},
		{"type": "video_url", "video_url": {"url": "https://example.com/a.mp4"}}
	]`

	var content MessageContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		t.Fatalf("unmarshal structured content failed: %v", err)
	}

	if len(content.Parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(content.Parts))
	}
	if content.Text != "describe this" {
		t.Errorf("expected text %q, got %q", "describe this", content.Text)
	}

	refs := content.MediaRefs()
	if len(refs) != 2 {
		t.Fatalf("expected 2 media refs, got %d (%v)", len(refs), refs)
	}
	if refs[0] != "image_url=https://example.com/a.png" {
		t.Errorf("unexpected first media ref: %q", refs[0])
	}
	if refs[1] != "video_url=https://example.com/a.mp4" {
		t.Errorf("unexpected second media ref: %q", refs[1])
	}
}

func TestMessageContentMultipleTextPartsAreJoined(t *testing.T) {
	var content MessageContent
	raw := `[{"type": "text", "text": "first"}, {"type": "text", "text": "second"}]`
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		t.Fatalf("unmarshal structured content failed: %v", err)
	}

	if content.Text != "first\nsecond" {
		t.Errorf("expected joined text %q, got %q", "first\nsecond", content.Text)
	}
	if len(content.MediaRefs()) != 0 {
		t.Errorf("expected no media refs for text-only parts")
	}
	if content.CacheText() != content.Text {
		t.Errorf("expected cache text to equal text for text-only parts, got %q", content.CacheText())
	}
}

func TestMessageContentUnmarshalNull(t *testing.T) {
	content := MessageContent{Text: "stale"}
	if err := json.Unmarshal([]byte(`null`), &content); err != nil {
		t.Fatalf("unmarshal null content failed: %v", err)
	}
	if content.Text != "" || content.Parts != nil {
		t.Errorf("expected empty content for null, got %+v", content)
	}
}

func TestMessageContentUnmarshalUnsupportedType(t *testing.T) {
	var content MessageContent
	if err := json.Unmarshal([]byte(`42`), &content); err == nil {
		t.Error("expected an error for a numeric content field")
	}
}

func TestMessageContentMarshalRoundTrip(t *testing.T) {
	cases := []string{
		`"plain text"`,
		`[{"type":"text","text":"look"},{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]`,
	}

	for _, want := range cases {
		var content MessageContent
		if err := json.Unmarshal([]byte(want), &content); err != nil {
			t.Fatalf("unmarshal %s failed: %v", want, err)
		}

		got, err := json.Marshal(content)
		if err != nil {
			t.Fatalf("marshal %s failed: %v", want, err)
		}
		if string(got) != want {
			t.Errorf("expected round trip %s, got %s", want, got)
		}
	}
}

func TestMessageContentCacheTextSeparatesDifferentMedia(t *testing.T) {
	first := mustUnmarshalContent(t, `[{"type":"text","text":"what is this"},{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]`)
	second := mustUnmarshalContent(t, `[{"type":"text","text":"what is this"},{"type":"image_url","image_url":{"url":"https://example.com/b.png"}}]`)
	repeat := mustUnmarshalContent(t, `[{"type":"text","text":"what is this"},{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]`)

	if first.CacheText() == second.CacheText() {
		t.Error("expected different media references to produce different cache text")
	}
	if first.CacheText() != repeat.CacheText() {
		t.Error("expected identical content to produce stable cache text")
	}
	if first.CacheText() == first.Text {
		t.Error("expected cache text to differ from plain text when media is attached")
	}
}

func TestMessageContentCacheTextIgnoresLargeInlineData(t *testing.T) {
	inline := make([]byte, 4096)
	for i := range inline {
		inline[i] = 'A'
	}

	content := mustUnmarshalContent(t, `[{"type":"image_url","image_url":{"url":"data:image/png;base64,`+string(inline)+`"}}]`)

	cacheText := content.CacheText()
	if cacheText == "" {
		t.Fatal("expected media-only content to produce a non-empty cache text")
	}
	if len(cacheText) > 64 {
		t.Errorf("expected a bounded cache text, got %d bytes", len(cacheText))
	}
}

func TestChatCompletionRequestAcceptsStructuredContent(t *testing.T) {
	body := `{
		"model": "test-model",
		"messages": [
			{"role": "system", "content": "be brief"},
			{"role": "user", "content": [
				{"type": "text", "text": "what happens here"},
				{"type": "image_url", "image_url": {"url": "https://example.com/a.png"}},
				{"type": "video_url", "video_url": {"url": "https://example.com/a.mp4"}}
			]}
		],
		"stream": false
	}`

	var req ChatCompletionRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("structured message content was rejected: %v", err)
	}

	if len(req.Messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(req.Messages))
	}
	if req.Messages[0].Content.Text != "be brief" {
		t.Errorf("expected string content to stay intact, got %q", req.Messages[0].Content.Text)
	}

	prompt := lastUserPrompt(req)
	if prompt == "" {
		t.Fatal("expected a non-empty prompt for a multimodal user message")
	}
	if len(req.Messages[1].Content.MediaRefs()) != 2 {
		t.Errorf("expected 2 media refs on the user message, got %d", len(req.Messages[1].Content.MediaRefs()))
	}
}

func TestChatCompletionRequestMediaOnlyMessageHasPrompt(t *testing.T) {
	body := `{"model":"test-model","messages":[{"role":"user","content":[{"type":"image_url","image_url":{"url":"https://example.com/a.png"}}]}]}`

	var req ChatCompletionRequest
	if err := json.Unmarshal([]byte(body), &req); err != nil {
		t.Fatalf("media-only message content was rejected: %v", err)
	}

	if prompt := lastUserPrompt(req); prompt == "" {
		t.Error("expected a media-only user message to yield a prompt instead of being rejected")
	}
}

// lastUserPrompt mirrors the prompt extraction used by the chat completions
// handler.
func lastUserPrompt(req ChatCompletionRequest) string {
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			return req.Messages[i].Content.CacheText()
		}
	}
	return ""
}

func mustUnmarshalContent(t *testing.T, raw string) MessageContent {
	t.Helper()

	var content MessageContent
	if err := json.Unmarshal([]byte(raw), &content); err != nil {
		t.Fatalf("unmarshal content failed: %v", err)
	}
	return content
}
