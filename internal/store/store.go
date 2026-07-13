// Package store persists history and favorites as JSON under %APPDATA%\ytcli.
package store

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"

	"github.com/AikonCWD/ytcli/internal/track"
)

const historyCap = 200

type Store struct{ dir string }

func New(dir string) *Store { return &Store{dir: dir} }

// DefaultDir returns %APPDATA%\ytcli.
func DefaultDir() (string, error) {
	base := os.Getenv("APPDATA")
	if base == "" {
		return "", errors.New("APPDATA no está definido")
	}
	return filepath.Join(base, "ytcli"), nil
}

func (s *Store) path(name string) string { return filepath.Join(s.dir, name) }

func (s *Store) load(name string) ([]track.Track, error) {
	b, err := os.ReadFile(s.path(name))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var ts []track.Track
	if err := json.Unmarshal(b, &ts); err != nil {
		return nil, err
	}
	return ts, nil
}

func (s *Store) save(name string, ts []track.Track) error {
	if err := os.MkdirAll(s.dir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(ts, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.path(name), b, 0o644)
}

func (s *Store) LoadHistory() ([]track.Track, error) { return s.load("history.json") }

func (s *Store) AppendHistory(t track.Track) error {
	ts, err := s.load("history.json")
	if err != nil {
		return err
	}
	out := make([]track.Track, 0, len(ts)+1)
	out = append(out, t)
	for _, x := range ts {
		if x.ID != t.ID {
			out = append(out, x)
		}
	}
	if len(out) > historyCap {
		out = out[:historyCap]
	}
	return s.save("history.json", out)
}

func (s *Store) LoadFavorites() ([]track.Track, error) { return s.load("favorites.json") }

func (s *Store) IsFavorite(id string) (bool, error) {
	ts, err := s.load("favorites.json")
	if err != nil {
		return false, err
	}
	for _, x := range ts {
		if x.ID == id {
			return true, nil
		}
	}
	return false, nil
}

// ToggleFavorite adds or removes t and returns the new favorite state.
func (s *Store) ToggleFavorite(t track.Track) (bool, error) {
	ts, err := s.load("favorites.json")
	if err != nil {
		return false, err
	}
	for i, x := range ts {
		if x.ID == t.ID {
			ts = append(ts[:i], ts[i+1:]...)
			return false, s.save("favorites.json", ts)
		}
	}
	ts = append([]track.Track{t}, ts...)
	return true, s.save("favorites.json", ts)
}
