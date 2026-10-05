package domain

import (
	"encoding/json"
	"testing"
)

func TestChatMessage_MarshalsImagePartsAsInlineDataURLs(t *testing.T) {
	message := ChatMessage{
		Role: "user",
		Content: MessageContent{Parts: []ContentPart{
			TextPart{Type: ContentPartTypeText, Text: "what is in this image?"},
			ImagePart{
				Type:     ContentPartTypeImage,
				ImageURL: ImageURL{URL: "data:image/png;base64,AAAA"},
			},
		}},
	}

	encoded, err := json.Marshal(message)
	if err != nil {
		t.Fatalf("cannot marshal the message: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("cannot decode the marshalled message: %v", err)
	}

	parts, isList := decoded["content"].([]any)
	if !isList {
		t.Fatalf("expected content to be a parts array, got %T", decoded["content"])
	}
	if len(parts) != 2 {
		t.Fatalf("expected two parts, got %d", len(parts))
	}

	textPart, isTextMap := parts[0].(map[string]any)
	if !isTextMap {
		t.Fatalf("expected the first part to be an object, got %T", parts[0])
	}
	if textPart["type"] != "text" || textPart["text"] != "what is in this image?" {
		t.Errorf("unexpected text part: %+v", textPart)
	}

	imagePart, isImageMap := parts[1].(map[string]any)
	if !isImageMap {
		t.Fatalf("expected the second part to be an object, got %T", parts[1])
	}
	if imagePart["type"] != "image_url" {
		t.Errorf("expected the image part type image_url, got %v", imagePart["type"])
	}

	imageURL, isImageURLMap := imagePart["image_url"].(map[string]any)
	if !isImageURLMap {
		t.Fatalf("expected image_url to be an object, got %T", imagePart["image_url"])
	}
	if imageURL["url"] != "data:image/png;base64,AAAA" {
		t.Errorf("expected the inline data url, got %v", imageURL["url"])
	}
}

func TestChatMessage_KeepsMarshallingPlainTextAsAString(t *testing.T) {
	encoded, err := json.Marshal(ChatMessage{Role: "user", Content: TextMessage("plain question")})
	if err != nil {
		t.Fatalf("cannot marshal the message: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("cannot decode the marshalled message: %v", err)
	}
	if decoded["content"] != "plain question" {
		t.Errorf("expected a plain string content, got %T", decoded["content"])
	}
}
