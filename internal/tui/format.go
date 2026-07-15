package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"
)

func fmtTime(sec int) string {
	if sec < 0 {
		sec = 0
	}
	if sec >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", sec/3600, sec%3600/60, sec%60)
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

// progressBar renders the played portion in the accent color with a ● knob at
// the playhead and dims the remainder. Without a duration (streams not yet
// probed) it renders an empty dimmed track, knob included only when seekable.
func progressBar(pos, dur, width int) string {
	if width <= 0 {
		return ""
	}
	if dur <= 0 {
		return dimStyle.Render(strings.Repeat("─", width))
	}
	knob := pos * width / dur
	if knob > width-1 {
		knob = width - 1
	}
	if knob < 0 {
		knob = 0
	}
	return accentStyle.Render(strings.Repeat("━", knob)+"●") +
		dimStyle.Render(strings.Repeat("─", width-knob-1))
}

// truncate limits s to max terminal cells (not runes: emoji and CJK are wide).
// Lines that overflow their box wrap, destabilizing the inline render height.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	return ansi.Truncate(s, max, "…")
}
