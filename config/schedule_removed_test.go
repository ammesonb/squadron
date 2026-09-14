package config_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"squadron/config"
)

func TestWorkerRejectsLegacyScheduleBlocks(t *testing.T) {
	hcl := fullBaseHCL() + `
mission "legacy_schedule" {
  commander { model = models.anthropic.claude_sonnet_4 }
  agents = [agents.test_agent]
  schedule { cron = "0 9 * * *" }
  task "work" { objective = "Do work" }
}`
	path := filepath.Join(t.TempDir(), "legacy-schedule.hcl")
	if err := os.WriteFile(path, []byte(hcl), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := config.LoadFile(path)
	if err == nil || !strings.Contains(err.Error(), "managed by Command Center") {
		t.Fatalf("LoadFile() error = %v, want product-managed schedule migration error", err)
	}
}
