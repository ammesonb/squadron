// Package commandcenter manages the local connection profile for a Squadron
// worker. It is deliberately separate from project HCL because the worker
// credential is a machine secret, not mission configuration.
package commandcenter

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var ErrNotConfigured = errors.New("command center connection is not configured")

const profileFileName = "command-center.json"

type Profile struct {
	URL              string `json:"url"`
	WorkerCredential string `json:"workerCredential"`
}

func ProfilePath(home string) string {
	return filepath.Join(home, profileFileName)
}

func Load(home string) (Profile, error) {
	data, err := os.ReadFile(ProfilePath(home))
	if errors.Is(err, os.ErrNotExist) {
		return Profile{}, ErrNotConfigured
	}
	if err != nil {
		return Profile{}, fmt.Errorf("read command center profile: %w", err)
	}

	var profile Profile
	if err := json.Unmarshal(data, &profile); err != nil {
		return Profile{}, fmt.Errorf("decode command center profile: %w", err)
	}
	if err := profile.Validate(); err != nil {
		return Profile{}, err
	}
	return profile, nil
}

func (p Profile) Validate() error {
	if strings.TrimSpace(p.URL) == "" {
		return fmt.Errorf("command center profile: URL is required")
	}
	if strings.TrimSpace(p.WorkerCredential) == "" {
		return fmt.Errorf("command center profile: worker credential is required")
	}
	return nil
}

func Save(home string, profile Profile) error {
	if err := profile.Validate(); err != nil {
		return err
	}
	if err := os.MkdirAll(home, 0700); err != nil {
		return fmt.Errorf("create Squadron home: %w", err)
	}

	data, err := json.Marshal(profile)
	if err != nil {
		return fmt.Errorf("encode command center profile: %w", err)
	}
	temporary, err := os.CreateTemp(home, profileFileName+"-*")
	if err != nil {
		return fmt.Errorf("create command center profile: %w", err)
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0600); err != nil {
		temporary.Close()
		return fmt.Errorf("secure command center profile: %w", err)
	}
	if _, err := temporary.Write(data); err != nil {
		temporary.Close()
		return fmt.Errorf("write command center profile: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close command center profile: %w", err)
	}
	if err := os.Rename(temporaryPath, ProfilePath(home)); err != nil {
		return fmt.Errorf("install command center profile: %w", err)
	}
	return os.Chmod(ProfilePath(home), 0600)
}
