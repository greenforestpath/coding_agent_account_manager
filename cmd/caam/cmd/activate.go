package cmd

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/authfile"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/config"
	caamdb "github.com/Dicklesworthstone/coding_agent_account_manager/internal/db"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/health"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/refresh"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/rotation"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/stealth"
	"github.com/spf13/cobra"
)

// activateCmd restores auth files from the vault.
var activateCmd = &cobra.Command{
	Use:     "activate <tool> [profile-name]",
	Aliases: []string{"switch", "use"},
	Short:   "Activate a profile (instant switch)",
	Long: `Restores auth files from the vault, instantly switching to that account.

This is the magic command - sub-second account switching without any login flows!

Examples:
  caam activate codex work-account
  caam activate codex
  caam activate claude personal-max
  caam activate gemini team-ultra
  caam activate claude --auto

The --auto flag enables smart profile rotation, which selects the best profile
based on health status, cooldown state, and usage patterns. Three algorithms
are available (configured in config.yaml):

  smart       - Multi-factor scoring (health, cooldown, recency)
  round_robin - Sequential rotation through profiles
  random      - Random selection

After activating, just run the tool normally - it will use the new account.`,
	Args: cobra.RangeArgs(1, 2),
	RunE: runActivate,
}

func init() {
	activateCmd.Flags().Bool("backup-current", false, "backup current auth before switching")
	activateCmd.Flags().Bool("force", false, "activate even if the profile is in cooldown")
	activateCmd.Flags().Bool("auto", false, "auto-select profile using rotation algorithm")
}

func runActivate(cmd *cobra.Command, args []string) error {
	tool := strings.ToLower(args[0])
	autoSelect, _ := cmd.Flags().GetBool("auto")

	var profileName string
	if len(args) == 2 && !autoSelect {
		profileName = args[1]
	} else if autoSelect {
		// Use rotation algorithm to select profile
		selected, err := selectProfileWithRotation(tool)
		if err != nil {
			return err
		}
		profileName = selected
	} else {
		var source string
		var err error
		profileName, source, err = resolveActivateProfile(tool)
		if err != nil {
			return err
		}
		if source != "" {
			fmt.Printf("Using %s: %s/%s\n", source, tool, profileName)
		}
	}

	getFileSet, ok := tools[tool]
	if !ok {
		return fmt.Errorf("unknown tool: %s (supported: codex, claude, gemini)", tool)
	}

	fileSet := getFileSet()

	// Safety: on first activate, preserve the user's pre-caam auth state.
	if did, err := vault.BackupOriginal(fileSet); err != nil {
		return fmt.Errorf("backup original auth: %w", err)
	} else if did {
		fmt.Printf("Backed up original %s auth to %s\n", tool, "_original")
	}

	spmCfg, err := config.LoadSPMConfig()
	if err != nil {
		// Invalid config should not crash activation; fall back to defaults.
		spmCfg = config.DefaultSPMConfig()
	}

	// Stealth: enforce per-profile cooldowns (opt-in).
	if spmCfg.Stealth.Cooldown.Enabled {
		force, _ := cmd.Flags().GetBool("force")

		db, err := caamdb.Open()
		if err != nil {
			fmt.Printf("Warning: could not open database for cooldown check: %v\n", err)
		} else {
			defer db.Close()

			now := time.Now().UTC()
			ev, err := db.ActiveCooldown(tool, profileName, now)
			if err != nil {
				fmt.Printf("Warning: could not check cooldowns: %v\n", err)
			} else if ev != nil {
				remaining := time.Until(ev.CooldownUntil)
				if remaining < 0 {
					remaining = 0
				}
				hitAgo := now.Sub(ev.HitAt)
				if hitAgo < 0 {
					hitAgo = 0
				}

				fmt.Printf("Warning: %s/%s is in cooldown\n", tool, profileName)
				fmt.Printf("  Limit hit: %s ago\n", formatDurationShort(hitAgo))
				fmt.Printf("  Cooldown remaining: %s\n", formatDurationShort(remaining))

				if !force {
					if !isTerminal() {
						return fmt.Errorf("%s/%s is in cooldown (%s remaining); re-run with --force to activate anyway", tool, profileName, formatDurationShort(remaining))
					}

					ok, err := confirmProceed(cmd.InOrStdin(), cmd.OutOrStdout())
					if err != nil {
						return fmt.Errorf("confirm proceed: %w", err)
					}
					if !ok {
						fmt.Println("Cancelled")
						return nil
					}
				} else {
					fmt.Println("Proceeding due to --force...")
				}
			}
		}
	}

	// Step 1: Refresh if needed
	_ = refreshIfNeeded(cmd.Context(), tool, profileName)

	// Smart auto-backup before switch (based on safety config)
	backupMode := strings.TrimSpace(spmCfg.Safety.AutoBackupBeforeSwitch)
	if backupMode == "" {
		backupMode = "smart" // Default
	}

	// Check if --backup-current flag overrides config
	backupFirst, _ := cmd.Flags().GetBool("backup-current")
	if backupFirst {
		backupMode = "always"
	}

	if backupMode != "never" {
		shouldBackup := false
		currentProfile, _ := vault.ActiveProfile(fileSet)

		switch backupMode {
		case "always":
			// Always backup if there are auth files and we're switching to a different profile
			shouldBackup = currentProfile != profileName
		case "smart":
			// Backup only if current state doesn't match any vault profile (would be lost)
			shouldBackup = currentProfile == "" && authfile.HasAuthFiles(fileSet)
		}

		if shouldBackup {
			backupName, err := vault.BackupCurrent(fileSet)
			if err != nil {
				fmt.Printf("Warning: could not auto-backup current state: %v\n", err)
			} else if backupName != "" {
				fmt.Printf("Auto-backed up current state to %s\n", backupName)

				// Rotate old backups if limit is set
				if spmCfg.Safety.MaxAutoBackups > 0 {
					if err := vault.RotateAutoBackups(tool, spmCfg.Safety.MaxAutoBackups); err != nil {
						fmt.Printf("Warning: could not rotate old backups: %v\n", err)
					}
				}
			}
		}
	}

	// Stealth: optional delay before the actual switch happens.
	if spmCfg.Stealth.SwitchDelay.Enabled {
		delay, err := stealth.ComputeDelay(spmCfg.Stealth.SwitchDelay.MinSeconds, spmCfg.Stealth.SwitchDelay.MaxSeconds, nil)
		if err != nil {
			fmt.Printf("Warning: invalid stealth.switch_delay config: %v\n", err)
		} else if delay > 0 {
			fmt.Printf("Stealth mode: waiting %d seconds before switch...\n", int(delay.Round(time.Second).Seconds()))

			sigCh := make(chan os.Signal, 1)
			signal.Notify(sigCh, os.Interrupt)

			skip := make(chan struct{})
			stop := make(chan struct{})
			go func() {
				select {
				case <-sigCh:
					close(skip)
				case <-stop:
				case <-cmd.Context().Done():
				}
			}()

			skipped, waitErr := stealth.Wait(cmd.Context(), delay, stealth.WaitOptions{
				Output:        os.Stdout,
				Skip:          skip,
				ShowCountdown: spmCfg.Stealth.SwitchDelay.ShowCountdown,
			})

			close(stop)
			signal.Stop(sigCh)

			if waitErr != nil {
				return fmt.Errorf("stealth delay: %w", waitErr)
			}
			if skipped {
				fmt.Println("Skipping delay...")
			}
		}
	}

	// Restore from vault
	if err := vault.Restore(fileSet, profileName); err != nil {
		return fmt.Errorf("activate failed: %w", err)
	}

	fmt.Printf("Activated %s profile '%s'\n", tool, profileName)
	fmt.Printf("  Run '%s' to start using this account\n", tool)
	return nil
}

func resolveActivateProfile(tool string) (profileName string, source string, err error) {
	// Prefer project association (if enabled).
	spmCfg, err := config.LoadSPMConfig()
	if err == nil && spmCfg.Project.Enabled && projectStore != nil {
		cwd, wdErr := os.Getwd()
		if wdErr != nil {
			return "", "", fmt.Errorf("get current directory: %w", wdErr)
		}
		resolved, resErr := projectStore.Resolve(cwd)
		if resErr == nil {
			if p := strings.TrimSpace(resolved.Profiles[tool]); p != "" {
				src := resolved.Sources[tool]
				if src == "" || src == cwd {
					return p, "project association", nil
				}
				if src == "<default>" {
					return p, "project default", nil
				}
				return p, "project association", nil
			}
		}
	}

	// Fall back to configured default profile (caam config.json).
	if cfg != nil {
		if p := strings.TrimSpace(cfg.GetDefault(tool)); p != "" {
			return p, "default profile", nil
		}
	}

	return "", "", fmt.Errorf("no profile specified for %s and no project association/default found\nHint: run 'caam activate %s <profile-name>', 'caam use %s <profile-name>', or 'caam project set %s <profile-name>'", tool, tool, tool, tool)
}

func refreshIfNeeded(ctx context.Context, provider, profile string) error {
	if ctx == nil {
		ctx = context.Background()
	}

	// Try to get health data. If missing, we might want to populate it?
	// But RefreshProfile uses vault path.
	// If we don't have health data, we don't know expiry, so we can't decide to refresh.
	// `getProfileHealth` in root.go parses files.
	// We should use that logic? `getProfileHealth` is in `root.go` (same package).
	h := getProfileHealth(provider, profile)

	if !refresh.ShouldRefresh(h, 0) {
		return nil
	}

	fmt.Printf("Refreshing token (%s)... ", health.FormatTimeRemaining(h.TokenExpiresAt))

	err := refresh.RefreshProfile(ctx, provider, profile, vault, healthStore)
	if err != nil {
		if errors.Is(err, refresh.ErrUnsupported) {
			fmt.Printf("skipped (%v)\n", err)
			return nil
		}
		fmt.Printf("failed (%v)\n", err)
		return nil // Continue activation even if refresh fails
	}

	fmt.Println("done")
	return nil
}

// selectProfileWithRotation uses the rotation algorithm to select a profile.
func selectProfileWithRotation(tool string) (string, error) {
	// Load SPM config to get rotation settings
	spmCfg, err := config.LoadSPMConfig()
	if err != nil {
		spmCfg = config.DefaultSPMConfig()
	}

	// Check if rotation is enabled
	if !spmCfg.Stealth.Rotation.Enabled {
		return "", fmt.Errorf("rotation is not enabled; enable it in config.yaml under stealth.rotation.enabled")
	}

	// Get list of profiles for this tool
	profiles, err := vault.List(tool)
	if err != nil {
		return "", fmt.Errorf("list profiles: %w", err)
	}

	if len(profiles) == 0 {
		return "", fmt.Errorf("no profiles found for %s; create one with 'caam backup %s <name>'", tool, tool)
	}

	// Get algorithm from config
	algorithm := rotation.Algorithm(spmCfg.Stealth.Rotation.Algorithm)
	if algorithm == "" {
		algorithm = rotation.AlgorithmSmart
	}

	// Open database for cooldown checks
	var db *caamdb.DB
	if spmCfg.Stealth.Cooldown.Enabled {
		db, err = caamdb.Open()
		if err != nil {
			fmt.Printf("Warning: could not open database for rotation: %v\n", err)
		} else {
			defer db.Close()
		}
	}

	// Get current active profile
	getFileSet, ok := tools[tool]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", tool)
	}
	currentProfile, _ := vault.ActiveProfile(getFileSet())

	// Create selector and select profile
	selector := rotation.NewSelector(algorithm, healthStore, db)
	result, err := selector.Select(tool, profiles, currentProfile)
	if err != nil {
		return "", fmt.Errorf("rotation select: %w", err)
	}

	// Display the selection result
	fmt.Print(rotation.FormatResult(result))
	fmt.Println()

	return result.Selected, nil
}
