package cmd

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/authfile"
	caamdb "github.com/Dicklesworthstone/coding_agent_account_manager/internal/db"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/health"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/rotation"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/wrap"
	"github.com/spf13/cobra"
)

// runCmd wraps AI CLI execution with automatic rate limit handling.
var runCmd = &cobra.Command{
	Use:   "run <tool> [-- args...]",
	Short: "Run AI CLI with automatic account switching",
	Long: `Wraps AI CLI execution with transparent rate limit detection and automatic
profile switching. This is the "zero friction" mode - just use caam run instead
of calling the CLI directly.

When a rate limit is detected:
1. The current profile is put into cooldown
2. The next best profile is automatically selected
3. The command is re-executed seamlessly

Examples:
  caam run claude -- "explain this code"
  caam run codex -- --model gpt-5 "write tests"
  caam run gemini -- "summarize this file"

  # Interactive mode (no auto-retry on rate limit)
  caam run claude

For shell integration, add an alias:
  alias claude='caam run claude --'

Then you can just use:
  claude "explain this code"

And rate limits will be handled automatically!`,
	Args:               cobra.MinimumNArgs(1),
	DisableFlagParsing: false,
	RunE:               runWrap,
}

func init() {
	runCmd.Flags().Int("max-retries", 1, "maximum retry attempts on rate limit (0 = no retries)")
	runCmd.Flags().Duration("cooldown", 60*time.Minute, "cooldown duration after rate limit")
	runCmd.Flags().Bool("quiet", false, "suppress profile switch notifications")
	runCmd.Flags().String("algorithm", "smart", "rotation algorithm (smart, round_robin, random)")
}

func runWrap(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("tool name required")
	}

	tool := strings.ToLower(args[0])

	// Validate tool
	if _, ok := tools[tool]; !ok {
		return fmt.Errorf("unknown tool: %s (supported: codex, claude, gemini)", tool)
	}

	// Parse CLI args (everything after the tool name)
	var cliArgs []string
	if len(args) > 1 {
		cliArgs = args[1:]
	}

	// Get flags
	maxRetries, _ := cmd.Flags().GetInt("max-retries")
	cooldown, _ := cmd.Flags().GetDuration("cooldown")
	quiet, _ := cmd.Flags().GetBool("quiet")
	algorithmStr, _ := cmd.Flags().GetString("algorithm")

	// Parse algorithm
	var algorithm rotation.Algorithm
	switch strings.ToLower(algorithmStr) {
	case "smart":
		algorithm = rotation.AlgorithmSmart
	case "round_robin", "roundrobin":
		algorithm = rotation.AlgorithmRoundRobin
	case "random":
		algorithm = rotation.AlgorithmRandom
	default:
		return fmt.Errorf("unknown algorithm: %s (supported: smart, round_robin, random)", algorithmStr)
	}

	// Initialize vault
	if vault == nil {
		vault = authfile.NewVault(authfile.DefaultVaultPath())
	}

	// Initialize database
	db, err := caamdb.Open()
	if err != nil {
		// Non-fatal: cooldowns won't be recorded but execution can continue
		fmt.Fprintf(os.Stderr, "warning: database unavailable, cooldowns will not be recorded\n")
		db = nil
	}
	if db != nil {
		defer db.Close()
	}

	// Initialize health storage
	healthStore := health.NewStorage("")

	// Build config
	cfg := wrap.Config{
		Provider:         tool,
		Args:             cliArgs,
		MaxRetries:       maxRetries,
		CooldownDuration: cooldown,
		NotifyOnSwitch:   !quiet,
		Algorithm:        algorithm,
		Stdout:           os.Stdout,
		Stderr:           os.Stderr,
	}

	// Get working directory
	cwd, err := os.Getwd()
	if err == nil {
		cfg.WorkDir = cwd
	}

	// Create wrapper
	wrapper := wrap.NewWrapper(vault, db, healthStore, cfg)

	// Set up context with signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigChan
		cancel()
	}()

	// Run wrapped command
	result := wrapper.Run(ctx)

	// Handle result
	if result.Err != nil {
		return result.Err
	}

	// Exit with the same code as the wrapped command
	// Note: os.Exit bypasses defer, so close db explicitly first
	if result.ExitCode != 0 {
		if db != nil {
			db.Close()
		}
		os.Exit(result.ExitCode)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(runCmd)
}
