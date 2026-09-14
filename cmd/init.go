package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"squadron/internal/paths"
)

var initConfigPath string

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize local Squadron runtime state",
	Long: `Create the local .squadron runtime directory.

Workspace variables and secrets are managed by Command Center and are not
stored by the worker. For guided configuration, use 'squadron quickstart'.`,
	Run: func(cmd *cobra.Command, args []string) {
		if err := applyHome(initConfigPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		if err := RunInit(); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().StringVarP(&initConfigPath, "config", "c", "", "Path to config directory")
}

func RunInit() error {
	if err := paths.EnsureHome(); err != nil {
		return fmt.Errorf("creating squadron home: %w", err)
	}
	if err := ensureSquadronGitignored(); err != nil {
		return fmt.Errorf("updating .gitignore: %w", err)
	}
	fmt.Println("Squadron runtime state initialized. Variables are managed in Command Center.")
	return nil
}

func ensureSquadronGitignored() error {
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}
	gitignorePath := filepath.Join(cwd, ".gitignore")
	existing, err := os.ReadFile(gitignorePath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	if gitignoreContains(existing, ".squadron") {
		return nil
	}
	var buf strings.Builder
	if len(existing) > 0 {
		buf.Write(existing)
		if existing[len(existing)-1] != '\n' {
			buf.WriteByte('\n')
		}
	}
	buf.WriteString(".squadron/\n")
	return os.WriteFile(gitignorePath, []byte(buf.String()), 0644)
}

func gitignoreContains(data []byte, entry string) bool {
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		line = strings.TrimPrefix(strings.TrimSuffix(line, "/"), "/")
		if line == entry {
			return true
		}
	}
	return false
}

// EnsureInitialized preserves the existing CLI contract while initialization
// now only creates local runtime state. It never creates variable storage.
func EnsureInitialized(_ bool) error { return paths.EnsureHome() }

func promptYesNo(question string) bool {
	reader := bufio.NewReader(os.Stdin)
	fmt.Printf("%s [y/N]: ", question)
	answer, _ := reader.ReadString('\n')
	answer = strings.TrimSpace(strings.ToLower(answer))
	return answer == "y" || answer == "yes"
}

func promptSecret(reader *bufio.Reader, prompt string) (string, error) {
	if term.IsTerminal(int(os.Stdin.Fd())) {
		fmt.Printf("%s: ", prompt)
		value, err := term.ReadPassword(int(os.Stdin.Fd()))
		fmt.Println()
		return strings.TrimSpace(string(value)), err
	}
	fmt.Printf("%s: ", prompt)
	value, err := reader.ReadString('\n')
	return strings.TrimSpace(value), err
}
