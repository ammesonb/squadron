package agent

import (
	"squadron/aitools"
	"squadron/llm"
	"squadron/store"
)

// sourcedToolMedia tracks which tool result owns each model-facing media
// block. The LLM protocol only needs the flattened blocks, while persistence
// keeps this association so UIs never have to infer ownership from timing.
type sourcedToolMedia struct {
	toolUseID string
	blocks    []llm.ContentBlock
}

// mediaBlocksToContentBlocks translates provider-neutral tool media into llm
// content blocks. Images and SVG-rasterized PNGs become image blocks; PDFs
// become document blocks. Unknown kinds are dropped.
func mediaBlocksToContentBlocks(media []aitools.MediaBlock) []llm.ContentBlock {
	if len(media) == 0 {
		return nil
	}
	parts := make([]llm.ContentBlock, 0, len(media))
	for _, m := range media {
		switch m.Kind {
		case aitools.MediaKindImage:
			parts = append(parts, llm.ContentBlock{
				Type:      llm.ContentTypeImage,
				ImageData: &llm.ImageBlock{Data: m.Data, MediaType: m.MediaType},
			})
		case aitools.MediaKindDocument:
			parts = append(parts, llm.ContentBlock{
				Type:     llm.ContentTypeDocument,
				Document: &llm.DocumentBlock{Data: m.Data, MediaType: m.MediaType, Filename: m.Filename},
			})
		}
	}
	return parts
}

func appendSourcedToolMedia(groups []sourcedToolMedia, toolUseID string, media []aitools.MediaBlock) ([]sourcedToolMedia, []llm.ContentBlock) {
	blocks := mediaBlocksToContentBlocks(media)
	if len(blocks) == 0 {
		return groups, nil
	}
	return append(groups, sourcedToolMedia{toolUseID: toolUseID, blocks: blocks}), blocks
}

// partsFromToolResultTurn projects a tool-result user turn for storage and
// tags media parts with their producing tool call. Restoring the session
// converts these tagged storage parts back to ordinary image/document blocks.
func partsFromToolResultTurn(message llm.Message, mediaGroups []sourcedToolMedia) []store.MessagePart {
	parts := PartsFromMessage(message)
	mediaPart := len(parts)
	for i, part := range parts {
		if part.Type == string(llm.ContentTypeImage) || part.Type == string(llm.ContentTypeDocument) {
			mediaPart = i
			break
		}
	}

	index := mediaPart
	for _, group := range mediaGroups {
		for range group.blocks {
			if index >= len(parts) {
				return parts
			}
			parts[index].ToolUseID = group.toolUseID
			switch parts[index].Type {
			case string(llm.ContentTypeImage):
				parts[index].Type = store.PartTypeToolImage
			case string(llm.ContentTypeDocument):
				parts[index].Type = store.PartTypeToolDocument
			}
			index++
		}
	}
	return parts
}
