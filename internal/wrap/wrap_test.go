package wrap

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/authfile"
	caamdb "github.com/Dicklesworthstone/coding_agent_account_manager/internal/db"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/health"
	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/rotation"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.MaxRetries != 1 {
		t.Errorf("MaxRetries = %d, want 1", cfg.MaxRetries)
	}
	if cfg.CooldownDuration != 60*time.Minute {
		t.Errorf("CooldownDuration = %v, want 60m", cfg.CooldownDuration)
	}
	if !cfg.NotifyOnSwitch {
		t.Error("NotifyOnSwitch = false, want true")
	}
	if cfg.Algorithm != rotation.AlgorithmSmart {
		t.Errorf("Algorithm = %v, want smart", cfg.Algorithm)
	}
	if cfg.Stdout == nil {
		t.Error("Stdout = nil, want os.Stdout")
	}
	if cfg.Stderr == nil {
		t.Error("Stderr = nil, want os.Stderr")
	}
}

func TestNewWrapper(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Provider = "claude"
	cfg.Stdout = nil // Should default to os.Stdout
	cfg.Stderr = nil // Should default to os.Stderr

	w := NewWrapper(nil, nil, nil, cfg)

	if w == nil {
		t.Fatal("NewWrapper returned nil")
	}
	if w.config.Stdout == nil {
		t.Error("Stdout not defaulted")
	}
	if w.config.Stderr == nil {
		t.Error("Stderr not defaulted")
	}
}

func TestWrapper_Run_NoProfiles(t *testing.T) {
	// Create temp vault with no profiles
	tmpDir := t.TempDir()
	vault := authfile.NewVault(tmpDir)

	cfg := DefaultConfig()
	cfg.Provider = "claude"
	cfg.NotifyOnSwitch = false

	stderr := &bytes.Buffer{}
	cfg.Stderr = stderr

	w := NewWrapper(vault, nil, nil, cfg)

	result := w.Run(context.Background())

	if result.ExitCode != 1 {
		t.Errorf("ExitCode = %d, want 1", result.ExitCode)
	}
	if result.Err == nil {
		t.Error("Err = nil, want error about no profiles")
	}
}

func TestWrapper_Run_WithProfile(t *testing.T) {
	// Create temp vault with a profile
	tmpDir := t.TempDir()
	vault := authfile.NewVault(tmpDir)

	// Create a fake profile
	profileDir := filepath.Join(tmpDir, "claude", "test@example.com")
	if err := os.MkdirAll(profileDir, 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(profileDir, ".claude.json"), []byte(`{}`), 0600); err != nil {
		t.Fatalf("write file: %v", err)
	}

	// Create health storage
	healthPath := filepath.Join(tmpDir, "health.json")
	healthStore := health.NewStorage(healthPath)

	// Create temp database
	dbPath := filepath.Join(tmpDir, "caam.db")
	db, err := caamdb.OpenAt(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer db.Close()

	cfg := DefaultConfig()
	cfg.Provider = "claude"
	cfg.Args = []string{"--version"} // Simple command that should work
	cfg.NotifyOnSwitch = false

	stdout := &bytes.Buffer{}
	stderr := &bytes.Buffer{}
	cfg.Stdout = stdout
	cfg.Stderr = stderr

	w := NewWrapper(vault, db, healthStore, cfg)

	// This will fail because claude isn't installed, but that's OK
	// We're testing the wrapper logic, not the actual CLI
	result := w.Run(context.Background())

	// Check that profile was used
	if len(result.ProfilesUsed) == 0 {
		t.Error("No profiles used")
	}
	if len(result.ProfilesUsed) > 0 && result.ProfilesUsed[0] != "test@example.com" {
		t.Errorf("ProfilesUsed[0] = %q, want test@example.com", result.ProfilesUsed[0])
	}
}

func TestAuthFileSetForProvider(t *testing.T) {
	tests := []struct {
		provider string
		wantOK   bool
	}{
		{"claude", true},
		{"codex", true},
		{"gemini", true},
		{"unknown", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			_, ok := authFileSetForProvider(tt.provider)
			if ok != tt.wantOK {
				t.Errorf("authFileSetForProvider(%q) ok = %v, want %v", tt.provider, ok, tt.wantOK)
			}
		})
	}
}

func TestBinForProvider(t *testing.T) {
	tests := []struct {
		provider string
		want     string
	}{
		{"claude", "claude"},
		{"codex", "codex"},
		{"gemini", "gemini"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.provider, func(t *testing.T) {
			got := binForProvider(tt.provider)
			if got != tt.want {
				t.Errorf("binForProvider(%q) = %q, want %q", tt.provider, got, tt.want)
			}
		})
	}
}

func TestTeeWriter(t *testing.T) {
	// The teeWriter is internal to the package and tested via runOnce.
	// This placeholder ensures the test file compiles.
}

func TestResult(t *testing.T) {
	r := &Result{
		ExitCode:     0,
		ProfilesUsed: []string{"a", "b"},
		RateLimitHit: true,
		RetryCount:   1,
	}

	if r.ExitCode != 0 {
		t.Errorf("ExitCode = %d, want 0", r.ExitCode)
	}
	if len(r.ProfilesUsed) != 2 {
		t.Errorf("len(ProfilesUsed) = %d, want 2", len(r.ProfilesUsed))
	}
	if !r.RateLimitHit {
		t.Error("RateLimitHit = false, want true")
	}
	if r.RetryCount != 1 {
		t.Errorf("RetryCount = %d, want 1", r.RetryCount)
	}
}
