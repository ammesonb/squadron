package config

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
)

func TestAgentSourcePreservesExactBlock(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "agents", "reviewers.hcl")
	raw := "# unicode before block: café 🛠\n" + `model "private" {
  api_key = "do-not-expose-unrelated-block"
}

agent "reviewer" {
  model = models.openai.gpt_5 # preserve expression and comment
  personality = <<-TEXT
    Keep {braces}, quotes, and café intact.
  TEXT
  skill "audit" {
    description = "nested block"
    instructions = "Review carefully"
  }
}

agent "other" { personality = "not this agent" }
`
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(raw), path)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	body, _, diags := file.Body.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "agent", LabelNames: []string{"name"}}}})
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	cfg := &Config{Agents: []Agent{{Name: "reviewer", Source: sourceForBlock(body.Blocks[0])}}}
	populateConfigSources(root, cfg, parser.Files())
	source := cfg.Agents[0].Source
	if source == nil || source.Path != "agents/reviewers.hcl" || source.StartLine != 6 || source.EndLine != 15 {
		t.Fatalf("unexpected source: %#v", source)
	}
	want := raw[strings.Index(raw, `agent "reviewer"`):strings.Index(raw, "\n\nagent \"other\"")]
	if source.Content != want {
		t.Fatalf("source was altered:\n%q\nwant:\n%q", source.Content, want)
	}
	if source.FileRevision != fmt.Sprintf("%x", sha256.Sum256([]byte(raw))) {
		t.Fatal("revision is not the hash of the original file bytes")
	}
	serialized, err := json.Marshal(cfg.Agents[0])
	if err != nil || strings.Contains(string(serialized), "source") || strings.Contains(string(serialized), "preserve expression") {
		t.Fatal("source metadata leaked into runtime agent serialization")
	}
	// No file was written. Source must come from the parser snapshot, not disk.
}

func TestMissionSourcePreservesExactBlock(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "missions", "release.hcl")
	raw := `model "private" {
  api_key = "do-not-expose-unrelated-block"
}

mission "release" {
  # Preserve mission comments and expressions.
  agents = [agents.reviewer]
  directive = "Ship café safely"

  task "review" {
    objective = "Review {everything}"
  }
}

mission "other" { agents = [] }
`
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(raw), path)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	body, _, diags := file.Body.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "mission", LabelNames: []string{"name"}}}})
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	cfg := &Config{Missions: []Mission{{Name: "release", Source: sourceForBlock(body.Blocks[0])}}}
	populateConfigSources(root, cfg, parser.Files())
	source := cfg.Missions[0].Source
	if source == nil || source.Path != "missions/release.hcl" || source.StartLine != 5 || source.EndLine != 13 {
		t.Fatalf("unexpected source: %#v", source)
	}
	want := raw[strings.Index(raw, `mission "release"`):strings.Index(raw, "\n\nmission \"other\"")]
	if source.Content != want {
		t.Fatalf("source was altered:\n%q\nwant:\n%q", source.Content, want)
	}
	serialized, err := json.Marshal(cfg.Missions[0])
	if err != nil || strings.Contains(string(serialized), "source") || strings.Contains(string(serialized), "Preserve mission comments") {
		t.Fatal("source metadata leaked into runtime mission serialization")
	}
}

func TestAgentSourceKeepsMissionScopesSeparate(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "missions.hcl")
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL([]byte(`mission "release" {
  agent "reviewer" {
    personality = "release reviewer"
  }
}
mission "audit" {
  agent "reviewer" {
    personality = "audit reviewer"
  }
}
`), path)
	if diags.HasErrors() {
		t.Fatal(diags)
	}
	cfg := &Config{}
	for _, mission := range file.Body.(*hclsyntax.Body).Blocks {
		content, _, diags := mission.Body.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "agent", LabelNames: []string{"name"}}}})
		if diags.HasErrors() {
			t.Fatal(diags)
		}
		cfg.Missions = append(cfg.Missions, Mission{Name: mission.Labels[0], LocalAgents: []Agent{{Name: "reviewer", Source: sourceForBlock(content.Blocks[0])}}})
	}
	populateConfigSources(root, cfg, parser.Files())
	for i, mission := range cfg.Missions {
		source := mission.LocalAgents[0].Source
		if source == nil || !strings.Contains(source.Content, mission.Name+" reviewer") || source.StartLine != 2+5*i {
			t.Fatalf("wrong source for %s: %#v", mission.Name, source)
		}
	}
}

func TestAgentSourceUnavailableOutsideRoot(t *testing.T) {
	root := t.TempDir()
	parser := hclparse.NewParser()
	file, _ := parser.ParseHCL([]byte(`agent "reviewer" { personality = "hidden" }`), filepath.Join(root, "..", "outside.hcl"))
	body, _, _ := file.Body.PartialContent(&hcl.BodySchema{Blocks: []hcl.BlockHeaderSchema{{Type: "agent", LabelNames: []string{"name"}}}})
	cfg := &Config{Agents: []Agent{{Name: "reviewer", Source: sourceForBlock(body.Blocks[0])}, {Name: "programmatic"}}}
	populateConfigSources(root, cfg, parser.Files())
	for _, a := range cfg.Agents {
		if a.Source != nil {
			t.Fatal("unexpected source for an agent without an in-workspace definition")
		}
	}
}
