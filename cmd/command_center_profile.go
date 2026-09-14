package cmd

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"strings"

	"golang.org/x/term"

	commandcenter "squadron/internal/commandcenter"
	"squadron/internal/paths"
)

func ensureCommandCenterProfile() (commandcenter.Profile, error) {
	home, err := paths.SquadronHome()
	if err != nil {
		return commandcenter.Profile{}, fmt.Errorf("locate Squadron home: %w", err)
	}
	profile, err := commandcenter.Load(home)
	if err == nil {
		return profile, nil
	}
	if !errors.Is(err, commandcenter.ErrNotConfigured) {
		return commandcenter.Profile{}, err
	}
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		return commandcenter.Profile{}, fmt.Errorf("command center connection is not configured; run squadron engage from an interactive terminal once")
	}

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("\nConnect this Squadron worker to Command Center.")
	fmt.Println("Get the runner credential from the workspace's connection details.")

	url, err := promptRequired(reader, "Command Center WebSocket URL")
	if err != nil {
		return commandcenter.Profile{}, err
	}
	credential, err := promptSecret(reader, "Runner credential")
	if err != nil {
		return commandcenter.Profile{}, fmt.Errorf("read runner credential: %w", err)
	}

	profile = commandcenter.Profile{
		URL:              url,
		WorkerCredential: credential,
	}
	if err := commandcenter.Save(home, profile); err != nil {
		return commandcenter.Profile{}, err
	}
	fmt.Printf("Saved the local Command Center connection in %s.\n", commandcenter.ProfilePath(home))
	return profile, nil
}

func promptRequired(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	value, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", fmt.Errorf("%s is required", strings.ToLower(label))
	}
	return value, nil
}
