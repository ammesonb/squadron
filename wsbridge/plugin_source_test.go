package wsbridge

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/mlund01/squadron-wire/protocol"
	"squadron/config"
)

func TestLocalPluginSourceHandlers(t *testing.T) {
	root := t.TempDir()
	writeTestPluginFile(t, root, "go.mod", "module example.test/plugin\n")
	writeTestPluginFile(t, root, "main.go", "package main\n")
	writeTestPluginFile(t, root, "internal/tool.go", "package internal\n")
	writeTestPluginFile(t, root, ".git/config", "hidden")
	writeTestPluginFile(t, root, ".env", "TOKEN=secret")
	writeTestPluginFile(t, root, "node_modules/module.js", "ignored")

	client := NewClient(&config.Config{Plugins: []config.Plugin{{Name: "demo", Source: root, Version: "local"}}}, true, "", "", nil, "test")
	defer client.Close()

	listRequest, _ := protocol.NewRequest(typeListLocalPluginFiles, localPluginRequest{PluginName: "demo"})
	listResponse, err := client.handleListLocalPluginFiles(listRequest)
	if err != nil {
		t.Fatal(err)
	}
	var listResult struct {
		Files []localPluginFile `json:"files"`
	}
	if err := protocol.DecodePayload(listResponse, &listResult); err != nil {
		t.Fatal(err)
	}
	got := make([]string, 0, len(listResult.Files))
	for _, file := range listResult.Files {
		got = append(got, file.Path)
	}
	want := []string{"go.mod", "internal/tool.go", "main.go"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("files = %v, want %v", got, want)
	}

	getRequest, _ := protocol.NewRequest(typeGetLocalPluginFile, localPluginRequest{PluginName: "demo", Path: "internal/tool.go"})
	getResponse, err := client.handleGetLocalPluginFile(getRequest)
	if err != nil {
		t.Fatal(err)
	}
	var content localPluginFileContent
	if err := protocol.DecodePayload(getResponse, &content); err != nil {
		t.Fatal(err)
	}
	if content.Content != "package internal\n" || content.Path != "internal/tool.go" {
		t.Fatalf("content = %#v", content)
	}
}

func TestLocalPluginSourceRejectsUnconfiguredAndEscapingPaths(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "escape.txt")); err != nil {
		t.Fatal(err)
	}
	client := NewClient(&config.Config{Plugins: []config.Plugin{
		{Name: "local", Source: root, Version: "local"},
		{Name: "remote", Source: "github.com/example/plugin", Version: "v1.0.0"},
	}}, true, "", "", nil, "test")
	defer client.Close()

	for _, request := range []localPluginRequest{
		{PluginName: "missing", Path: "main.go"},
		{PluginName: "remote", Path: "main.go"},
		{PluginName: "local", Path: "../secret.txt"},
		{PluginName: "local", Path: ".env"},
		{PluginName: "local", Path: "escape.txt"},
	} {
		envelope, _ := protocol.NewRequest(typeGetLocalPluginFile, request)
		if _, err := client.handleGetLocalPluginFile(envelope); err == nil {
			t.Fatalf("request %#v unexpectedly succeeded", request)
		}
	}
}

func writeTestPluginFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}
