package tui

import (
	"errors"
	"testing"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AikonCWD/ytcli/internal/player"
	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/track"
)

type fakePlayer struct {
	loaded   string
	toggled  int
	seeked   int
	volume   int
	endCh    chan struct{}
	curState player.State
	loadErr  error
}

func newFakePlayer() *fakePlayer { return &fakePlayer{endCh: make(chan struct{}, 1)} }

func (f *fakePlayer) Load(u string) error    { f.loaded = u; return f.loadErr }
func (f *fakePlayer) TogglePause() error     { f.toggled++; return nil }
func (f *fakePlayer) Seek(d int) error       { f.seeked += d; return nil }
func (f *fakePlayer) SetVolume(v int) error  { f.volume = v; return nil }
func (f *fakePlayer) State() player.State    { return f.curState }
func (f *fakePlayer) EndCh() <-chan struct{} { return f.endCh }

type fakeSearch struct{ result []track.Track }

func (f *fakeSearch) Search(string, int) ([]track.Track, error) { return f.result, nil }
func (f *fakeSearch) Resolve(string) ([]track.Track, error)     { return f.result, nil }

type fakeStore struct{ appended, favToggled int }

func (f *fakeStore) AppendHistory(track.Track) error          { f.appended++; return nil }
func (f *fakeStore) ToggleFavorite(track.Track) (bool, error) { f.favToggled++; return true, nil }
func (f *fakeStore) LoadHistory() ([]track.Track, error)      { return nil, nil }
func (f *fakeStore) LoadFavorites() ([]track.Track, error)    { return nil, nil }

func newTestModel() (Model, *fakePlayer, *fakeStore) {
	q := queue.New()
	q.Add(track.Track{ID: "a", URL: "ua"}, track.Track{ID: "b", URL: "ub"})
	fp := newFakePlayer()
	fs := &fakeStore{}
	m := New(q, fp, &fakeSearch{}, fs, 80)
	return m, fp, fs
}

func key(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

func TestSpaceTogglesPause(t *testing.T) {
	m, fp, _ := newTestModel()
	m.Update(key(' '))
	// space is a special key, not a rune; send it as such:
	m.Update(tea.KeyMsg{Type: tea.KeySpace})
	if fp.toggled == 0 {
		t.Fatal("space should toggle pause")
	}
}

func TestSeekKeys(t *testing.T) {
	m, fp, _ := newTestModel()
	m.Update(tea.KeyMsg{Type: tea.KeyRight})
	m.Update(tea.KeyMsg{Type: tea.KeyLeft})
	if fp.seeked != 0 { // +10 then -10
		t.Fatalf("seeked = %d; want 0", fp.seeked)
	}
}

func TestNextLoadsAndRecordsHistory(t *testing.T) {
	m, fp, fs := newTestModel()
	m.Update(key('n'))
	if fp.loaded != "ub" {
		t.Fatalf("loaded = %q; want ub", fp.loaded)
	}
	if fs.appended == 0 {
		t.Fatal("next should append to history")
	}
}

func TestVolumeKeys(t *testing.T) {
	m, fp, _ := newTestModel()
	m.Update(tea.KeyMsg{Type: tea.KeyUp})
	if fp.volume != 85 {
		t.Fatalf("volume = %d; want 85", fp.volume)
	}
}

func TestRepeatCycles(t *testing.T) {
	m, _, _ := newTestModel()
	m2, _ := m.Update(key('r'))
	if m2.(Model).q.Repeat() != queue.RepeatAll {
		t.Fatal("r should cycle to RepeatAll")
	}
}

func TestShuffleToggles(t *testing.T) {
	m, _, _ := newTestModel()
	m2, _ := m.Update(key('s'))
	if !m2.(Model).q.Shuffle() {
		t.Fatal("s should enable shuffle")
	}
}

func TestTabTogglesMode(t *testing.T) {
	m, _, _ := newTestModel()
	m2, _ := m.Update(tea.KeyMsg{Type: tea.KeyTab})
	if m2.(Model).mode != modeExpanded {
		t.Fatal("tab should expand")
	}
}

func TestSlashEntersSearch(t *testing.T) {
	m, _, _ := newTestModel()
	m2, _ := m.Update(key('/'))
	mm := m2.(Model)
	if !mm.searching || mm.tab != tabSearch {
		t.Fatal("/ should enter search mode on the Search tab")
	}
}

func TestSpaceRuneTogglesPause(t *testing.T) {
	m, fp, _ := newTestModel()
	m.Update(key(' '))
	if fp.toggled == 0 {
		t.Fatal("space rune should toggle pause")
	}
}

func TestPlayLoadErrorSetsStatusAndSkipsHistory(t *testing.T) {
	m, fp, fs := newTestModel()
	fp.loadErr = errors.New("boom")
	m2, _ := m.Update(key('n'))
	mm := m2.(Model)
	if mm.status == "" {
		t.Fatal("load error should set status")
	}
	if fs.appended != 0 {
		t.Fatal("history should not be appended when load fails")
	}
}
