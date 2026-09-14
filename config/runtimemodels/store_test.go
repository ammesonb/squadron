package runtimemodels

import "testing"

func TestReplaceIsAtomicSnapshot(t *testing.T) {
	Replace(map[string]Connection{"old": {Provider: "openai"}})
	Replace(map[string]Connection{"local": {Provider: "openai_compatible", BaseURL: "http://localhost:11434/v1"}})
	if _, ok := Get("old"); ok {
		t.Fatal("stale connection survived replacement")
	}
	if got, ok := Get("local"); !ok || got.BaseURL == "" {
		t.Fatalf("connection missing: %#v", got)
	}
}
