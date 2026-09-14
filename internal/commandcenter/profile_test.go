package commandcenter

import (
	"errors"
	"os"
	"testing"
)

func TestSaveAndLoad(t *testing.T) {
	home := t.TempDir()
	want := Profile{
		URL:              "wss://command.example.com/ws",
		WorkerCredential: "credential",
	}
	if err := Save(home, want); err != nil {
		t.Fatalf("save profile: %v", err)
	}

	got, err := Load(home)
	if err != nil {
		t.Fatalf("load profile: %v", err)
	}
	if got != want {
		t.Fatalf("profile = %#v, want %#v", got, want)
	}

	info, err := os.Stat(ProfilePath(home))
	if err != nil {
		t.Fatalf("stat profile: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Fatalf("profile permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestLoadReturnsNotConfigured(t *testing.T) {
	_, err := Load(t.TempDir())
	if !errors.Is(err, ErrNotConfigured) {
		t.Fatalf("load error = %v, want ErrNotConfigured", err)
	}
}
