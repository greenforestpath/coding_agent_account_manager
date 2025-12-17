package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNew(t *testing.T) {
	m := New()
	if len(m.providers) != 3 {
		t.Errorf("expected 3 providers, got %d", len(m.providers))
	}
	if m.activeProvider != 0 {
		t.Errorf("expected activeProvider 0, got %d", m.activeProvider)
	}
	if m.providerPanel == nil {
		t.Error("expected providerPanel to be initialized")
	}
}

func TestNewWithProviders(t *testing.T) {
	providers := []string{"test1", "test2"}
	m := NewWithProviders(providers)
	if len(m.providers) != 2 {
		t.Errorf("expected 2 providers, got %d", len(m.providers))
	}
}

func TestDefaultProviders(t *testing.T) {
	providers := DefaultProviders()
	expected := []string{"claude", "codex", "gemini"}
	if len(providers) != len(expected) {
		t.Errorf("expected %d providers, got %d", len(expected), len(providers))
	}
	for i, p := range providers {
		if p != expected[i] {
			t.Errorf("expected provider %s at index %d, got %s", expected[i], i, p)
		}
	}
}

func TestModelUpdate(t *testing.T) {
	m := New()

	// Test window size message
	model, _ := m.Update(tea.WindowSizeMsg{Width: 100, Height: 50})
	updated := model.(Model)
	if updated.width != 100 || updated.height != 50 {
		t.Errorf("expected dimensions 100x50, got %dx%d", updated.width, updated.height)
	}

	// Test profiles loaded message
	profiles := map[string][]Profile{
		"claude": {{Name: "test@example.com", Provider: "claude", IsActive: true}},
	}
	model, _ = updated.Update(profilesLoadedMsg{profiles: profiles})
	updated = model.(Model)
	if len(updated.profiles["claude"]) != 1 {
		t.Errorf("expected 1 claude profile, got %d", len(updated.profiles["claude"]))
	}
}

func TestCurrentProfiles(t *testing.T) {
	m := New()
	m.profiles = map[string][]Profile{
		"claude": {{Name: "a@b.com"}},
		"codex":  {{Name: "c@d.com"}, {Name: "e@f.com"}},
	}

	profiles := m.currentProfiles()
	if len(profiles) != 1 {
		t.Errorf("expected 1 profile for claude, got %d", len(profiles))
	}

	m.activeProvider = 1
	profiles = m.currentProfiles()
	if len(profiles) != 2 {
		t.Errorf("expected 2 profiles for codex, got %d", len(profiles))
	}
}

func TestCurrentProvider(t *testing.T) {
	m := New()
	if m.currentProvider() != "claude" {
		t.Errorf("expected claude, got %s", m.currentProvider())
	}
	m.activeProvider = 2
	if m.currentProvider() != "gemini" {
		t.Errorf("expected gemini, got %s", m.currentProvider())
	}
}

func TestProviderPanelView(t *testing.T) {
	p := NewProviderPanel(DefaultProviders())
	p.SetProfileCounts(map[string]int{"claude": 2, "codex": 1, "gemini": 0})
	p.SetActiveProvider(0)

	view := p.View()
	if view == "" {
		t.Error("expected non-empty view")
	}
}

func TestCapitalizeFirst(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"claude", "Claude"},
		{"Codex", "Codex"},
		{"", ""},
		{"gemini", "Gemini"},
	}

	for _, tc := range tests {
		result := capitalizeFirst(tc.input)
		if result != tc.expected {
			t.Errorf("capitalizeFirst(%q) = %q, expected %q", tc.input, result, tc.expected)
		}
	}
}
