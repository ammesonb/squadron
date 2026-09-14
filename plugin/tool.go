package plugin

import (
	"context"
	"encoding/json"

	"squadron/aitools"
)

type PluginTool struct {
	provider ToolProvider
	info     *ToolInfo
}

func NewPluginTool(provider ToolProvider, info *ToolInfo) *PluginTool {
	return &PluginTool{
		provider: provider,
		info:     info,
	}
}

func (t *PluginTool) ToolName() string {
	return t.info.Name
}

func (t *PluginTool) ToolDescription() string {
	return t.info.Description
}

func (t *PluginTool) ToolPayloadSchema() aitools.Schema {
	return t.info.Schema
}

func (t *PluginTool) ToolOutputSchema() json.RawMessage {
	return t.info.OutputSchema
}

func (t *PluginTool) Call(ctx context.Context, params string) string {
	result, err := t.provider.Call(ctx, t.info.Name, params)
	if err != nil {
		return "error: " + err.Error()
	}
	return result
}

// CallMedia preserves image data returned by legacy string-based plugins as
// typed media. Plugin authors do not need a second transport contract: data
// URLs and recognized base64 image fields are promoted at this adapter
// boundary, before large-result interception can truncate them.
func (t *PluginTool) CallMedia(ctx context.Context, params string) (string, []aitools.MediaBlock) {
	result := t.Call(ctx, params)
	if len(result) >= len("error: ") && result[:len("error: ")] == "error: " {
		return result, nil
	}

	extracted := aitools.ExtractImages(result)
	media := make([]aitools.MediaBlock, 0, len(extracted.Images))
	for _, image := range extracted.Images {
		media = append(media, aitools.MediaBlock{
			Kind:      aitools.MediaKindImage,
			MediaType: image.MediaType,
			Data:      image.Data,
		})
	}
	return extracted.RemainingText, media
}
