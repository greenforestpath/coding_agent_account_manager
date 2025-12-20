package signals

import (
	"os"
	"path/filepath"
	"testing"
)

func TestPIDFileRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "caam.pid")

	if err := WritePIDFile(pidPath, 12345); err != nil {
		t.Fatalf("WritePIDFile: %v", err)
	}

	pid, err := ReadPIDFile(pidPath)
	if err != nil {
		t.Fatalf("ReadPIDFile: %v", err)
	}
	if pid != 12345 {
		t.Fatalf("pid=%d, want 12345", pid)
	}

	if err := RemovePIDFile(pidPath); err != nil {
		t.Fatalf("RemovePIDFile: %v", err)
	}
	if _, err := os.Stat(pidPath); !os.IsNotExist(err) {
		t.Fatalf("pid file should be removed, stat err=%v", err)
	}
}

func TestDefaultPIDFilePathUsesCAAMHome(t *testing.T) {
	orig := os.Getenv("CAAM_HOME")
	defer os.Setenv("CAAM_HOME", orig)

	tmpDir := t.TempDir()
	os.Setenv("CAAM_HOME", tmpDir)

	got := DefaultPIDFilePath()
	want := filepath.Join(tmpDir, "caam.pid")
	if got != want {
		t.Fatalf("DefaultPIDFilePath=%q, want %q", got, want)
	}
}

func TestDefaultPIDFilePathWithoutCAAMHome(t *testing.T) {
	orig := os.Getenv("CAAM_HOME")
	defer os.Setenv("CAAM_HOME", orig)

	os.Unsetenv("CAAM_HOME")

	got := DefaultPIDFilePath()
	// Should end with caam.pid
	if !filepath.IsAbs(got) && got != filepath.Join(".caam", "caam.pid") {
		if filepath.Base(got) != "caam.pid" {
			t.Fatalf("DefaultPIDFilePath=%q should end with caam.pid", got)
		}
	}
}

func TestWritePIDFileCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "subdir", "nested", "caam.pid")

	if err := WritePIDFile(pidPath, 99999); err != nil {
		t.Fatalf("WritePIDFile with nested dir: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(pidPath); os.IsNotExist(err) {
		t.Fatal("PID file was not created")
	}

	// Verify content
	pid, err := ReadPIDFile(pidPath)
	if err != nil {
		t.Fatalf("ReadPIDFile: %v", err)
	}
	if pid != 99999 {
		t.Fatalf("pid=%d, want 99999", pid)
	}
}

func TestReadPIDFileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "nonexistent.pid")

	_, err := ReadPIDFile(pidPath)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestRemovePIDFileNotExists(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "nonexistent.pid")

	// Should not error when file doesn't exist
	err := RemovePIDFile(pidPath)
	if err != nil {
		t.Fatalf("RemovePIDFile on nonexistent file should not error: %v", err)
	}
}

func TestReadPIDFileInvalidContent(t *testing.T) {
	tmpDir := t.TempDir()
	pidPath := filepath.Join(tmpDir, "bad.pid")

	// Write invalid content
	if err := os.WriteFile(pidPath, []byte("not-a-number\n"), 0600); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := ReadPIDFile(pidPath)
	if err == nil {
		t.Fatal("expected error for invalid PID content")
	}
}
