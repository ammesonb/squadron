package wsbridge

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/mlund01/squadron-wire/protocol"
	"squadron/config"
)

const (
	typeListLocalPluginFiles       protocol.MessageType = "list_local_plugin_files"
	typeListLocalPluginFilesResult protocol.MessageType = "list_local_plugin_files_result"
	typeGetLocalPluginFile         protocol.MessageType = "get_local_plugin_file"
	typeGetLocalPluginFileResult   protocol.MessageType = "get_local_plugin_file_result"
	maxPluginSourceFileSize                             = 1 << 20
)

type localPluginRequest struct {
	PluginName string `json:"pluginName"`
	Path       string `json:"path,omitempty"`
}

type localPluginFile struct {
	Path string `json:"path"`
	Size int64  `json:"size"`
}

type localPluginFileContent struct {
	PluginName string `json:"pluginName"`
	Path       string `json:"path"`
	Content    string `json:"content"`
	Size       int64  `json:"size"`
}

func (c *Client) handleListLocalPluginFiles(env *protocol.Envelope) (*protocol.Envelope, error) {
	var request localPluginRequest
	if err := protocol.DecodePayload(env, &request); err != nil {
		return nil, fmt.Errorf("decode local plugin request: %w", err)
	}

	_, root, err := c.localPluginRoot(request.PluginName)
	if err != nil {
		return nil, err
	}

	files := make([]localPluginFile, 0)
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if path != root && shouldSkipPluginDirectory(entry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasPrefix(entry.Name(), ".") || !isPreviewablePluginFile(entry.Name()) {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || info.Size() > maxPluginSourceFileSize {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		files = append(files, localPluginFile{Path: filepath.ToSlash(relative), Size: info.Size()})
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list local plugin files: %w", err)
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })

	return protocol.NewResponse(env.RequestID, typeListLocalPluginFilesResult, map[string]any{
		"pluginName": request.PluginName,
		"files":      files,
	})
}

func (c *Client) handleGetLocalPluginFile(env *protocol.Envelope) (*protocol.Envelope, error) {
	var request localPluginRequest
	if err := protocol.DecodePayload(env, &request); err != nil {
		return nil, fmt.Errorf("decode local plugin file request: %w", err)
	}
	if err := validatePluginRelativePath(request.Path); err != nil {
		return nil, err
	}
	if !isPreviewablePluginPath(request.Path) {
		return nil, fmt.Errorf("local plugin path is not a previewable source file")
	}

	_, root, err := c.localPluginRoot(request.PluginName)
	if err != nil {
		return nil, err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, filepath.FromSlash(request.Path)))
	if err != nil {
		return nil, fmt.Errorf("resolve local plugin file: %w", err)
	}
	if err := ensureWithinRoot(root, resolved); err != nil {
		return nil, err
	}
	info, err := os.Stat(resolved)
	if err != nil {
		return nil, fmt.Errorf("stat local plugin file: %w", err)
	}
	if !info.Mode().IsRegular() {
		return nil, fmt.Errorf("local plugin path is not a regular file")
	}
	if info.Size() > maxPluginSourceFileSize {
		return nil, fmt.Errorf("local plugin file exceeds the 1 MiB preview limit")
	}
	content, err := os.ReadFile(resolved)
	if err != nil {
		return nil, fmt.Errorf("read local plugin file: %w", err)
	}
	if !utf8.Valid(content) || bytes.IndexByte(content, 0) >= 0 {
		return nil, fmt.Errorf("binary local plugin files cannot be previewed")
	}

	return protocol.NewResponse(env.RequestID, typeGetLocalPluginFileResult, localPluginFileContent{
		PluginName: request.PluginName,
		Path:       filepath.ToSlash(filepath.Clean(request.Path)),
		Content:    string(content),
		Size:       info.Size(),
	})
}

func (c *Client) localPluginRoot(name string) (*config.Plugin, string, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, "", fmt.Errorf("plugin name is required")
	}
	cfg := c.getConfig()
	if cfg == nil {
		return nil, "", fmt.Errorf("workspace configuration is unavailable")
	}
	for i := range cfg.Plugins {
		plugin := &cfg.Plugins[i]
		if plugin.Name != name {
			continue
		}
		if !plugin.IsLocal() || !plugin.IsLocalSource() || !filepath.IsAbs(plugin.Source) {
			return nil, "", fmt.Errorf("plugin %q is not a configured local plugin", name)
		}
		root, err := filepath.EvalSymlinks(plugin.Source)
		if err != nil {
			return nil, "", fmt.Errorf("resolve local plugin source: %w", err)
		}
		info, err := os.Stat(root)
		if err != nil {
			return nil, "", fmt.Errorf("stat local plugin source: %w", err)
		}
		if !info.IsDir() {
			return nil, "", fmt.Errorf("local plugin source is not a directory")
		}
		return plugin, root, nil
	}
	return nil, "", fmt.Errorf("local plugin %q is not configured", name)
}

func validatePluginRelativePath(path string) error {
	if path == "" || strings.Contains(path, "\\") || filepath.IsAbs(path) {
		return fmt.Errorf("invalid local plugin file path")
	}
	cleaned := filepath.Clean(filepath.FromSlash(path))
	if cleaned == "." || cleaned == ".." || strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return fmt.Errorf("invalid local plugin file path")
	}
	return nil
}

func ensureWithinRoot(root, path string) error {
	relative, err := filepath.Rel(root, path)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("local plugin file escapes its configured source")
	}
	return nil
}

func shouldSkipPluginDirectory(name string) bool {
	if strings.HasPrefix(name, ".") {
		return true
	}
	switch name {
	case "node_modules", "vendor", "dist", "build", "coverage", "__pycache__":
		return true
	default:
		return false
	}
}

func isPreviewablePluginPath(path string) bool {
	parts := strings.Split(filepath.ToSlash(path), "/")
	for _, part := range parts[:len(parts)-1] {
		if shouldSkipPluginDirectory(part) {
			return false
		}
	}
	name := parts[len(parts)-1]
	return !strings.HasPrefix(name, ".") && isPreviewablePluginFile(name)
}

func isPreviewablePluginFile(name string) bool {
	switch strings.ToLower(name) {
	case "dockerfile", "makefile", "license", "notice":
		return true
	}
	switch strings.ToLower(filepath.Ext(name)) {
	case ".c", ".cc", ".cpp", ".css", ".go", ".h", ".hcl", ".html", ".java", ".js", ".jsx", ".json", ".kt", ".md", ".mod", ".proto", ".py", ".rb", ".rs", ".sh", ".sql", ".sum", ".toml", ".ts", ".tsx", ".txt", ".xml", ".yaml", ".yml":
		return true
	default:
		return false
	}
}
