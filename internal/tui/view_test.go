package tui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"

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
	m.width = 80
	m.mode = modeExpanded
	out := m.View()
	for _, label := range []string{"Cola", "Buscar", "Historial", "Favoritos"} {
		if !strings.Contains(out, label) {
			t.Fatalf("expanded view should show tab %q:\n%s", label, out)
		}
	}
}

func TestLiveTrackShowsEnDirectoInsteadOfTimes(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "Radio lofi", Live: true})
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
	m.st = player.State{Position: 73696, Duration: 73702} // mpv DVR-edge values
	out := m.View()
	if !strings.Contains(out, "EN DIRECTO") {
		t.Fatalf("live track should show EN DIRECTO:\n%s", out)
	}
	if strings.Contains(out, "20:28:16") {
		t.Fatalf("live track should not show DVR position/duration:\n%s", out)
	}
}

// Every rendered line must fit the terminal or it wraps in the real terminal,
// destabilizing the box height and breaking bubbletea's inline renderer.
func TestNoLineOverflowsTerminalWidth(t *testing.T) {
	long := strings.Repeat("Títulø 日本語 muy largo ", 8)
	for _, w := range []int{46, 60, 80, 120} {
		q := queue.New()
		q.Add(track.Track{ID: "a", URL: "ua", Title: long, Channel: "Canal 日本語", Duration: 3725})
		q.Add(track.Track{ID: "b", URL: "ub", Title: "corta", Live: true})
		m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
		m.width, m.height = w, 30
		m.st = player.State{Position: 5000, Duration: 7000}
		m.status = long // a long status must not wrap either
		for _, mode := range []mode{modeCompact, modeExpanded} {
			m.mode = mode
			for i, line := range strings.Split(m.View(), "\n") {
				if lw := lipgloss.Width(line); lw >= w {
					t.Fatalf("width=%d mode=%v line %d overflows (%d cells): %q", w, mode, i, lw, line)
				}
			}
		}
		m.showHelp = true
		for i, line := range strings.Split(m.View(), "\n") {
			if lw := lipgloss.Width(line); lw >= w {
				t.Fatalf("width=%d help line %d overflows (%d cells): %q", w, i, lw, line)
			}
		}
		m.showHelp = false
	}
}

// The tab header is 45 cells un-styled; at narrow widths it must be truncated
// or it wraps inside the box and the frame height destabilizes.
func TestExpandedViewStableHeightAtNarrowWidth(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "T"})
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
	m.width, m.height = 46, 30
	m.mode = modeExpanded
	m.st = player.State{Position: 73696, Duration: 73702} // long HH:MM:SS times
	lines := strings.Split(m.View(), "\n")
	// compact (6) + expanded: border + header + 1 row + footer + border (5)
	if len(lines) != 11 {
		t.Fatalf("expanded view at width 46 should be 11 lines, got %d:\n%s", len(lines), m.View())
	}
}

func TestCompactViewAlwaysFourContentLines(t *testing.T) {
	long := strings.Repeat("larguísimo ", 20)
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: long})
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
	m.width = 46
	lines := strings.Split(m.View(), "\n")
	if len(lines) != 6 { // border + 4 content + border
		t.Fatalf("compact box must always be 6 lines, got %d:\n%s", len(lines), m.View())
	}
}

// Muted volume and repeat-one get their own icons (🔇, 🔂).
func TestCompactViewStateIcons(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "T"})
	q.SetRepeat(queue.RepeatOne)
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 0) // vol 0 = mute
	out := m.View()
	if !strings.Contains(out, "🔇") {
		t.Fatalf("muted view should show 🔇:\n%s", out)
	}
	if !strings.Contains(out, "🔂") {
		t.Fatalf("repeat-one view should show 🔂:\n%s", out)
	}
	m.vol = 80
	q.SetRepeat(queue.RepeatAll)
	out = m.View()
	if !strings.Contains(out, "🔊") || !strings.Contains(out, "🔁") {
		t.Fatalf("normal view should show 🔊 and 🔁:\n%s", out)
	}
}

// Favorited tracks show ⭐ in every list except the Favoritos tab itself.
func TestFavoriteStarShownInListsButNotFavoritesTab(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "Uno"})
	fav := track.Track{ID: "a", URL: "ua", Title: "Uno"}
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{favs: []track.Track{fav}}, 80)
	m.width, m.height = 80, 30
	m.mode = modeExpanded
	if out := m.View(); !strings.Contains(out, "⭐") {
		t.Fatalf("queue tab should mark favorites with ⭐:\n%s", out)
	}
	m.tab = tabFavorites
	m.favorites = []track.Track{fav}
	if out := m.View(); strings.Contains(out, "⭐") {
		t.Fatalf("favorites tab should not repeat the ⭐ marker:\n%s", out)
	}
}

func TestHelpViewListsKeys(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "T"})
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
	m.width, m.height = 80, 30
	m.mode = modeExpanded
	m.showHelp = true
	out := m.View()
	for _, want := range []string{"Ayuda", "shuffle", "repeat", "pestañas", "salir"} {
		if !strings.Contains(out, want) {
			t.Fatalf("help view should mention %q:\n%s", want, out)
		}
	}
}

func TestExpandedListShowsMetaAndCurrentMarker(t *testing.T) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua", Title: "Uno", Channel: "Canal", Duration: 225})
	q.Add(track.Track{ID: "b", URL: "ub", Title: "Dos", Live: true})
	m := New(q, newFakePlayer(), &fakeSearch{}, &fakeStore{}, 80)
	m.width, m.height = 80, 30
	m.mode = modeExpanded
	m.cursor = 1
	out := m.View()
	if !strings.Contains(out, "Canal") || !strings.Contains(out, "3:45") {
		t.Fatalf("queue rows should show channel and duration:\n%s", out)
	}
	if !strings.Contains(out, "♪ Uno") {
		t.Fatalf("current track should be marked with ♪:\n%s", out)
	}
	if !strings.Contains(out, "directo") {
		t.Fatalf("live rows should be marked:\n%s", out)
	}
}
