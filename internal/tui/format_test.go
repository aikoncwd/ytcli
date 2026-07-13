package tui

import "testing"

func TestFmtTime(t *testing.T) {
	cases := map[int]string{0: "0:00", 7: "0:07", 67: "1:07", 3599: "59:59", -5: "0:00"}
	for in, want := range cases {
		if got := fmtTime(in); got != want {
			t.Fatalf("fmtTime(%d) = %q; want %q", in, got, want)
		}
	}
}

func TestProgressBar(t *testing.T) {
	if got := progressBar(0, 100, 10); got != "░░░░░░░░░░" {
		t.Fatalf("empty bar = %q", got)
	}
	if got := progressBar(50, 100, 10); got != "▓▓▓▓▓░░░░░" {
		t.Fatalf("half bar = %q", got)
	}
	if got := progressBar(100, 100, 10); got != "▓▓▓▓▓▓▓▓▓▓" {
		t.Fatalf("full bar = %q", got)
	}
	if got := progressBar(10, 0, 10); got != "░░░░░░░░░░" {
		t.Fatalf("zero-duration bar = %q", got)
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
