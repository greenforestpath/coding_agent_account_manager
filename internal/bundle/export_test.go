package bundle

import (
	"archive/zip"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDefaultExportOptions(t *testing.T) {
	opts := DefaultExportOptions()

	if !opts.IncludeConfig {
		t.Error("IncludeConfig should default to true")
	}
	if !opts.IncludeProjects {
		t.Error("IncludeProjects should default to true")
	}
	if !opts.IncludeHealth {
		t.Error("IncludeHealth should default to true")
	}
	if opts.IncludeDatabase {
		t.Error("IncludeDatabase should default to false")
	}
	if !opts.IncludeSyncConfig {
		t.Error("IncludeSyncConfig should default to true")
	}
	if opts.Encrypt {
		t.Error("Encrypt should default to false")
	}
}

func TestVaultExporter_Export_DryRun(t *testing.T) {
	// Create test vault structure
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")
	claudeDir := filepath.Join(vaultDir, "claude", "alice@gmail.com")
	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "auth.json"), []byte(`{"token":"test"}`), 0600); err != nil {
		t.Fatal(err)
	}

	exporter := &VaultExporter{
		VaultPath: vaultDir,
		DataPath:  tmpDir,
	}

	opts := &ExportOptions{
		OutputDir: tmpDir,
		DryRun:    true,
	}

	result, err := exporter.Export(opts)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if result.OutputPath == "" {
		t.Error("OutputPath should be set")
	}

	if result.Manifest == nil {
		t.Error("Manifest should be set")
	}

	// Verify no file was created in dry run
	if _, err := os.Stat(result.OutputPath); !os.IsNotExist(err) {
		t.Error("File should not be created in dry run mode")
	}

	// Check manifest has the profile
	if len(result.Manifest.Contents.Vault.Profiles) == 0 {
		t.Error("Vault profiles should be populated")
	}

	profiles := result.Manifest.Contents.Vault.Profiles["claude"]
	if len(profiles) != 1 || profiles[0] != "alice@gmail.com" {
		t.Errorf("Expected claude/alice@gmail.com, got %v", profiles)
	}
}

func TestVaultExporter_Export_Full(t *testing.T) {
	// Create test vault structure
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")
	claudeDir := filepath.Join(vaultDir, "claude", "alice@gmail.com")
	codexDir := filepath.Join(vaultDir, "codex", "work@company.com")

	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(codexDir, 0700); err != nil {
		t.Fatal(err)
	}

	if err := os.WriteFile(filepath.Join(claudeDir, "auth.json"), []byte(`{"token":"claude_token"}`), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(codexDir, "auth.json"), []byte(`{"token":"codex_token"}`), 0600); err != nil {
		t.Fatal(err)
	}

	// Create config file
	configPath := filepath.Join(tmpDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("default_provider: claude"), 0600); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	exporter := &VaultExporter{
		VaultPath:  vaultDir,
		DataPath:   tmpDir,
		ConfigPath: configPath,
	}

	opts := &ExportOptions{
		OutputDir:     outputDir,
		IncludeConfig: true,
	}

	result, err := exporter.Export(opts)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Errorf("Output file should exist: %v", err)
	}

	// Verify it's a valid zip
	reader, err := zip.OpenReader(result.OutputPath)
	if err != nil {
		t.Fatalf("Should be valid zip: %v", err)
	}
	defer reader.Close()

	// Check expected files are in the zip
	fileNames := make(map[string]bool)
	for _, f := range reader.File {
		fileNames[f.Name] = true
	}

	if !fileNames["manifest.json"] {
		t.Error("manifest.json should be in bundle")
	}
	if !fileNames["vault/claude/alice@gmail.com/auth.json"] {
		t.Error("claude auth should be in bundle")
	}
	if !fileNames["vault/codex/work@company.com/auth.json"] {
		t.Error("codex auth should be in bundle")
	}
	if !fileNames["config.yaml"] {
		t.Error("config should be in bundle")
	}

	// Verify manifest content
	if result.Manifest.Contents.Vault.TotalProfiles != 2 {
		t.Errorf("TotalProfiles = %d, want 2", result.Manifest.Contents.Vault.TotalProfiles)
	}
}

func TestVaultExporter_Export_WithEncryption(t *testing.T) {
	// Create test vault structure
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")
	claudeDir := filepath.Join(vaultDir, "claude", "test@gmail.com")

	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "auth.json"), []byte(`{"token":"secret"}`), 0600); err != nil {
		t.Fatal(err)
	}

	outputDir := filepath.Join(tmpDir, "output")
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		t.Fatal(err)
	}

	exporter := &VaultExporter{
		VaultPath: vaultDir,
		DataPath:  tmpDir,
	}

	password := "test-password-123"
	opts := &ExportOptions{
		OutputDir: outputDir,
		Encrypt:   true,
		Password:  password,
	}

	result, err := exporter.Export(opts)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	if !result.Encrypted {
		t.Error("Result should be marked as encrypted")
	}

	// Verify file exists
	if _, err := os.Stat(result.OutputPath); err != nil {
		t.Errorf("Output file should exist: %v", err)
	}

	// Verify metadata file exists
	metaPath := result.OutputPath + ".meta"
	metaData, err := os.ReadFile(metaPath)
	if err != nil {
		t.Fatalf("Metadata file should exist: %v", err)
	}

	var meta EncryptionMetadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatalf("Metadata should be valid JSON: %v", err)
	}

	if meta.Algorithm != "aes-256-gcm" {
		t.Errorf("Algorithm = %q, want aes-256-gcm", meta.Algorithm)
	}

	// The encrypted file should NOT be a valid zip directly
	if _, err := zip.OpenReader(result.OutputPath); err == nil {
		t.Error("Encrypted file should not be readable as plain zip")
	}
}

func TestVaultExporter_Export_ProviderFilter(t *testing.T) {
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")

	// Create profiles for multiple providers
	for _, provider := range []string{"claude", "codex", "gemini"} {
		dir := filepath.Join(vaultDir, provider, "profile@test.com")
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "auth.json"), []byte("{}"), 0600); err != nil {
			t.Fatal(err)
		}
	}

	exporter := &VaultExporter{
		VaultPath: vaultDir,
		DataPath:  tmpDir,
	}

	opts := &ExportOptions{
		OutputDir:      filepath.Join(tmpDir, "output"),
		ProviderFilter: []string{"claude"},
		DryRun:         true,
	}

	result, err := exporter.Export(opts)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should only have claude
	if len(result.Manifest.Contents.Vault.Profiles) != 1 {
		t.Errorf("Should have 1 provider, got %d", len(result.Manifest.Contents.Vault.Profiles))
	}

	if _, ok := result.Manifest.Contents.Vault.Profiles["claude"]; !ok {
		t.Error("Should have claude provider")
	}
}

func TestVaultExporter_Export_SkipsSystemProfiles(t *testing.T) {
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")

	// Create user profile
	userDir := filepath.Join(vaultDir, "claude", "user@gmail.com")
	if err := os.MkdirAll(userDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userDir, "auth.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	// Create system profile (starts with _)
	systemDir := filepath.Join(vaultDir, "claude", "_original")
	if err := os.MkdirAll(systemDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(systemDir, "auth.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	exporter := &VaultExporter{
		VaultPath: vaultDir,
		DataPath:  tmpDir,
	}

	result, err := exporter.Export(&ExportOptions{
		OutputDir: filepath.Join(tmpDir, "output"),
		DryRun:    true,
	})
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	// Should only have user profile
	profiles := result.Manifest.Contents.Vault.Profiles["claude"]
	if len(profiles) != 1 || profiles[0] != "user@gmail.com" {
		t.Errorf("Should only include user profile, got %v", profiles)
	}
}

func TestVaultExporter_Export_VerboseFilename(t *testing.T) {
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")
	claudeDir := filepath.Join(vaultDir, "claude", "test@gmail.com")

	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "auth.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	exporter := &VaultExporter{
		VaultPath: vaultDir,
		DataPath:  tmpDir,
	}

	opts := &ExportOptions{
		OutputDir:       filepath.Join(tmpDir, "output"),
		VerboseFilename: true,
		DryRun:          true,
	}

	result, err := exporter.Export(opts)
	if err != nil {
		t.Fatalf("Export failed: %v", err)
	}

	filename := filepath.Base(result.OutputPath)
	if !strings.HasPrefix(filename, "Exported_Coding_Agent_Account_Auth_Info") {
		t.Errorf("Verbose filename should start with expected prefix, got %s", filename)
	}
}

func TestVaultExporter_Export_NoVault(t *testing.T) {
	tmpDir := t.TempDir()

	exporter := &VaultExporter{
		VaultPath: filepath.Join(tmpDir, "nonexistent"),
		DataPath:  tmpDir,
	}

	_, err := exporter.Export(&ExportOptions{
		OutputDir: tmpDir,
	})

	if err == nil {
		t.Error("Should fail with nonexistent vault")
	}
}

func TestVaultExporter_Export_EmptyVault(t *testing.T) {
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")

	if err := os.MkdirAll(vaultDir, 0700); err != nil {
		t.Fatal(err)
	}

	exporter := &VaultExporter{
		VaultPath: vaultDir,
		DataPath:  tmpDir,
	}

	_, err := exporter.Export(&ExportOptions{
		OutputDir: tmpDir,
	})

	if err == nil {
		t.Error("Should fail with empty vault")
	}
}

func TestVaultExporter_Export_EncryptionRequiresPassword(t *testing.T) {
	tmpDir := t.TempDir()
	vaultDir := filepath.Join(tmpDir, "vault")
	claudeDir := filepath.Join(vaultDir, "claude", "test@gmail.com")

	if err := os.MkdirAll(claudeDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(claudeDir, "auth.json"), []byte("{}"), 0600); err != nil {
		t.Fatal(err)
	}

	exporter := &VaultExporter{
		VaultPath: vaultDir,
		DataPath:  tmpDir,
	}

	_, err := exporter.Export(&ExportOptions{
		OutputDir: tmpDir,
		Encrypt:   true,
		Password:  "", // No password!
	})

	if err == nil {
		t.Error("Should fail when encryption enabled without password")
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0 B"},
		{100, "100 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
	}

	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}

func TestContains(t *testing.T) {
	slice := []string{"claude", "codex", "gemini"}

	if !contains(slice, "claude") {
		t.Error("Should find claude")
	}
	if !contains(slice, "CLAUDE") {
		t.Error("Should find CLAUDE (case insensitive)")
	}
	if contains(slice, "unknown") {
		t.Error("Should not find unknown")
	}
}

func TestMatchesAny(t *testing.T) {
	patterns := []string{"alice", "bob"}

	if !matchesAny("alice@gmail.com", patterns) {
		t.Error("Should match alice pattern")
	}
	if !matchesAny("ALICE@gmail.com", patterns) {
		t.Error("Should match ALICE pattern (case insensitive)")
	}
	if matchesAny("charlie@gmail.com", patterns) {
		t.Error("Should not match charlie")
	}
}

func TestCreateZipFromDir(t *testing.T) {
	tmpDir := t.TempDir()

	// Create some files
	subDir := filepath.Join(tmpDir, "subdir")
	if err := os.MkdirAll(subDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(tmpDir, "file1.txt"), []byte("content1"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "file2.txt"), []byte("content2"), 0600); err != nil {
		t.Fatal(err)
	}

	zipPath := filepath.Join(tmpDir, "test.zip")
	if err := createZipFromDir(tmpDir, zipPath); err != nil {
		t.Fatalf("createZipFromDir failed: %v", err)
	}

	// Verify zip contents
	reader, err := zip.OpenReader(zipPath)
	if err != nil {
		t.Fatalf("Failed to open zip: %v", err)
	}
	defer reader.Close()

	fileNames := make(map[string]bool)
	for _, f := range reader.File {
		fileNames[f.Name] = true
	}

	if !fileNames["file1.txt"] {
		t.Error("file1.txt should be in zip")
	}
	if !fileNames["subdir/file2.txt"] {
		t.Error("subdir/file2.txt should be in zip")
	}
}
