package store

import (
	"testing"

	"github.com/AikonCWD/ytcli/internal/track"
)

func tk(id string) track.Track { return track.Track{ID: id, Title: id, URL: "u" + id} }

func TestHistoryRoundTripAndDedup(t *testing.T) {
	s := New(t.TempDir())
	if err := s.AppendHistory(tk("a")); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendHistory(tk("b")); err != nil {
		t.Fatal(err)
	}
	if err := s.AppendHistory(tk("a")); err != nil { // move a to front
		t.Fatal(err)
	}
	h, err := s.LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 2 || h[0].ID != "a" || h[1].ID != "b" {
		t.Fatalf("history = %+v; want [a b]", h)
	}
}

func TestHistoryEmptyWhenMissing(t *testing.T) {
	s := New(t.TempDir())
	h, err := s.LoadHistory()
	if err != nil {
		t.Fatal(err)
	}
	if len(h) != 0 {
		t.Fatalf("want empty history, got %v", h)
	}
}

func TestHistoryCap(t *testing.T) {
	s := New(t.TempDir())
	for i := 0; i < historyCap+50; i++ {
		s.AppendHistory(track.Track{ID: string(rune('A'+i%26)) + string(rune(i))})
	}
	h, _ := s.LoadHistory()
	if len(h) > historyCap {
		t.Fatalf("history len = %d; want <= %d", len(h), historyCap)
	}
}

func TestToggleFavorite(t *testing.T) {
	s := New(t.TempDir())
	on, err := s.ToggleFavorite(tk("a"))
	if err != nil || !on {
		t.Fatalf("first toggle = %v,%v; want true,nil", on, err)
	}
	fav, _ := s.IsFavorite("a")
	if !fav {
		t.Fatal("a should be favorite")
	}
	off, _ := s.ToggleFavorite(tk("a"))
	if off {
		t.Fatal("second toggle should turn favorite off")
	}
	fav, _ = s.IsFavorite("a")
	if fav {
		t.Fatal("a should no longer be favorite")
	}
}
