package tui

import (
	"os"
	"testing"

	"github.com/charmbracelet/bubbles/progress"
	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProgressOptionsFromEnv(t *testing.T) {
	envKeys := []string{
		"NO_COLOR",
		"TERM",
		"REDUCED_MOTION",
		"CAAM_TUI_REDUCED_MOTION",
		"CAAM_REDUCED_MOTION",
		"REDUCE_MOTION",
		"CAAM_REDUCE_MOTION",
	}
	orig := make(map[string]string)
	origSet := make(map[string]bool)
	for _, key := range envKeys {
		if val, ok := os.LookupEnv(key); ok {
			orig[key] = val
			origSet[key] = true
		}
	}
	defer func() {
		for _, key := range envKeys {
			if origSet[key] {
				os.Setenv(key, orig[key])
			} else {
				os.Unsetenv(key)
			}
		}
	}()

	tests := []struct {
		name               string
		envVars            map[string]string
		expectNoColor      bool
		expectReduceMotion bool
	}{
		{
			name:               "default (no env vars)",
			envVars:            map[string]string{},
			expectNoColor:      false,
			expectReduceMotion: false,
		},
		{
			name:               "NO_COLOR set",
			envVars:            map[string]string{"NO_COLOR": "1"},
			expectNoColor:      true,
			expectReduceMotion: false,
		},
		{
			name:               "TERM=dumb",
			envVars:            map[string]string{"TERM": "dumb"},
			expectNoColor:      true,
			expectReduceMotion: false,
		},
		{
			name:               "REDUCE_MOTION set",
			envVars:            map[string]string{"REDUCE_MOTION": "1"},
			expectNoColor:      false,
			expectReduceMotion: true,
		},
		{
			name:               "REDUCED_MOTION set",
			envVars:            map[string]string{"REDUCED_MOTION": "1"},
			expectNoColor:      false,
			expectReduceMotion: true,
		},
		{
			name:               "CAAM_TUI_REDUCED_MOTION set",
			envVars:            map[string]string{"CAAM_TUI_REDUCED_MOTION": "true"},
			expectNoColor:      false,
			expectReduceMotion: true,
		},
		{
			name:               "both NO_COLOR and REDUCE_MOTION",
			envVars:            map[string]string{"NO_COLOR": "1", "REDUCE_MOTION": "1"},
			expectNoColor:      true,
			expectReduceMotion: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all relevant env vars
			for _, key := range envKeys {
				os.Unsetenv(key)
			}

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
			}

			opts := ProgressOptionsFromEnv()
			assert.Equal(t, tt.expectNoColor, opts.NoColor, "NoColor mismatch")
			assert.Equal(t, tt.expectReduceMotion, opts.ReduceMotion, "ReduceMotion mismatch")
		})
	}
}

func TestNewProgress(t *testing.T) {
	tests := []struct {
		name    string
		opts    ProgressOptions
		checkFn func(t *testing.T, p *Progress)
	}{
		{
			name: "default progress bar",
			opts: ProgressOptions{
				Width: 40,
			},
			checkFn: func(t *testing.T, p *Progress) {
				require.NotNil(t, p)
				assert.Equal(t, 40, p.width)
				assert.False(t, p.noColor)
				assert.False(t, p.reduceMotion)
				assert.True(t, p.IsAnimated())
			},
		},
		{
			name: "progress with NoColor",
			opts: ProgressOptions{
				Width:   30,
				NoColor: true,
			},
			checkFn: func(t *testing.T, p *Progress) {
				require.NotNil(t, p)
				assert.True(t, p.noColor)
				assert.True(t, p.IsAnimated())
			},
		},
		{
			name: "progress with ReduceMotion",
			opts: ProgressOptions{
				Width:        40,
				ReduceMotion: true,
			},
			checkFn: func(t *testing.T, p *Progress) {
				require.NotNil(t, p)
				assert.True(t, p.reduceMotion)
				assert.False(t, p.IsAnimated())
			},
		},
		{
			name: "progress with custom color",
			opts: ProgressOptions{
				Width: 50,
				Color: lipgloss.Color("#ff0000"),
			},
			checkFn: func(t *testing.T, p *Progress) {
				require.NotNil(t, p)
				assert.Equal(t, 50, p.width)
				assert.True(t, p.IsAnimated())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProgress(tt.opts)
			tt.checkFn(t, p)
		})
	}
}

func TestNewProgressWithTheme(t *testing.T) {
	tests := []struct {
		name       string
		themeOpts  ThemeOptions
		width      int
		expectAnim bool
	}{
		{
			name:       "default theme",
			themeOpts:  DefaultThemeOptions(),
			width:      40,
			expectAnim: true,
		},
		{
			name:       "NoColor theme",
			themeOpts:  ThemeOptions{NoColor: true},
			width:      30,
			expectAnim: true,
		},
		{
			name:       "Reduced motion theme",
			themeOpts:  ThemeOptions{ReducedMotion: true},
			width:      40,
			expectAnim: false,
		},
		{
			name:       "high contrast theme",
			themeOpts:  ThemeOptions{Contrast: ContrastHigh},
			width:      50,
			expectAnim: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			theme := NewTheme(tt.themeOpts)
			p := NewProgressWithTheme(theme, tt.width)

			require.NotNil(t, p)
			assert.Equal(t, tt.width, p.width)
			assert.Equal(t, tt.expectAnim, p.IsAnimated())
		})
	}
}

func TestProgressSetPercent(t *testing.T) {
	tests := []struct {
		name         string
		opts         ProgressOptions
		percent      float64
		expectCmd    bool
		expectStored float64
	}{
		{
			name:         "animated progress returns command",
			opts:         ProgressOptions{Width: 40},
			percent:      0.5,
			expectCmd:    true,
			expectStored: 0.5,
		},
		{
			name:         "reduced motion returns nil",
			opts:         ProgressOptions{Width: 40, ReduceMotion: true},
			percent:      0.75,
			expectCmd:    false,
			expectStored: 0.75,
		},
		{
			name:         "clamps to 0",
			opts:         ProgressOptions{Width: 40},
			percent:      -0.5,
			expectCmd:    true,
			expectStored: 0,
		},
		{
			name:         "clamps to 1",
			opts:         ProgressOptions{Width: 40},
			percent:      1.5,
			expectCmd:    true,
			expectStored: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProgress(tt.opts)
			cmd := p.SetPercent(tt.percent)

			if tt.expectCmd {
				assert.NotNil(t, cmd, "expected SetPercent to return a command")
			} else {
				assert.Nil(t, cmd, "expected SetPercent to return nil")
			}

			assert.Equal(t, tt.expectStored, p.Percent())
		})
	}
}

func TestProgressSetPercentInstant(t *testing.T) {
	p := NewProgress(ProgressOptions{Width: 40})

	p.SetPercentInstant(0.5)
	assert.Equal(t, 0.5, p.Percent())

	p.SetPercentInstant(0.8)
	assert.Equal(t, 0.8, p.Percent())

	// Test clamping
	p.SetPercentInstant(-1)
	assert.Equal(t, 0.0, p.Percent())

	p.SetPercentInstant(2)
	assert.Equal(t, 1.0, p.Percent())
}

func TestProgressView(t *testing.T) {
	tests := []struct {
		name    string
		opts    ProgressOptions
		percent float64
	}{
		{
			name:    "renders at 0%",
			opts:    ProgressOptions{Width: 20},
			percent: 0,
		},
		{
			name:    "renders at 50%",
			opts:    ProgressOptions{Width: 20},
			percent: 0.5,
		},
		{
			name:    "renders at 100%",
			opts:    ProgressOptions{Width: 20},
			percent: 1.0,
		},
		{
			name:    "NoColor progress renders",
			opts:    ProgressOptions{Width: 20, NoColor: true},
			percent: 0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := NewProgress(tt.opts)
			p.SetPercentInstant(tt.percent)
			view := p.View()

			assert.NotEmpty(t, view, "progress view should not be empty")
		})
	}
}

func TestProgressViewWithPercent(t *testing.T) {
	p := NewProgress(ProgressOptions{Width: 20})
	p.SetPercentInstant(0.5)

	view := p.ViewWithPercent()
	assert.NotEmpty(t, view)
	// The view should contain the progress bar and percentage
	assert.Contains(t, view, "%")
}

func TestProgressSetWidth(t *testing.T) {
	p := NewProgress(ProgressOptions{Width: 40})
	assert.Equal(t, 40, p.Width())

	p.SetWidth(60)
	assert.Equal(t, 60, p.Width())

	// Setting width to 0 or negative should be ignored
	p.SetWidth(0)
	assert.Equal(t, 60, p.Width())

	p.SetWidth(-10)
	assert.Equal(t, 60, p.Width())
}

func TestProgressUpdate(t *testing.T) {
	t.Run("processes frame messages", func(t *testing.T) {
		p := NewProgress(ProgressOptions{Width: 40})
		require.NotNil(t, p)

		// Create a progress frame message
		frameMsg := progress.FrameMsg{}

		newP, _ := p.Update(frameMsg)
		assert.NotNil(t, newP)
	})

	t.Run("ignores other messages", func(t *testing.T) {
		p := NewProgress(ProgressOptions{Width: 40})
		require.NotNil(t, p)

		newP, cmd := p.Update("random message")
		assert.NotNil(t, newP)
		assert.Nil(t, cmd)
	})
}

func TestProgressNilSafety(t *testing.T) {
	var p *Progress

	// All methods should be nil-safe
	assert.NotPanics(t, func() { p.Init() })
	assert.NotPanics(t, func() { p.Update(nil) })
	assert.NotPanics(t, func() { p.View() })
	assert.NotPanics(t, func() { p.ViewWithPercent() })
	assert.NotPanics(t, func() { p.SetPercent(0.5) })
	assert.NotPanics(t, func() { p.SetPercentInstant(0.5) })
	assert.NotPanics(t, func() { p.Percent() })
	assert.NotPanics(t, func() { p.SetWidth(40) })
	assert.NotPanics(t, func() { p.Width() })
	assert.NotPanics(t, func() { p.IsAnimated() })

	assert.Equal(t, "", p.View())
	assert.Equal(t, "", p.ViewWithPercent())
	assert.Equal(t, 0.0, p.Percent())
	assert.Equal(t, 0, p.Width())
	assert.False(t, p.IsAnimated())
	assert.Nil(t, p.Init())
	assert.Nil(t, p.SetPercent(0.5))

	newP, cmd := p.Update(nil)
	assert.Nil(t, newP)
	assert.Nil(t, cmd)
}

func TestProgressWithLabel(t *testing.T) {
	theme := NewTheme(DefaultThemeOptions())

	t.Run("creates labeled progress", func(t *testing.T) {
		p := NewProgressWithLabel(theme, "Exporting:", 30)
		require.NotNil(t, p)
		assert.Equal(t, "Exporting:", p.Label)
		assert.NotNil(t, p.Progress)
	})

	t.Run("sets percent", func(t *testing.T) {
		p := NewProgressWithLabel(theme, "Loading:", 30)
		cmd := p.SetPercent(0.5)
		// Animated progress returns a command
		assert.NotNil(t, cmd)
		assert.Equal(t, 0.5, p.Progress.Percent())
	})

	t.Run("renders view with label", func(t *testing.T) {
		p := NewProgressWithLabel(theme, "Processing:", 30)
		p.Progress.SetPercentInstant(0.5)
		view := p.View()
		assert.Contains(t, view, "Processing:")
	})

	t.Run("handles nil", func(t *testing.T) {
		var p *ProgressWithLabel
		assert.NotPanics(t, func() { p.SetPercent(0.5) })
		assert.NotPanics(t, func() { p.View() })
		assert.NotPanics(t, func() { p.Update(nil) })
		assert.Equal(t, "", p.View())
	})
}

func TestProgressEnabled(t *testing.T) {
	t.Run("enabled with default theme", func(t *testing.T) {
		theme := NewTheme(DefaultThemeOptions())
		assert.True(t, progressEnabled(theme))
	})

	t.Run("disabled with reduced motion", func(t *testing.T) {
		theme := NewTheme(ThemeOptions{ReducedMotion: true})
		assert.False(t, progressEnabled(theme))
	})
}

func TestProgressStyle(t *testing.T) {
	t.Run("returns style with color", func(t *testing.T) {
		theme := NewTheme(DefaultThemeOptions())
		style := progressStyle(theme)
		assert.NotNil(t, style)
	})

	t.Run("returns empty style with NoColor", func(t *testing.T) {
		theme := NewTheme(ThemeOptions{NoColor: true})
		style := progressStyle(theme)
		assert.NotNil(t, style)
	})
}

func TestDefaultProgressOptions(t *testing.T) {
	opts := DefaultProgressOptions()
	assert.Equal(t, 40, opts.Width)
	assert.True(t, opts.ShowPercentage)
	assert.False(t, opts.NoColor)
	assert.False(t, opts.ReduceMotion)
}
