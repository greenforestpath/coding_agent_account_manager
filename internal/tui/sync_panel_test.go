package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestSyncPanel_View_Empty(t *testing.T) {
	p := NewSyncPanel()
	p.SetSize(120, 40)

	out := p.View()
	if out == "" {
		t.Fatalf("View() returned empty")
	}
	if want := "No machines"; !strings.Contains(out, want) {
		t.Fatalf("View() output missing %q, got: %s", want, out)
	}
}

func TestSyncPanel_Toggle(t *testing.T) {
	p := NewSyncPanel()

	if p.Visible() {
		t.Fatalf("panel should not be visible initially")
	}

	p.Toggle()
	if !p.Visible() {
		t.Fatalf("panel should be visible after toggle")
	}

	p.Toggle()
	if p.Visible() {
		t.Fatalf("panel should not be visible after second toggle")
	}
}

func TestSyncPanel_Navigation(t *testing.T) {
	p := NewSyncPanel()

	// Can't move up/down with no machines
	p.MoveUp()
	p.MoveDown()
	// Should not panic
}

func TestModel_SyncPanel_ToggleWithKey(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("CAAM_HOME", tmpDir)

	m := New()
	m.width = 120
	m.height = 40

	// Toggle on with 'S'
	model, _ := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("S")})
	m = model.(Model)
	if m.syncPanel == nil || !m.syncPanel.Visible() {
		t.Fatalf("sync panel not visible after toggle")
	}

	// Close with esc
	model, _ = m.Update(tea.KeyMsg{Type: tea.KeyEscape})
	m = model.(Model)
	if m.syncPanel.Visible() {
		t.Fatalf("sync panel still visible after esc")
	}
}

func TestSyncPanel_SetLoading(t *testing.T) {
	p := NewSyncPanel()
	p.SetSize(120, 40)
	p.SetLoading(true)

	out := p.View()
	if !strings.Contains(out, "Loading") {
		t.Fatalf("View() should show loading state, got: %s", out)
	}

	p.SetLoading(false)
	out = p.View()
	if strings.Contains(out, "Loading") {
		t.Fatalf("View() should not show loading after SetLoading(false)")
	}
}
