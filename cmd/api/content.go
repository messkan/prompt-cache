package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
)

// ContentURL is the nested object used by non-text content parts to carry a
// remote URL or an inline data reference.
type ContentURL struct {
	URL string `json:"url"`
}

// ContentPart is a single element of a structured message content array. Text
// parts carry Text, while non-text parts such as images or video carry a URL
// object.
type ContentPart struct {
	Type     string      `json:"type"`
	Text     string      `json:"text,omitempty"`
	ImageURL *ContentURL `json:"image_url,omitempty"`
	VideoURL *ContentURL `json:"video_url,omitempty"`
}

// MessageContent holds the content field of a chat message. Clients may send it
// either as a plain string or as an array of typed content parts, so both forms
// are accepted and are re-emitted in the form they arrived in.
type MessageContent struct {
	// Text is the plain string content, or the joined text of every text part
	// when the structured array form is used.
	Text string
	// Parts holds the structured content parts, and is nil for string content.
	Parts []ContentPart
}

// UnmarshalJSON accepts both the plain string form and the content part array
// form of a message content field.
func (mc *MessageContent) UnmarshalJSON(data []byte) error {
	*mc = MessageContent{}

	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || string(trimmed) == "null" {
		return nil
	}

	if trimmed[0] == '"' {
		return json.Unmarshal(trimmed, &mc.Text)
	}

	var parts []ContentPart
	if err := json.Unmarshal(trimmed, &parts); err != nil {
		return err
	}

	texts := make([]string, 0, len(parts))
	for _, part := range parts {
		if part.Text != "" {
			texts = append(texts, part.Text)
		}
	}

	mc.Parts = parts
	mc.Text = strings.Join(texts, "\n")
	return nil
}

// MarshalJSON emits the content in the same form it was received in.
func (mc MessageContent) MarshalJSON() ([]byte, error) {
	if mc.Parts != nil {
		return json.Marshal(mc.Parts)
	}
	return json.Marshal(mc.Text)
}

// MediaRefs returns the references carried by the non-text parts, in the order
// in which those parts appear.
func (mc MessageContent) MediaRefs() []string {
	refs := make([]string, 0, len(mc.Parts))
	for _, part := range mc.Parts {
		for _, ref := range []*ContentURL{part.ImageURL, part.VideoURL} {
			if ref != nil && ref.URL != "" {
				refs = append(refs, part.Type+"="+ref.URL)
			}
		}
	}
	return refs
}

// CacheText returns the text used for cache lookups, cache keys and embeddings.
// Non-text parts are reduced to a short digest so that requests that share the
// same text but reference different media are cached separately, and so that
// large inline data references never reach the embedding provider or the prompt
// store. String content is returned unchanged, which keeps existing cache keys
// stable.
func (mc MessageContent) CacheText() string {
	refs := mc.MediaRefs()
	if len(refs) == 0 {
		return mc.Text
	}

	digest := sha256.Sum256([]byte(strings.Join(refs, "\n")))
	return strings.TrimSpace(mc.Text + " [media:" + hex.EncodeToString(digest[:8]) + "]")
}
