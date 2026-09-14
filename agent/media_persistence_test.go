package agent

import (
	"testing"

	"squadron/aitools"
	"squadron/llm"
	"squadron/store"
)

func TestPartsFromToolResultTurnKeepsToolMediaOwnership(t *testing.T) {
	results := []llm.ToolResultBlock{{ToolUseID: "call-1", Content: "[image]"}}
	media := []aitools.MediaBlock{{Kind: aitools.MediaKindImage, MediaType: "image/png", Data: "QUJD"}}
	groups, blocks := appendSourcedToolMedia(nil, "call-1", media)
	message := llm.Message{Role: llm.RoleUser, Parts: []llm.ContentBlock{
		{Type: llm.ContentTypeToolResult, ToolResult: &results[0]},
		blocks[0],
	}}

	parts := partsFromToolResultTurn(message, groups)
	if len(parts) != 2 {
		t.Fatalf("parts count = %d", len(parts))
	}
	if parts[1].Type != store.PartTypeToolImage || parts[1].ToolUseID != "call-1" {
		t.Fatalf("tool image association not persisted: %#v", parts[1])
	}
	restored, err := contentBlockFromPart(parts[1])
	if err != nil {
		t.Fatal(err)
	}
	if restored.Type != llm.ContentTypeImage || restored.ImageData == nil || restored.ImageData.Data != "QUJD" {
		t.Fatalf("tool image was not restored as model-facing image: %#v", restored)
	}
}
