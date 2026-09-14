package config

import (
	"crypto/sha256"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

// ConfigSource identifies an exact, unevaluated block in the loaded config.
// FileRevision hashes the full original file so a future diff/apply workflow
// can reject a stale base. Content deliberately excludes unrelated blocks.
type ConfigSource struct {
	Path         string `json:"path"`
	StartLine    int    `json:"startLine"`
	EndLine      int    `json:"endLine"`
	Content      string `json:"content"`
	FileRevision string `json:"fileRevision"`
	rangeInFile  hcl.Range
}

func sourceForBlock(block *hcl.Block) *ConfigSource {
	body, ok := block.Body.(*hclsyntax.Body)
	if !ok {
		return nil
	}
	rng := hcl.RangeBetween(block.DefRange, body.SrcRange)
	return &ConfigSource{rangeInFile: rng}
}

// Populate from the same parser bytes used for evaluation, never reread disk:
// edits made after loading must not masquerade as the runner's active config.
func populateConfigSources(root string, cfg *Config, files map[string]*hcl.File) {
	populate := func(source **ConfigSource) {
		if *source == nil {
			return
		}
		current := *source
		rng := current.rangeInFile
		file := files[rng.Filename]
		absolute, err := filepath.Abs(rng.Filename)
		if err != nil || file == nil {
			*source = nil
			return
		}
		relative, err := filepath.Rel(root, absolute)
		if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) || filepath.IsAbs(relative) {
			*source = nil
			return
		}
		if rng.Start.Byte < 0 || rng.End.Byte > len(file.Bytes) || rng.End.Byte <= rng.Start.Byte {
			*source = nil
			return
		}
		current.Path = filepath.ToSlash(relative)
		current.StartLine, current.EndLine = rng.Start.Line, rng.End.Line
		current.Content = string(file.Bytes[rng.Start.Byte:rng.End.Byte])
		current.FileRevision = fmt.Sprintf("%x", sha256.Sum256(file.Bytes))
	}
	for i := range cfg.Agents {
		populate(&cfg.Agents[i].Source)
	}
	for i := range cfg.Missions {
		populate(&cfg.Missions[i].Source)
		for j := range cfg.Missions[i].LocalAgents {
			populate(&cfg.Missions[i].LocalAgents[j].Source)
		}
	}
}
