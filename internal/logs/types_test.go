package logs

import (
	"testing"
	"time"
)

func TestLogEntry_CalculateTotalTokens(t *testing.T) {
	entry := &LogEntry{
		InputTokens:       100,
		OutputTokens:      200,
		CacheReadTokens:   50,
		CacheCreateTokens: 25,
	}

	total := entry.CalculateTotalTokens()
	if total != 375 {
		t.Errorf("CalculateTotalTokens() = %d, want 375", total)
	}
}

func TestTokenUsage_Add(t *testing.T) {
	usage := NewTokenUsage()

	// Add first entry
	usage.Add(&LogEntry{
		Model:        "claude-3-opus",
		InputTokens:  100,
		OutputTokens: 200,
	})

	if usage.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", usage.InputTokens)
	}
	if usage.OutputTokens != 200 {
		t.Errorf("OutputTokens = %d, want 200", usage.OutputTokens)
	}
	if usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", usage.TotalTokens)
	}

	// Add second entry with same model
	usage.Add(&LogEntry{
		Model:        "claude-3-opus",
		InputTokens:  50,
		OutputTokens: 100,
	})

	if usage.TotalTokens != 450 {
		t.Errorf("TotalTokens after second add = %d, want 450", usage.TotalTokens)
	}

	// Check model breakdown
	modelUsage, ok := usage.ByModel["claude-3-opus"]
	if !ok {
		t.Fatal("Model usage for claude-3-opus not found")
	}
	if modelUsage.InputTokens != 150 {
		t.Errorf("Model InputTokens = %d, want 150", modelUsage.InputTokens)
	}
	if modelUsage.TotalTokens != 450 {
		t.Errorf("Model TotalTokens = %d, want 450", modelUsage.TotalTokens)
	}
}

func TestTokenUsage_MultipleModels(t *testing.T) {
	usage := NewTokenUsage()

	usage.Add(&LogEntry{
		Model:        "claude-3-opus",
		InputTokens:  100,
		OutputTokens: 200,
	})
	usage.Add(&LogEntry{
		Model:        "claude-3-sonnet",
		InputTokens:  50,
		OutputTokens: 100,
	})

	if len(usage.ByModel) != 2 {
		t.Errorf("ByModel has %d entries, want 2", len(usage.ByModel))
	}

	opus := usage.ByModel["claude-3-opus"]
	if opus.TotalTokens != 300 {
		t.Errorf("opus TotalTokens = %d, want 300", opus.TotalTokens)
	}

	sonnet := usage.ByModel["claude-3-sonnet"]
	if sonnet.TotalTokens != 150 {
		t.Errorf("sonnet TotalTokens = %d, want 150", sonnet.TotalTokens)
	}
}

func TestScanResult_TokenUsage(t *testing.T) {
	result := &ScanResult{
		Provider: "claude",
		Entries: []*LogEntry{
			{Model: "claude-3-opus", InputTokens: 100, OutputTokens: 200},
			{Model: "claude-3-opus", InputTokens: 50, OutputTokens: 100},
		},
	}

	usage := result.TokenUsage()
	if usage.TotalTokens != 450 {
		t.Errorf("TokenUsage().TotalTokens = %d, want 450", usage.TotalTokens)
	}
}

func TestScanResult_FilterByModel(t *testing.T) {
	result := &ScanResult{
		Provider: "claude",
		Entries: []*LogEntry{
			{Model: "claude-3-opus", InputTokens: 100},
			{Model: "claude-3-sonnet", InputTokens: 50},
			{Model: "claude-3-opus", InputTokens: 75},
		},
	}

	opus := result.FilterByModel("claude-3-opus")
	if len(opus) != 2 {
		t.Errorf("FilterByModel(opus) returned %d entries, want 2", len(opus))
	}

	sonnet := result.FilterByModel("claude-3-sonnet")
	if len(sonnet) != 1 {
		t.Errorf("FilterByModel(sonnet) returned %d entries, want 1", len(sonnet))
	}
}

func TestScanResult_FilterByType(t *testing.T) {
	result := &ScanResult{
		Provider: "claude",
		Entries: []*LogEntry{
			{Type: "response", InputTokens: 100},
			{Type: "request", InputTokens: 50},
			{Type: "response", InputTokens: 75},
		},
	}

	responses := result.FilterByType("response")
	if len(responses) != 2 {
		t.Errorf("FilterByType(response) returned %d entries, want 2", len(responses))
	}
}

func TestScanResult_Models(t *testing.T) {
	result := &ScanResult{
		Provider: "claude",
		Entries: []*LogEntry{
			{Model: "claude-3-opus"},
			{Model: "claude-3-sonnet"},
			{Model: "claude-3-opus"}, // duplicate
			{Model: ""},              // empty
		},
	}

	models := result.Models()
	if len(models) != 2 {
		t.Errorf("Models() returned %d models, want 2", len(models))
	}

	// Check both models are present
	found := make(map[string]bool)
	for _, m := range models {
		found[m] = true
	}
	if !found["claude-3-opus"] || !found["claude-3-sonnet"] {
		t.Errorf("Models() = %v, want [claude-3-opus, claude-3-sonnet]", models)
	}
}

func TestNewTokenUsage(t *testing.T) {
	usage := NewTokenUsage()
	if usage == nil {
		t.Fatal("NewTokenUsage() returned nil")
	}
	if usage.ByModel == nil {
		t.Error("NewTokenUsage().ByModel is nil")
	}
}

func TestLogEntry_ZeroValues(t *testing.T) {
	entry := &LogEntry{}

	// Should handle zero values gracefully
	if entry.CalculateTotalTokens() != 0 {
		t.Errorf("Empty entry CalculateTotalTokens() = %d, want 0", entry.CalculateTotalTokens())
	}
}

func TestTokenUsage_EmptyModel(t *testing.T) {
	usage := NewTokenUsage()

	// Add entry with empty model
	usage.Add(&LogEntry{
		Model:        "",
		InputTokens:  100,
		OutputTokens: 200,
	})

	// Total should still be tracked
	if usage.TotalTokens != 300 {
		t.Errorf("TotalTokens = %d, want 300", usage.TotalTokens)
	}

	// But ByModel should be empty since model was empty
	if len(usage.ByModel) != 0 {
		t.Errorf("ByModel has %d entries, want 0 for empty model", len(usage.ByModel))
	}
}

func TestScanResult_Empty(t *testing.T) {
	result := &ScanResult{
		Provider: "claude",
		Entries:  nil,
	}

	usage := result.TokenUsage()
	if usage.TotalTokens != 0 {
		t.Errorf("Empty result TokenUsage().TotalTokens = %d, want 0", usage.TotalTokens)
	}

	models := result.Models()
	if len(models) != 0 {
		t.Errorf("Empty result Models() has %d entries, want 0", len(models))
	}
}

func TestScanResult_TimeWindow(t *testing.T) {
	now := time.Now()
	since := now.Add(-1 * time.Hour)

	result := &ScanResult{
		Provider: "claude",
		Since:    since,
		Until:    now,
	}

	if result.Since != since {
		t.Errorf("Since = %v, want %v", result.Since, since)
	}
	if result.Until != now {
		t.Errorf("Until = %v, want %v", result.Until, now)
	}
}
