package tui

import "github.com/charmbracelet/bubbles/key"

// keyMap defines all keybindings for the TUI.
type keyMap struct {
	// Navigation
	Up    key.Binding
	Down  key.Binding
	Left  key.Binding
	Right key.Binding
	Tab   key.Binding

	// Actions
	Enter  key.Binding
	Backup key.Binding
	Delete key.Binding

	// General
	Help key.Binding
	Quit key.Binding
}

// defaultKeyMap returns the default keybindings.
func defaultKeyMap() keyMap {
	return keyMap{
		Up: key.NewBinding(
			key.WithKeys("up", "k"),
			key.WithHelp("↑/k", "move up"),
		),
		Down: key.NewBinding(
			key.WithKeys("down", "j"),
			key.WithHelp("↓/j", "move down"),
		),
		Left: key.NewBinding(
			key.WithKeys("left", "h"),
			key.WithHelp("←/h", "previous provider"),
		),
		Right: key.NewBinding(
			key.WithKeys("right", "l"),
			key.WithHelp("→/l", "next provider"),
		),
		Tab: key.NewBinding(
			key.WithKeys("tab"),
			key.WithHelp("tab", "cycle providers"),
		),
		Enter: key.NewBinding(
			key.WithKeys("enter"),
			key.WithHelp("enter", "activate profile"),
		),
		Backup: key.NewBinding(
			key.WithKeys("b"),
			key.WithHelp("b", "backup current auth"),
		),
		Delete: key.NewBinding(
			key.WithKeys("d"),
			key.WithHelp("d", "delete profile"),
		),
		Help: key.NewBinding(
			key.WithKeys("?"),
			key.WithHelp("?", "toggle help"),
		),
		Quit: key.NewBinding(
			key.WithKeys("q", "esc", "ctrl+c"),
			key.WithHelp("q", "quit"),
		),
	}
}

// ShortHelp returns keybindings for the short help view.
func (k keyMap) ShortHelp() []key.Binding {
	return []key.Binding{k.Help, k.Quit}
}

// FullHelp returns keybindings for the full help view.
func (k keyMap) FullHelp() [][]key.Binding {
	return [][]key.Binding{
		{k.Up, k.Down, k.Left, k.Right},
		{k.Enter, k.Backup, k.Delete},
		{k.Help, k.Quit},
	}
}
