package cmd

import (
	"fmt"
	"time"
)

// formatDurationShort renders a duration with compact hours/minutes for CLI output.
func formatDurationShort(d time.Duration) string {
	if d <= 0 {
		return "0m"
	}

	d = d.Round(time.Minute)
	if d < time.Minute {
		return "<1m"
	}

	hours := int(d.Hours())
	mins := int(d.Minutes()) % 60
	switch {
	case hours <= 0:
		return fmt.Sprintf("%dm", mins)
	case mins == 0:
		return fmt.Sprintf("%dh", hours)
	default:
		return fmt.Sprintf("%dh%dm", hours, mins)
	}
}
