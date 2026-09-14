package plugin

import (
	"context"
	"testing"
)

type mediaTestProvider struct {
	result string
	err    error
}

func (p *mediaTestProvider) Configure(map[string]string) error { return nil }
func (p *mediaTestProvider) Call(context.Context, string, string) (string, error) {
	return p.result, p.err
}
func (p *mediaTestProvider) GetToolInfo(string) (*ToolInfo, error) { return nil, nil }
func (p *mediaTestProvider) ListTools() ([]*ToolInfo, error)       { return nil, nil }

func TestPluginToolCallMediaPromotesDataURL(t *testing.T) {
	provider := &mediaTestProvider{result: "captured data:image/png;base64,iVBORw0KGgoAAA=="}
	tool := NewPluginTool(provider, &ToolInfo{Name: "screenshot"})

	text, media := tool.CallMedia(context.Background(), `{}`)
	if text != "captured [image]" {
		t.Fatalf("text = %q, want image placeholder", text)
	}
	if len(media) != 1 || media[0].MediaType != "image/png" || media[0].Data != "iVBORw0KGgoAAA==" {
		t.Fatalf("unexpected media: %#v", media)
	}
}
