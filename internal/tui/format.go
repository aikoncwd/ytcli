package tui

import "fmt"

func fmtTime(sec int) string {
	if sec < 0 {
		sec = 0
	}
	return fmt.Sprintf("%d:%02d", sec/60, sec%60)
}

func progressBar(pos, dur, width int) string {
	if width <= 0 {
		return ""
	}
	filled := 0
	if dur > 0 {
		filled = pos * width / dur
		if filled > width {
			filled = width
		}
	}
	out := make([]rune, width)
	for i := 0; i < width; i++ {
		if i < filled {
			out[i] = '▓'
		} else {
			out[i] = '░'
		}
	}
	return string(out)
}

func truncate(s string, max int) string {
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 1 {
		return string(r[:max])
	}
	return string(r[:max-1]) + "…"
}
