package llm

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// This exercises the SDK's actual HTTP serialization rather than only our
// intermediate request structs. A tool-produced image must reach the OpenAI
// Responses API as input_image content on the turn after its tool result.
func TestOpenAIProviderSerializesToolImageInRequest(t *testing.T) {
	body := make(chan string, 1)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		body <- string(data)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":"resp_media","object":"response","created_at":0,"status":"completed","output":[],"error":null,"incomplete_details":null,"instructions":null,"parallel_tool_calls":true,"tool_choice":"auto","tools":[],"model":"gpt-5","metadata":{},"temperature":1,"top_p":1,"usage":{"input_tokens":1,"output_tokens":1,"total_tokens":2,"input_tokens_details":{"cached_tokens":0},"output_tokens_details":{"reasoning_tokens":0}}`))
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-key", server.URL)
	_, _ = provider.Chat(context.Background(), &ChatRequest{
		Model: "gpt-5",
		Messages: []Message{
			{
				Role: RoleAssistant,
				Parts: []ContentBlock{{
					Type:    ContentTypeToolUse,
					ToolUse: &ToolUseBlock{ID: "call_img", Name: "screenshot", Input: json.RawMessage(`{}`)},
				}},
			},
			{
				Role: RoleUser,
				Parts: []ContentBlock{
					{Type: ContentTypeToolResult, ToolResult: &ToolResultBlock{ToolUseID: "call_img", Content: "[image]"}},
					{Type: ContentTypeImage, ImageData: &ImageBlock{MediaType: "image/jpeg", Data: "/9j/AAAA"}},
				},
			},
		},
	})
	requestBody := <-body
	for _, required := range []string{
		`"type":"function_call_output"`,
		`"call_id":"call_img"`,
		`"type":"input_image"`,
		`"image_url":"data:image/jpeg;base64,/9j/AAAA"`,
	} {
		if !strings.Contains(requestBody, required) {
			t.Fatalf("serialized request is missing %s:\n%s", required, requestBody)
		}
	}
}
