package tui

import (
	"strings"
	"testing"

	"github.com/AikonCWD/ytcli/internal/player"
	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/track"
)

func TestCompactViewShowsTitleAndTime(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "Mi Canción", Channel: "Chan"})
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
	m.st = player.State{Position: 65, Duration: 200, Volume: 80}
	out := m.View()
	if !strings.Contains(out, "Mi Canción") {
		t.Fatalf("compact view should show title:\n%s", out)
	}
	if !strings.Contains(out, "1:05") || !strings.Contains(out, "3:20") {
		t.Fatalf("compact view should show times:\n%s", out)
	}
}

func TestExpandedViewShowsTabs(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "T"})
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
	m.mode = modeExpanded
	out := m.View()
	for _, label := range []string{"Cola", "Buscar", "Historial", "Favoritos"} {
		if !strings.Contains(out, label) {
			t.Fatalf("expanded view should show tab %q:\n%s", label, out)
		}
	}
}
