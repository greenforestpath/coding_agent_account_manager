package tui

import (
	"github.com/charmbracelet/bubbles/progress"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ProgressOptions configures progress bar behavior.
type ProgressOptions struct {
	// Width is the total width of the progress bar in characters.
	Width int
	// ShowPercentage displays percentage text next to the bar.
	ShowPercentage bool
	// Color is the filled portion color (ignored if NoColor).
	Color lipgloss.TerminalColor
	// NoColor disables colors.
	NoColor bool
	// ReduceMotion disables animated transitions (instant updates).
	ReduceMotion bool
}

// Progress wraps the bubbles progress bar with theme and accessibility support.
type Progress struct {
	progress     progress.Model
	percent      float64
	width        int
	noColor      bool
	reduceMotion bool
}

// DefaultProgressOptions provides baseline options.
func DefaultProgressOptions() ProgressOptions {
	return ProgressOptions{
		Width:          40,
		ShowPercentage: true,
	}
}

// ProgressOptionsFromEnv derives progress options from environment.
// Respects:
// - NO_COLOR (disables color output)
// - TERM=dumb (disables color output)
// - CAAM_TUI_REDUCED_MOTION / CAAM_REDUCED_MOTION / REDUCED_MOTION (disables animation)
func ProgressOptionsFromEnv() ProgressOptions {
	spinnerOpts := SpinnerOptionsFromEnv()
	return ProgressOptions{
		Width:          40,
		ShowPercentage: true,
		NoColor:        spinnerOpts.NoColor,
		ReduceMotion:   spinnerOpts.ReduceMotion,
	}
}

// NewProgress creates a progress bar with the given options.
func NewProgress(opts ProgressOptions) *Progress {
	p := progress.New(
		progress.WithDefaultGradient(),
		progress.WithoutPercentage(),
	)

	if opts.Width > 0 {
		p.Width = opts.Width
	}

	// Configure for no color mode
	if opts.NoColor {
		p.Full = '█'
		p.FullColor = ""
		p.Empty = '░'
		p.EmptyColor = ""
	} else if opts.Color != nil {
		// Use custom color
		if c, ok := opts.Color.(lipgloss.Color); ok {
			p.FullColor = string(c)
		}
	}

	// Disable spring animation if reduced motion is requested
	if opts.ReduceMotion {
		p.SetSpringOptions(0, 0)
	}

	return &Progress{
		progress:     p,
		width:        opts.Width,
		noColor:      opts.NoColor,
		reduceMotion: opts.ReduceMotion,
	}
}

// NewProgressWithTheme creates a progress bar using theme settings.
func NewProgressWithTheme(theme Theme, width int) *Progress {
	envOpts := ProgressOptionsFromEnv()

	opts := ProgressOptions{
		Width:          width,
		ShowPercentage: true,
		Color:          theme.Palette.Accent,
		NoColor:        theme.NoColor || envOpts.NoColor,
		ReduceMotion:   theme.ReducedMotion || envOpts.ReduceMotion,
	}

	p := NewProgress(opts)
	if p != nil && !opts.NoColor {
		// Apply theme gradient colors
		if acc, ok := theme.Palette.Accent.(lipgloss.Color); ok {
			if accAlt, ok := theme.Palette.AccentAlt.(lipgloss.Color); ok {
				p.progress.FullColor = string(acc)
				p.progress.EmptyColor = ""
				// Set gradient from accent to accent alt
				p.progress = progress.New(
					progress.WithGradient(string(acc), string(accAlt)),
					progress.WithoutPercentage(),
				)
				p.progress.Width = width
				if opts.ReduceMotion {
					p.progress.SetSpringOptions(0, 0)
				}
			}
		}
	}
	return p
}

// Init initializes the progress bar (required for bubbletea).
func (p *Progress) Init() tea.Cmd {
	if p == nil {
		return nil
	}
	return nil
}

// Update handles progress bar animation messages.
func (p *Progress) Update(msg tea.Msg) (*Progress, tea.Cmd) {
	if p == nil {
		return p, nil
	}

	// Handle progress frame messages for smooth animation
	if frameMsg, ok := msg.(progress.FrameMsg); ok {
		var cmd tea.Cmd
		progressModel, cmd := p.progress.Update(frameMsg)
		p.progress = progressModel.(progress.Model)
		return p, cmd
	}

	return p, nil
}

// SetPercent updates the progress percentage (0.0 to 1.0).
// Returns a command for animated transition (nil if reduced motion).
func (p *Progress) SetPercent(percent float64) tea.Cmd {
	if p == nil {
		return nil
	}

	// Clamp to valid range
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	p.percent = percent

	if p.reduceMotion {
		// Instant update, no animation
		p.progress.SetPercent(percent)
		return nil
	}

	// Return animated transition command
	return p.progress.SetPercent(percent)
}

// SetPercentInstant updates the progress instantly without animation.
func (p *Progress) SetPercentInstant(percent float64) {
	if p == nil {
		return
	}

	// Clamp to valid range
	if percent < 0 {
		percent = 0
	}
	if percent > 1 {
		percent = 1
	}

	p.percent = percent
	p.progress.SetPercent(percent)
}

// Percent returns the current progress percentage.
func (p *Progress) Percent() float64 {
	if p == nil {
		return 0
	}
	return p.percent
}

// View renders the progress bar.
func (p *Progress) View() string {
	if p == nil {
		return ""
	}
	return p.progress.View()
}

// ViewWithPercent renders the progress bar with percentage text.
func (p *Progress) ViewWithPercent() string {
	if p == nil {
		return ""
	}

	percentStr := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(formatPercent(p.percent))

	return p.progress.View() + " " + percentStr
}

// SetWidth updates the progress bar width.
func (p *Progress) SetWidth(width int) {
	if p == nil || width <= 0 {
		return
	}
	p.width = width
	p.progress.Width = width
}

// Width returns the current progress bar width.
func (p *Progress) Width() int {
	if p == nil {
		return 0
	}
	return p.width
}

// IsAnimated returns true if the progress bar uses animated transitions.
func (p *Progress) IsAnimated() bool {
	if p == nil {
		return false
	}
	return !p.reduceMotion
}

// formatPercent formats a 0-1 percentage as a string like "45%".
func formatPercent(p float64) string {
	return lipgloss.NewStyle().Render(
		lipgloss.NewStyle().Width(4).Align(lipgloss.Right).Render(
			string(rune('0'+int(p*100)/100)) +
				string(rune('0'+(int(p*100)/10)%10)) +
				string(rune('0'+int(p*100)%10)) + "%",
		),
	)
}

// ProgressWithLabel combines a label, progress bar, and percentage into a single line.
type ProgressWithLabel struct {
	Label    string
	Progress *Progress
	theme    Theme
}

// NewProgressWithLabel creates a labeled progress bar.
func NewProgressWithLabel(theme Theme, label string, width int) *ProgressWithLabel {
	return &ProgressWithLabel{
		Label:    label,
		Progress: NewProgressWithTheme(theme, width),
		theme:    theme,
	}
}

// SetPercent updates the progress percentage.
func (p *ProgressWithLabel) SetPercent(percent float64) tea.Cmd {
	if p == nil || p.Progress == nil {
		return nil
	}
	return p.Progress.SetPercent(percent)
}

// Update handles messages for the progress bar.
func (p *ProgressWithLabel) Update(msg tea.Msg) (*ProgressWithLabel, tea.Cmd) {
	if p == nil || p.Progress == nil {
		return p, nil
	}
	var cmd tea.Cmd
	p.Progress, cmd = p.Progress.Update(msg)
	return p, cmd
}

// View renders the labeled progress bar.
func (p *ProgressWithLabel) View() string {
	if p == nil || p.Progress == nil {
		return ""
	}

	labelStyle := lipgloss.NewStyle()
	if !p.theme.NoColor {
		labelStyle = labelStyle.Foreground(p.theme.Palette.Muted)
	}

	return labelStyle.Render(p.Label) + " " + p.Progress.ViewWithPercent()
}

// progressStyle returns the centralized progress bar styling for the current theme.
func progressStyle(theme Theme) lipgloss.Style {
	style := lipgloss.NewStyle()
	if theme.NoColor {
		return style
	}
	return style.Foreground(theme.Palette.Accent)
}

// progressEnabled returns true if animated progress is enabled.
func progressEnabled(theme Theme) bool {
	return !theme.ReducedMotion
}
