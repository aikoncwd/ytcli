package store

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/AikonCWD/ytcli/internal/track"
)

func tk(id string) track.Track {
	return track.Track{ID: id, Title: "T" + id, URL: "https://www.youtube.com/watch?v=" + id}
}

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
		s.AppendHistory(tk(fmt.Sprintf("id%03d", i)))
	}
	h, _ := s.LoadHistory()
	if len(h) != historyCap {
		t.Fatalf("history len = %d; want %d", len(h), historyCap)
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

func TestRoundTripKeepsAllFields(t *testing.T) {
	s := New(t.TempDir())
	want := track.Track{ID: "x1", URL: "https://youtu.be/x1", Title: "Canción | rara\tcon tab",
		Channel: "Canal", Duration: 3725}
	live := track.Track{ID: "x2", URL: "https://youtu.be/x2", Title: "Directo", Live: true}
	if err := s.SavePlaylist([]track.Track{want, live}); err != nil {
		t.Fatal(err)
	}
	got, err := s.LoadPlaylist()
	if err != nil || len(got) != 2 {
		t.Fatalf("playlist = %+v, %v", got, err)
	}
	if got[0].ID != "x1" || got[0].Duration != 3725 || got[0].Channel != "Canal" {
		t.Fatalf("fields lost: %+v", got[0])
	}
	if strings.Contains(got[0].Title, "\t") {
		t.Fatalf("tab should be sanitized: %q", got[0].Title)
	}
	if !got[1].Live || got[1].Duration != 0 {
		t.Fatalf("live flag lost: %+v", got[1])
	}
}

func TestHandEditedFile(t *testing.T) {
	dir := t.TempDir()
	content := "# comentario\r\n\r\n" +
		"https://www.youtube.com/watch?v=abc123\r\n" + // bare URL, CRLF
		"https://youtu.be/def456\tMi título\r\n"
	if err := os.WriteFile(filepath.Join(dir, "playlist.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := New(dir).LoadPlaylist()
	if err != nil || len(got) != 2 {
		t.Fatalf("playlist = %+v, %v", got, err)
	}
	if got[0].ID != "abc123" {
		t.Fatalf("ID should derive from watch URL: %+v", got[0])
	}
	if got[1].ID != "def456" || got[1].Title != "Mi título" {
		t.Fatalf("ID should derive from youtu.be URL: %+v", got[1])
	}
}

func TestMigrateJSON(t *testing.T) {
	oldDir, newDir := t.TempDir(), t.TempDir()
	oldJSON := `[{"ID":"a","URL":"https://youtu.be/a","Title":"Vieja","Channel":"C","Duration":10}]`
	if err := os.WriteFile(filepath.Join(oldDir, "history.json"), []byte(oldJSON), 0o644); err != nil {
		t.Fatal(err)
	}
	s := New(newDir)
	if err := s.MigrateJSON(oldDir); err != nil {
		t.Fatal(err)
	}
	h, _ := s.LoadHistory()
	if len(h) != 1 || h[0].ID != "a" || h[0].Title != "Vieja" {
		t.Fatalf("migrated history = %+v", h)
	}
	// A second migration must not clobber existing text files.
	s.AppendHistory(tk("b"))
	if err := s.MigrateJSON(oldDir); err != nil {
		t.Fatal(err)
	}
	h, _ = s.LoadHistory()
	if len(h) != 2 {
		t.Fatalf("migration overwrote existing file: %+v", h)
	}
}

func TestVideoIDFromURL(t *testing.T) {
	cases := map[string]string{
		"https://www.youtube.com/watch?v=abc":     "abc",
		"https://youtu.be/def":                    "def",
		"https://example.com/algo":                "https://example.com/algo",
		"https://www.youtube.com/watch?v=x&t=10s": "x",
		"https://www.youtube.com/shorts/s1":       "s1",
		"https://www.youtube.com/live/l1":         "l1",
		"https://www.youtube.com/embed/e1":        "e1",
	}
	for in, want := range cases {
		if got := videoIDFromURL(in); got != want {
			t.Errorf("videoIDFromURL(%q) = %q; want %q", in, got, want)
		}
	}
}

func TestJunkLinesAreIgnoredAndSchemelessNormalized(t *testing.T) {
	dir := t.TempDir()
	content := "apuntar más jazz aquí\n" + // stray note, not a URL
		"www.youtube.com/watch?v=abc\n" + // scheme-less URL
		"youtu.be/def\n"
	if err := os.WriteFile(filepath.Join(dir, "playlist.txt"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := New(dir).LoadPlaylist()
	if err != nil || len(got) != 2 {
		t.Fatalf("playlist = %+v, %v; want 2 tracks (junk ignored)", got, err)
	}
	if got[0].URL != "https://www.youtube.com/watch?v=abc" || got[0].ID != "abc" {
		t.Fatalf("scheme-less URL not normalized: %+v", got[0])
	}
	if got[1].URL != "https://youtu.be/def" || got[1].ID != "def" {
		t.Fatalf("scheme-less youtu.be not normalized: %+v", got[1])
	}
}
