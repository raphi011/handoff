package util

import (
	"fmt"
	"time"
)

func FormatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)

	if diff < 0 {
		diff = -diff
	}

	if diff < time.Minute {
		seconds := int(diff.Seconds())
		return fmt.Sprintf("%d s ago", seconds)
	}

	if diff < time.Hour {
		minutes := int(diff.Minutes())
		return fmt.Sprintf("%d min ago", minutes)
	}

	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%d h ago", hours)
	}

	if diff < 7*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%d day%s ago", days, pluralize(days))
	}

	return t.Format("Jan 2")
}

func pluralize(n int) string {
	if n > 1 {
		return "s"
	}
	return ""
}
