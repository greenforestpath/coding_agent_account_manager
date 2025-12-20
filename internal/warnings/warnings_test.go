package warnings

import (
	"bytes"
	"testing"
	"time"
)

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{LevelInfo, "info"},
		{LevelWarning, "warning"},
		{LevelCritical, "critical"},
		{Level(99), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level.String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		d    time.Duration
		want string
	}{
		{30 * time.Second, "less than a minute"},
		{1 * time.Minute, "1 minute"},
		{45 * time.Minute, "45 minutes"},
		{1 * time.Hour, "1 hour"},
		{2 * time.Hour, "2 hours"},
		{23 * time.Hour, "23 hours"},
		{24 * time.Hour, "1 day"},
		{48 * time.Hour, "2 days"},
		{7 * 24 * time.Hour, "7 days"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := formatDuration(tt.d); got != tt.want {
				t.Errorf("formatDuration(%v) = %q, want %q", tt.d, got, tt.want)
			}
		})
	}
}

func TestPrint(t *testing.T) {
	tests := []struct {
		name     string
		warnings []Warning
		noColor  bool
		wantPrinted bool
	}{
		{
			name:        "empty warnings",
			warnings:    []Warning{},
			noColor:     true,
			wantPrinted: false,
		},
		{
			name: "single warning",
			warnings: []Warning{
				{
					Level:   LevelWarning,
					Tool:    "claude",
					Profile: "work",
					Message: "Token expires in 2 hours",
					Action:  "caam refresh claude work",
				},
			},
			noColor:     true,
			wantPrinted: true,
		},
		{
			name: "critical warning",
			warnings: []Warning{
				{
					Level:   LevelCritical,
					Tool:    "codex",
					Profile: "personal",
					Message: "Token EXPIRED",
					Action:  "caam login codex personal",
				},
			},
			noColor:     true,
			wantPrinted: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			got := Print(&buf, tt.warnings, tt.noColor)
			if got != tt.wantPrinted {
				t.Errorf("Print() = %v, want %v", got, tt.wantPrinted)
			}

			if tt.wantPrinted && buf.Len() == 0 {
				t.Error("Print() returned true but wrote nothing")
			}

			if !tt.wantPrinted && buf.Len() > 0 {
				t.Errorf("Print() returned false but wrote: %s", buf.String())
			}
		})
	}
}

func TestPrintOutput(t *testing.T) {
	warnings := []Warning{
		{
			Level:   LevelWarning,
			Tool:    "claude",
			Profile: "work",
			Message: "Token expires in 2 hours",
			Action:  "caam refresh claude work",
		},
	}

	var buf bytes.Buffer
	Print(&buf, warnings, true)

	output := buf.String()
	if output == "" {
		t.Fatal("Print() wrote nothing")
	}

	// Check key parts are present
	checks := []string{
		"claude/work",
		"Token expires in 2 hours",
		"caam refresh claude work",
	}

	for _, check := range checks {
		if !bytes.Contains(buf.Bytes(), []byte(check)) {
			t.Errorf("Print() output missing %q\nGot: %s", check, output)
		}
	}
}

func TestPrintWithColor(t *testing.T) {
	warnings := []Warning{
		{
			Level:   LevelCritical,
			Tool:    "codex",
			Profile: "test",
			Message: "Token EXPIRED",
		},
	}

	var buf bytes.Buffer
	Print(&buf, warnings, false) // With color

	// Should contain ANSI color codes
	if !bytes.Contains(buf.Bytes(), []byte("\033[31m")) {
		t.Error("Print() with color should contain red ANSI code for critical")
	}
	if !bytes.Contains(buf.Bytes(), []byte("\033[0m")) {
		t.Error("Print() with color should contain reset ANSI code")
	}

	// But without color should not
	buf.Reset()
	Print(&buf, warnings, true) // No color

	if bytes.Contains(buf.Bytes(), []byte("\033[")) {
		t.Error("Print() without color should not contain ANSI codes")
	}
}

func TestFilter(t *testing.T) {
	warnings := []Warning{
		{Level: LevelInfo, Message: "info1"},
		{Level: LevelWarning, Message: "warn1"},
		{Level: LevelCritical, Message: "crit1"},
		{Level: LevelInfo, Message: "info2"},
		{Level: LevelWarning, Message: "warn2"},
	}

	tests := []struct {
		minLevel Level
		wantLen  int
	}{
		{LevelInfo, 5},
		{LevelWarning, 3},
		{LevelCritical, 1},
	}

	for _, tt := range tests {
		t.Run(tt.minLevel.String(), func(t *testing.T) {
			got := Filter(warnings, tt.minLevel)
			if len(got) != tt.wantLen {
				t.Errorf("Filter() returned %d warnings, want %d", len(got), tt.wantLen)
			}
		})
	}
}

func TestWarningStruct(t *testing.T) {
	w := Warning{
		Level:   LevelWarning,
		Tool:    "claude",
		Profile: "work",
		Message: "Token expires soon",
		Action:  "caam refresh claude work",
	}

	if w.Level != LevelWarning {
		t.Errorf("Level = %v, want %v", w.Level, LevelWarning)
	}
	if w.Tool != "claude" {
		t.Errorf("Tool = %q, want %q", w.Tool, "claude")
	}
	if w.Profile != "work" {
		t.Errorf("Profile = %q, want %q", w.Profile, "work")
	}
	if w.Message != "Token expires soon" {
		t.Errorf("Message = %q, want %q", w.Message, "Token expires soon")
	}
	if w.Action != "caam refresh claude work" {
		t.Errorf("Action = %q, want %q", w.Action, "caam refresh claude work")
	}
}
