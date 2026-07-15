package tui

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
)

func TestFmtTime(t *testing.T) {
	cases := map[int]string{0: "0:00", 7: "0:07", 67: "1:07", 3599: "59:59", -5: "0:00",
		3600: "1:00:00", 3725: "1:02:05", 73696: "20:28:16"}
	for in, want := range cases {
		if got := fmtTime(in); got != want {
			t.Fatalf("fmtTime(%d) = %q; want %q", in, got, want)
		}
	}
}

func TestTruncateCountsDisplayCellsNotRunes(t *testing.T) {
	// Each CJK char is 2 cells wide; 4 runes = 8 cells must not fit in 6.
	got := truncate("日本語テ", 6)
	if got == "日本語テ" {
		t.Fatalf("wide string should be truncated by cell width, got %q", got)
	}
}

// The bar carries color; assertions compare the ANSI-stripped glyphs and the
// visible width (which must always equal the requested width).
func TestProgressBar(t *testing.T) {
	if got := ansi.Strip(progressBar(0, 100, 10)); got != "●─────────" {
		t.Fatalf("empty bar = %q", got)
	}
	if got := ansi.Strip(progressBar(50, 100, 10)); got != "━━━━━●────" {
		t.Fatalf("half bar = %q", got)
	}
	if got := ansi.Strip(progressBar(100, 100, 10)); got != "━━━━━━━━━●" {
		t.Fatalf("full bar = %q", got)
	}
	if got := ansi.Strip(progressBar(10, 0, 10)); got != "──────────" {
		t.Fatalf("zero-duration bar = %q", got)
	}
	for _, pos := range []int{0, 50, 100, 200} {
		if w := ansi.StringWidth(progressBar(pos, 100, 10)); w != 10 {
			t.Fatalf("bar width at pos=%d is %d cells; want 10", pos, w)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("hello", 10); got != "hello" {
		t.Fatalf("truncate short = %q", got)
	}
	if got := truncate("hello world", 5); got != "hell…" {
		t.Fatalf("truncate long = %q; want hell…", got)
	}
	if got := truncate("hello", 1); got != "…" {
		t.Fatalf("truncate max=1 = %q; want …", got)
	}
	if got := truncate("hello", 0); got != "" {
		t.Fatalf("truncate max=0 = %q; want empty", got)
	}
	if got := truncate("hello", -3); got != "" {
		t.Fatalf("truncate negative max = %q; want empty", got)
	}
}
