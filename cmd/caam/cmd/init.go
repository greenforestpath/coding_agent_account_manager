// Package cmd implements the CLI commands for caam.
package cmd

import (
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/authfile"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/profile"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize caam for first-time use",
	Long: `Sets up caam for first-time use by creating necessary directories
and detecting installed CLI tools.

This command is idempotent - it's safe to run multiple times.

Actions performed:
  1. Creates data directories (~/.local/share/caam/{vault,profiles})
  2. Creates config directory (~/.config/caam/)
  3. Detects installed CLI tools (codex, claude, gemini)

Examples:
  caam init           # Interactive setup
  caam init --quiet   # Non-interactive, just create directories`,
	RunE: func(cmd *cobra.Command, args []string) error {
		quiet, _ := cmd.Flags().GetBool("quiet")

		if !quiet {
			fmt.Println("Welcome to caam - Coding Agent Account Manager!")
			fmt.Println()
		}

		// Create directories
		if err := createDirectories(quiet); err != nil {
			return err
		}

		// Detect tools
		detectTools(quiet)

		// Show next steps
		if !quiet {
			printNextSteps()
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.Flags().Bool("quiet", false, "non-interactive mode, just create directories")
}

// createDirectories creates the necessary data directories.
func createDirectories(quiet bool) error {
	if !quiet {
		fmt.Println("Creating directories...")
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("get home dir: %w", err)
	}

	// Determine paths using XDG conventions
	xdgData := os.Getenv("XDG_DATA_HOME")
	if xdgData == "" {
		xdgData = filepath.Join(homeDir, ".local", "share")
	}

	xdgConfig := os.Getenv("XDG_CONFIG_HOME")
	if xdgConfig == "" {
		xdgConfig = filepath.Join(homeDir, ".config")
	}

	dirs := []struct {
		path string
		name string
	}{
		{filepath.Join(xdgData, "caam"), "caam data directory"},
		{authfile.DefaultVaultPath(), "vault directory"},
		{profile.DefaultStorePath(), "profiles directory"},
		{filepath.Join(xdgConfig, "caam"), "config directory"},
	}

	for _, dir := range dirs {
		// Check if exists
		if info, err := os.Stat(dir.path); err == nil && info.IsDir() {
			if !quiet {
				fmt.Printf("  ✓ %s already exists\n", dir.name)
			}
			continue
		}

		// Create directory
		if err := os.MkdirAll(dir.path, 0700); err != nil {
			return fmt.Errorf("create %s: %w", dir.name, err)
		}

		if !quiet {
			fmt.Printf("  ✓ Created %s\n", dir.name)
		}
	}

	if !quiet {
		fmt.Println()
	}

	return nil
}

// detectTools checks for installed CLI tools.
func detectTools(quiet bool) {
	if !quiet {
		fmt.Println("Detecting CLI tools...")
	}

	toolBinaries := map[string]string{
		"codex":  "codex",
		"claude": "claude",
		"gemini": "gemini",
	}

	foundCount := 0
	for tool, binary := range toolBinaries {
		path, err := osexec.LookPath(binary)
		if err == nil {
			if !quiet {
				fmt.Printf("  ✓ %s found at %s\n", tool, path)
			}
			foundCount++
		} else {
			if !quiet {
				fmt.Printf("  ✗ %s not found\n", tool)
			}
		}
	}

	if !quiet {
		fmt.Println()
		if foundCount == 0 {
			fmt.Println("No CLI tools found. Install at least one:")
			fmt.Println("  - Codex CLI: https://github.com/openai/codex-cli")
			fmt.Println("  - Claude Code: https://github.com/anthropics/claude-code")
			fmt.Println("  - Gemini CLI: https://github.com/google/gemini-cli")
			fmt.Println()
		}
	}
}

// printNextSteps shows what to do after initialization.
func printNextSteps() {
	fmt.Println("caam is ready to use!")
	fmt.Println()
	fmt.Println("Next steps:")
	fmt.Println("  1. Login to your AI coding tool (e.g., 'codex login' or '/login' in claude)")
	fmt.Println("  2. Backup your auth: caam backup <tool> <email@example.com>")
	fmt.Println("  3. Repeat for additional accounts")
	fmt.Println("  4. Switch instantly: caam activate <tool> <profile>")
	fmt.Println()
	fmt.Println("Useful commands:")
	fmt.Println("  caam status    - Show currently active profiles")
	fmt.Println("  caam ls        - List all saved profiles")
	fmt.Println("  caam doctor    - Check for setup issues")
	fmt.Println()
}
