package cmd

import (
	"testing"
	"time"

	"github.com/Dicklesworthstone/coding_agent_account_manager/internal/sync"
)

// Test helper functions that are exported

func TestGetStatusIcon(t *testing.T) {
	tests := []struct {
		status   string
		expected string
	}{
		{sync.StatusOnline, "🟢"},
		{sync.StatusOffline, "🔴"},
		{sync.StatusSyncing, "🔄"},
		{sync.StatusError, "⚠️"},
		{sync.StatusUnknown, "⚪"},
		{"other", "⚪"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			got := getStatusIcon(tt.status)
			if got != tt.expected {
				t.Errorf("getStatusIcon(%q) = %q, want %q", tt.status, got, tt.expected)
			}
		})
	}
}
