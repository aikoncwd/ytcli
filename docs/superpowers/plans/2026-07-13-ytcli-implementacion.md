# ytcli — Plan de Implementación

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Construir una TUI en Go que reproduce audio de YouTube (vídeos y playlists) con una interfaz compacta/expandible, búsqueda por texto y persistencia de historial y favoritos.

**Architecture:** La app orquesta dos binarios externos: **mpv** (motor de audio, controlado por su IPC JSON sobre un named pipe de Windows) y **yt-dlp** (búsqueda, expansión de playlists y metadatos). La lógica se reparte en paquetes `internal/` con fronteras claras (`track`, `queue`, `store`, `youtube`, `deps`, `player`, `tui`) y un `main` que cablea todo. La UI es Bubbletea en modo inline (sin alt-screen) para ocupar pocas líneas.

**Tech Stack:** Go 1.23+, Bubbletea + Lipgloss (Charm), `github.com/Microsoft/go-winio` (named pipe), `github.com/bodgit/sevenzip` (extraer mpv .7z), mpv y yt-dlp como binarios externos.

## Global Constraints

- Plataforma objetivo: **solo Windows** (rutas, named pipe `\\.\pipe\ytcli-mpv`, extensiones `.exe`).
- Go **1.23+**.
- Módulo Go: `github.com/AikonCWD/ytcli`.
- Reproducción: **streaming directo** (nunca descargar audio a disco).
- El tipo `track.Track` es la moneda común entre paquetes: `{ID, URL, Title, Channel string; Duration int}` (Duration en segundos).
- Directorios: binarios en `%LOCALAPPDATA%\ytcli\bin`; datos (historial/favoritos) en `%APPDATA%\ytcli`.
- Salto de seek: **±10 segundos**. Paso de volumen: **5**. Volumen inicial: **80**.
- Búsqueda por defecto: **15** resultados. Historial: tope **200** entradas.
- Dependencias externas se **auto-descargan** en el primer arranque si no están en PATH ni en el binDir.
- TDD: cada paso de código va precedido de su test. Commits frecuentes.
- Prerrequisito del desarrollador: toolchain de Go instalado (`go version` ≥ 1.23).

---

### Task 1: Scaffolding del módulo y tipo `Track`

**Files:**
- Create: `go.mod`
- Create: `internal/track/track.go`
- Create: `.gitignore`

**Interfaces:**
- Consumes: nada.
- Produces: `track.Track{ID, URL, Title, Channel string; Duration int}` — tipo compartido por todos los paquetes.

- [ ] **Step 1: Inicializar el módulo**

Run:
```bash
go mod init github.com/AikonCWD/ytcli
```
Expected: crea `go.mod` con `module github.com/AikonCWD/ytcli` y `go 1.23` (o superior).

- [ ] **Step 2: Crear el tipo `Track`**

`internal/track/track.go`:
```go
// Package track defines the shared media item used across ytcli.
package track

// Track is a single playable YouTube item.
type Track struct {
	ID       string // YouTube video id
	URL      string // watch URL handed to mpv
	Title    string
	Channel  string
	Duration int // seconds; 0 if unknown
}
```

- [ ] **Step 3: Crear `.gitignore`**

`.gitignore`:
```
/ytcli.exe
/ytcli
/dist/
```

- [ ] **Step 4: Verificar que compila**

Run:
```bash
go build ./...
```
Expected: sin salida (éxito), sin errores de compilación.

- [ ] **Step 5: Commit**

```bash
git add go.mod internal/track/track.go .gitignore
git commit -m "feat: scaffolding del módulo y tipo Track"
```

---

### Task 2: Paquete `queue` (cola de reproducción)

**Files:**
- Create: `internal/queue/queue.go`
- Test: `internal/queue/queue_test.go`

**Interfaces:**
- Consumes: `track.Track`.
- Produces:
  - `queue.RepeatMode` con `RepeatOff`, `RepeatAll`, `RepeatOne`.
  - `func New() *Queue`
  - `(*Queue) Add(tracks ...track.Track)`
  - `(*Queue) Len() int`
  - `(*Queue) Current() (track.Track, bool)`
  - `(*Queue) CurrentIndex() int` (índice dentro del orden de reproducción; -1 si vacía)
  - `(*Queue) Tracks() []track.Track` (en orden de reproducción)
  - `(*Queue) Next() (track.Track, bool)`
  - `(*Queue) Prev() (track.Track, bool)`
  - `(*Queue) JumpTo(index int) (track.Track, bool)`
  - `(*Queue) SetShuffle(on bool)` / `(*Queue) Shuffle() bool`
  - `(*Queue) SetRepeat(m RepeatMode)` / `(*Queue) Repeat() RepeatMode`

- [ ] **Step 1: Escribir los tests que fallan**

`internal/queue/queue_test.go`:
```go
package queue

import (
	"sort"
	"testing"

	"github.com/AikonCWD/ytcli/internal/track"
)

func tracks(ids ...string) []track.Track {
	out := make([]track.Track, len(ids))
	for i, id := range ids {
		out[i] = track.Track{ID: id, URL: "u" + id, Title: id}
	}
	return out
}

func TestAddAndCurrent(t *testing.T) {
	q := New()
	if _, ok := q.Current(); ok {
		t.Fatal("empty queue should have no current")
	}
	q.Add(tracks("a", "b")...)
	cur, ok := q.Current()
	if !ok || cur.ID != "a" {
		t.Fatalf("current = %v, %v; want a", cur.ID, ok)
	}
	if q.Len() != 2 {
		t.Fatalf("len = %d; want 2", q.Len())
	}
}

func TestNextRepeatOff(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	if n, ok := q.Next(); !ok || n.ID != "b" {
		t.Fatalf("next = %v,%v; want b,true", n.ID, ok)
	}
	if _, ok := q.Next(); ok {
		t.Fatal("next past end should be false with RepeatOff")
	}
}

func TestNextRepeatAllWraps(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	q.SetRepeat(RepeatAll)
	q.Next()                    // b
	n, ok := q.Next()           // wrap to a
	if !ok || n.ID != "a" {
		t.Fatalf("wrap = %v,%v; want a,true", n.ID, ok)
	}
}

func TestNextRepeatOneStays(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	q.SetRepeat(RepeatOne)
	n, ok := q.Next()
	if !ok || n.ID != "a" {
		t.Fatalf("repeat-one next = %v,%v; want a,true", n.ID, ok)
	}
}

func TestPrevRepeatOff(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b")...)
	if _, ok := q.Prev(); ok {
		t.Fatal("prev before start should be false")
	}
	q.Next() // b
	p, ok := q.Prev()
	if !ok || p.ID != "a" {
		t.Fatalf("prev = %v,%v; want a,true", p.ID, ok)
	}
}

func TestJumpTo(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b", "c")...)
	j, ok := q.JumpTo(2)
	if !ok || j.ID != "c" {
		t.Fatalf("jump = %v,%v; want c,true", j.ID, ok)
	}
	if _, ok := q.JumpTo(9); ok {
		t.Fatal("jump out of range should be false")
	}
}

func TestShufflePreservesSetAndCurrent(t *testing.T) {
	q := New()
	q.Add(tracks("a", "b", "c", "d")...)
	q.Next() // current = b
	q.SetShuffle(true)
	cur, _ := q.Current()
	if cur.ID != "b" {
		t.Fatalf("shuffle changed current to %v; want b", cur.ID)
	}
	var ids []string
	for _, tr := range q.Tracks() {
		ids = append(ids, tr.ID)
	}
	sort.Strings(ids)
	want := []string{"a", "b", "c", "d"}
	for i := range want {
		if ids[i] != want[i] {
			t.Fatalf("shuffle lost tracks: got %v", ids)
		}
	}
}
```

- [ ] **Step 2: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/queue/
```
Expected: FALLA (no compila: `New`, `RepeatOff`, etc. no existen).

- [ ] **Step 3: Implementar `queue`**

`internal/queue/queue.go`:
```go
// Package queue holds the in-memory playback queue: order, current position,
// shuffle and repeat. Pure logic with no external dependencies.
package queue

import (
	"math/rand"

	"github.com/AikonCWD/ytcli/internal/track"
)

// RepeatMode controls what Next/Prev do at the ends of the queue.
type RepeatMode int

const (
	RepeatOff RepeatMode = iota
	RepeatAll
	RepeatOne
)

// Queue is the ordered list of tracks plus playback state.
type Queue struct {
	tracks  []track.Track
	order   []int // indices into tracks, in playback order
	pos     int   // index into order; -1 when empty
	shuffle bool
	repeat  RepeatMode
}

func New() *Queue { return &Queue{pos: -1, repeat: RepeatOff} }

func (q *Queue) Add(tracks ...track.Track) {
	start := len(q.tracks)
	q.tracks = append(q.tracks, tracks...)
	for i := range tracks {
		q.order = append(q.order, start+i)
	}
	if q.pos == -1 && len(q.order) > 0 {
		q.pos = 0
	}
}

func (q *Queue) Len() int          { return len(q.tracks) }
func (q *Queue) CurrentIndex() int { return q.pos }

func (q *Queue) Current() (track.Track, bool) {
	if q.pos < 0 || q.pos >= len(q.order) {
		return track.Track{}, false
	}
	return q.tracks[q.order[q.pos]], true
}

func (q *Queue) Tracks() []track.Track {
	out := make([]track.Track, 0, len(q.order))
	for _, i := range q.order {
		out = append(out, q.tracks[i])
	}
	return out
}

func (q *Queue) Next() (track.Track, bool) {
	if len(q.order) == 0 {
		return track.Track{}, false
	}
	switch q.repeat {
	case RepeatOne:
		// stay on current
	case RepeatAll:
		q.pos = (q.pos + 1) % len(q.order)
	default: // RepeatOff
		if q.pos+1 >= len(q.order) {
			return track.Track{}, false
		}
		q.pos++
	}
	return q.Current()
}

func (q *Queue) Prev() (track.Track, bool) {
	if len(q.order) == 0 {
		return track.Track{}, false
	}
	switch q.repeat {
	case RepeatOne:
		// stay
	case RepeatAll:
		q.pos = (q.pos - 1 + len(q.order)) % len(q.order)
	default:
		if q.pos-1 < 0 {
			return track.Track{}, false
		}
		q.pos--
	}
	return q.Current()
}

func (q *Queue) JumpTo(index int) (track.Track, bool) {
	if index < 0 || index >= len(q.order) {
		return track.Track{}, false
	}
	q.pos = index
	return q.Current()
}

func (q *Queue) SetShuffle(on bool) {
	if on == q.shuffle {
		return
	}
	q.shuffle = on
	cur := -1
	if q.pos >= 0 && q.pos < len(q.order) {
		cur = q.order[q.pos]
	}
	if on {
		rand.Shuffle(len(q.order), func(i, j int) {
			q.order[i], q.order[j] = q.order[j], q.order[i]
		})
	} else {
		for i := range q.order {
			q.order[i] = i
		}
	}
	if cur >= 0 {
		for i, idx := range q.order {
			if idx == cur {
				q.pos = i
				break
			}
		}
	}
}

func (q *Queue) Shuffle() bool             { return q.shuffle }
func (q *Queue) SetRepeat(m RepeatMode)    { q.repeat = m }
func (q *Queue) Repeat() RepeatMode        { return q.repeat }
```

- [ ] **Step 4: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/queue/
```
Expected: PASA (`ok  github.com/AikonCWD/ytcli/internal/queue`).

- [ ] **Step 5: Commit**

```bash
git add internal/queue/
git commit -m "feat: paquete queue con shuffle y modos de repeat"
```

---

### Task 3: Paquete `store` (persistencia de historial y favoritos)

**Files:**
- Create: `internal/store/store.go`
- Test: `internal/store/store_test.go`

**Interfaces:**
- Consumes: `track.Track`.
- Produces:
  - `func New(dir string) *Store`
  - `func DefaultDir() (string, error)` (→ `%APPDATA%\ytcli`)
  - `(*Store) LoadHistory() ([]track.Track, error)`
  - `(*Store) AppendHistory(t track.Track) error` (dedup move-to-front, tope 200)
  - `(*Store) LoadFavorites() ([]track.Track, error)`
  - `(*Store) IsFavorite(id string) (bool, error)`
  - `(*Store) ToggleFavorite(t track.Track) (bool, error)` (devuelve el nuevo estado)

- [ ] **Step 1: Escribir los tests que fallan**

`internal/store/store_test.go`:
```go
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
		s.AppendHistory(track.Track{ID: string(rune('A' + i%26)) + string(rune(i))})
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
```

- [ ] **Step 2: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/store/
```
Expected: FALLA (símbolos indefinidos).

- [ ] **Step 3: Implementar `store`**

`internal/store/store.go`:
```go
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
```

- [ ] **Step 4: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/store/
```
Expected: PASA.

- [ ] **Step 5: Commit**

```bash
git add internal/store/
git commit -m "feat: paquete store para historial y favoritos"
```

---

### Task 4: Paquete `youtube` (envoltorio de yt-dlp)

**Files:**
- Create: `internal/youtube/youtube.go`
- Test: `internal/youtube/youtube_test.go`

**Interfaces:**
- Consumes: `track.Track`; ruta al binario yt-dlp.
- Produces:
  - `func New(ytDlpPath string) *Client`
  - `(*Client) Search(query string, n int) ([]track.Track, error)`
  - `(*Client) Resolve(url string) ([]track.Track, error)`
  - Internos testeables: `buildSearchArgs(query string, n int) []string`, `buildResolveArgs(url string) []string`, `parseDump(b []byte) ([]track.Track, error)`.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/youtube/youtube_test.go`:
```go
package youtube

import "testing"

func TestBuildSearchArgs(t *testing.T) {
	got := buildSearchArgs("daft punk", 15)
	want := []string{"ytsearch15:daft punk", "-J", "--flat-playlist", "--no-warnings"}
	if len(got) != len(want) {
		t.Fatalf("args = %v; want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("args[%d] = %q; want %q", i, got[i], want[i])
		}
	}
}

func TestBuildResolveArgs(t *testing.T) {
	got := buildResolveArgs("https://youtu.be/x")
	if got[0] != "https://youtu.be/x" || got[1] != "-J" {
		t.Fatalf("resolve args = %v", got)
	}
}

func TestParseDumpSingleVideo(t *testing.T) {
	js := []byte(`{"id":"abc","title":"Song","channel":"Chan","duration":211.0,"webpage_url":"https://youtu.be/abc"}`)
	got, err := parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("len = %d; want 1", len(got))
	}
	tr := got[0]
	if tr.ID != "abc" || tr.Title != "Song" || tr.Channel != "Chan" || tr.Duration != 211 || tr.URL != "https://youtu.be/abc" {
		t.Fatalf("track = %+v", tr)
	}
}

func TestParseDumpPlaylistUsesUploaderAndBuildsURL(t *testing.T) {
	js := []byte(`{"_type":"playlist","entries":[
		{"id":"1","title":"A","uploader":"U1"},
		{"id":"2","title":"B","channel":"C2","duration":10.0}
	]}`)
	got, err := parseDump(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("len = %d; want 2", len(got))
	}
	if got[0].Channel != "U1" {
		t.Fatalf("entry0 channel = %q; want U1 (uploader fallback)", got[0].Channel)
	}
	if got[0].URL != "https://www.youtube.com/watch?v=1" {
		t.Fatalf("entry0 url = %q; want built URL", got[0].URL)
	}
	if got[1].Duration != 10 {
		t.Fatalf("entry1 duration = %d; want 10", got[1].Duration)
	}
}
```

- [ ] **Step 2: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/youtube/
```
Expected: FALLA (símbolos indefinidos).

- [ ] **Step 3: Implementar `youtube`**

`internal/youtube/youtube.go`:
```go
// Package youtube wraps the yt-dlp binary for search, playlist expansion and
// metadata extraction. It never downloads media; mpv streams that directly.
package youtube

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"github.com/AikonCWD/ytcli/internal/track"
)

type Client struct{ bin string }

func New(ytDlpPath string) *Client { return &Client{bin: ytDlpPath} }

func buildSearchArgs(query string, n int) []string {
	return []string{fmt.Sprintf("ytsearch%d:%s", n, query), "-J", "--flat-playlist", "--no-warnings"}
}

func buildResolveArgs(url string) []string {
	return []string{url, "-J", "--flat-playlist", "--no-warnings"}
}

type rawEntry struct {
	ID         string   `json:"id"`
	Title      string   `json:"title"`
	Channel    string   `json:"channel"`
	Uploader   string   `json:"uploader"`
	Duration   *float64 `json:"duration"`
	WebpageURL string   `json:"webpage_url"`
	URL        string   `json:"url"`
}

type rawDump struct {
	rawEntry
	Entries []rawEntry `json:"entries"`
}

func (e rawEntry) toTrack() track.Track {
	ch := e.Channel
	if ch == "" {
		ch = e.Uploader
	}
	u := e.WebpageURL
	if u == "" {
		u = e.URL
	}
	if u == "" && e.ID != "" {
		u = "https://www.youtube.com/watch?v=" + e.ID
	}
	d := 0
	if e.Duration != nil {
		d = int(*e.Duration)
	}
	return track.Track{ID: e.ID, URL: u, Title: e.Title, Channel: ch, Duration: d}
}

// parseDump reads yt-dlp -J output: a single video object, or a playlist
// object with an "entries" array (also used by ytsearch results).
func parseDump(b []byte) ([]track.Track, error) {
	var d rawDump
	if err := json.Unmarshal(b, &d); err != nil {
		return nil, err
	}
	if len(d.Entries) > 0 {
		out := make([]track.Track, 0, len(d.Entries))
		for _, e := range d.Entries {
			out = append(out, e.toTrack())
		}
		return out, nil
	}
	return []track.Track{d.rawEntry.toTrack()}, nil
}

func (c *Client) run(args []string) ([]track.Track, error) {
	var out, stderr bytes.Buffer
	cmd := exec.Command(c.bin, args...)
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("yt-dlp falló: %w: %s", err, stderr.String())
	}
	return parseDump(out.Bytes())
}

func (c *Client) Search(query string, n int) ([]track.Track, error) {
	return c.run(buildSearchArgs(query, n))
}

func (c *Client) Resolve(url string) ([]track.Track, error) {
	return c.run(buildResolveArgs(url))
}
```

- [ ] **Step 4: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/youtube/
```
Expected: PASA.

- [ ] **Step 5: Commit**

```bash
git add internal/youtube/
git commit -m "feat: paquete youtube (envoltorio de yt-dlp con parseo -J)"
```

---

### Task 5: Paquete `deps` (bootstrap y auto-descarga de mpv/yt-dlp)

**Files:**
- Create: `internal/deps/deps.go`
- Test: `internal/deps/deps_test.go`

**Interfaces:**
- Consumes: variables de entorno de Windows (`LOCALAPPDATA`), red (GitHub).
- Produces:
  - `type Paths struct{ Mpv, YtDlp string }`
  - `func BinDir() (string, error)` (→ `%LOCALAPPDATA%\ytcli\bin`)
  - `func Ensure(binDir string, progress func(string)) (Paths, error)`
  - Internos testeables: `resolveBinary(name, binDir string, lookPath func(string)(string,error), exists func(string)bool) (string, bool)`, `parseReleaseAssets(r io.Reader) ([]asset, error)`, `pickMpvAsset(assets []asset) (asset, error)`.

- [ ] **Step 1: Añadir la dependencia sevenzip**

Run:
```bash
go get github.com/bodgit/sevenzip@latest
```
Expected: actualiza `go.mod`/`go.sum` con `github.com/bodgit/sevenzip`.

- [ ] **Step 2: Escribir los tests que fallan**

`internal/deps/deps_test.go`:
```go
package deps

import (
	"errors"
	"strings"
	"testing"
)

func TestResolveBinaryPreferPath(t *testing.T) {
	look := func(n string) (string, error) { return `C:\tools\` + n + ".exe", nil }
	exists := func(string) bool { return false }
	got, ok := resolveBinary("mpv", `C:\bin`, look, exists)
	if !ok || got != `C:\tools\mpv.exe` {
		t.Fatalf("resolve = %q,%v; want C:\\tools\\mpv.exe,true", got, ok)
	}
}

func TestResolveBinaryFallbackBinDir(t *testing.T) {
	look := func(string) (string, error) { return "", errors.New("not found") }
	exists := func(p string) bool { return strings.HasSuffix(p, `\bin\yt-dlp.exe`) }
	got, ok := resolveBinary("yt-dlp", `C:\bin`, look, exists)
	if !ok || !strings.HasSuffix(got, `\bin\yt-dlp.exe`) {
		t.Fatalf("resolve = %q,%v; want bin dir path,true", got, ok)
	}
}

func TestResolveBinaryMissing(t *testing.T) {
	look := func(string) (string, error) { return "", errors.New("no") }
	exists := func(string) bool { return false }
	if _, ok := resolveBinary("mpv", `C:\bin`, look, exists); ok {
		t.Fatal("missing binary should return false")
	}
}

func TestParseReleaseAssets(t *testing.T) {
	js := strings.NewReader(`{"assets":[
		{"name":"mpv-x86_64-20250101-git-abc.7z","browser_download_url":"https://x/a.7z"},
		{"name":"mpv-dev-x86_64-20250101.7z","browser_download_url":"https://x/dev.7z"}
	]}`)
	assets, err := parseReleaseAssets(js)
	if err != nil {
		t.Fatal(err)
	}
	if len(assets) != 2 || assets[0].Name != "mpv-x86_64-20250101-git-abc.7z" {
		t.Fatalf("assets = %+v", assets)
	}
}

func TestPickMpvAsset(t *testing.T) {
	assets := []asset{
		{Name: "mpv-dev-x86_64-20250101.7z", URL: "dev"},
		{Name: "mpv-x86_64-v3-20250101-git-abc.7z", URL: "v3"},
		{Name: "mpv-x86_64-20250101-git-abc.7z", URL: "good"},
	}
	got, err := pickMpvAsset(assets)
	if err != nil {
		t.Fatal(err)
	}
	if got.URL != "good" {
		t.Fatalf("picked %q; want the plain x86_64 build", got.URL)
	}
}
```

- [ ] **Step 3: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/deps/
```
Expected: FALLA (símbolos indefinidos).

- [ ] **Step 4: Implementar `deps`**

`internal/deps/deps.go`:
```go
// Package deps locates mpv and yt-dlp, downloading them into %LOCALAPPDATA%\ytcli\bin
// on first run when they are not already available on PATH.
package deps

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

type Paths struct {
	Mpv   string
	YtDlp string
}

const (
	ytDlpURL   = "https://github.com/yt-dlp/yt-dlp/releases/latest/download/yt-dlp.exe"
	mpvRelease = "https://api.github.com/repos/shinchiro/mpv-winbuild-cmake/releases/latest"
)

// BinDir returns %LOCALAPPDATA%\ytcli\bin.
func BinDir() (string, error) {
	base := os.Getenv("LOCALAPPDATA")
	if base == "" {
		return "", errors.New("LOCALAPPDATA no está definido")
	}
	return filepath.Join(base, "ytcli", "bin"), nil
}

func fileExists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}

// resolveBinary returns an existing path for name, preferring PATH then binDir.
func resolveBinary(name, binDir string,
	lookPath func(string) (string, error),
	exists func(string) bool) (string, bool) {
	if p, err := lookPath(name); err == nil {
		return p, true
	}
	cand := filepath.Join(binDir, name+".exe")
	if exists(cand) {
		return cand, true
	}
	return "", false
}

type asset struct {
	Name string `json:"name"`
	URL  string `json:"browser_download_url"`
}

type release struct {
	Assets []asset `json:"assets"`
}

func parseReleaseAssets(r io.Reader) ([]asset, error) {
	var rel release
	if err := json.NewDecoder(r).Decode(&rel); err != nil {
		return nil, err
	}
	return rel.Assets, nil
}

// mpvAssetRE matches the plain 64-bit build, excluding "-dev-" and "-v3-" variants.
var mpvAssetRE = regexp.MustCompile(`^mpv-x86_64-\d[\w.-]*\.7z$`)

func pickMpvAsset(assets []asset) (asset, error) {
	for _, a := range assets {
		if mpvAssetRE.MatchString(a.Name) {
			return a, nil
		}
	}
	return asset{}, errors.New("no se encontró un asset mpv compatible")
}

func download(url, dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("descarga %s: HTTP %d", url, resp.StatusCode)
	}
	f, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}

// Ensure returns paths to mpv and yt-dlp, downloading any that are missing.
func Ensure(binDir string, progress func(string)) (Paths, error) {
	var p Paths

	if path, ok := resolveBinary("yt-dlp", binDir, exec.LookPath, fileExists); ok {
		p.YtDlp = path
	} else {
		progress("Descargando yt-dlp…")
		dst := filepath.Join(binDir, "yt-dlp.exe")
		if err := download(ytDlpURL, dst); err != nil {
			return p, fmt.Errorf("descargando yt-dlp: %w", err)
		}
		p.YtDlp = dst
	}

	if path, ok := resolveBinary("mpv", binDir, exec.LookPath, fileExists); ok {
		p.Mpv = path
	} else {
		progress("Buscando la última versión de mpv…")
		path, err := downloadMpv(binDir, progress)
		if err != nil {
			return p, fmt.Errorf("descargando mpv: %w", err)
		}
		p.Mpv = path
	}
	return p, nil
}

func downloadMpv(binDir string, progress func(string)) (string, error) {
	resp, err := http.Get(mpvRelease)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("consultando release de mpv: HTTP %d", resp.StatusCode)
	}
	assets, err := parseReleaseAssets(resp.Body)
	if err != nil {
		return "", err
	}
	a, err := pickMpvAsset(assets)
	if err != nil {
		return "", err
	}
	progress("Descargando mpv (" + a.Name + ")…")
	archive := filepath.Join(binDir, a.Name)
	if err := download(a.URL, archive); err != nil {
		return "", err
	}
	progress("Extrayendo mpv…")
	dst := filepath.Join(binDir, "mpv.exe")
	if err := extractMpvExe(archive, dst); err != nil {
		return "", err
	}
	_ = os.Remove(archive)
	return dst, nil
}
```

- [ ] **Step 5: Implementar la extracción del .7z**

`internal/deps/extract.go`:
```go
package deps

import (
	"errors"
	"io"
	"os"
	"path/filepath"

	"github.com/bodgit/sevenzip"
)

// extractMpvExe pulls mpv.exe out of a shinchiro mpv .7z archive.
func extractMpvExe(archivePath, destExe string) error {
	r, err := sevenzip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer r.Close()
	for _, f := range r.File {
		if filepath.Base(f.Name) == "mpv.exe" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			defer rc.Close()
			out, err := os.Create(destExe)
			if err != nil {
				return err
			}
			defer out.Close()
			_, err = io.Copy(out, rc)
			return err
		}
	}
	return errors.New("mpv.exe no está dentro del archivo .7z")
}
```

- [ ] **Step 6: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/deps/
```
Expected: PASA (tests de `resolveBinary`, `parseReleaseAssets`, `pickMpvAsset`).

- [ ] **Step 7: Prueba manual de descarga (integración)**

Borra `%LOCALAPPDATA%\ytcli\bin` si existe y ejecuta un pequeño programa temporal o, más adelante, el binario final en una máquina sin mpv/yt-dlp en PATH. Verifica que aparecen `mpv.exe` y `yt-dlp.exe` en el binDir. **Nota:** esta parte usa red y extracción real; no se cubre con tests automáticos.

- [ ] **Step 8: Commit**

```bash
git add internal/deps/ go.mod go.sum
git commit -m "feat: paquete deps con auto-descarga de mpv y yt-dlp"
```

---

### Task 6: Paquete `player` (proceso mpv + IPC JSON)

**Files:**
- Create: `internal/player/player.go`
- Create: `internal/player/ipc.go`
- Test: `internal/player/ipc_test.go`

**Interfaces:**
- Consumes: ruta al binario mpv (de `deps`).
- Produces:
  - `const PipeName = \\.\pipe\ytcli-mpv`
  - `type State struct{ Position, Duration, Volume int; Paused bool; Title string }`
  - `func New(mpvPath string) *Player`
  - `(*Player) Start() error` (lanza mpv, conecta el pipe, observa propiedades)
  - `(*Player) Close() error`
  - `(*Player) Load(url string) error`
  - `(*Player) TogglePause() error`
  - `(*Player) SetPaused(v bool) error`
  - `(*Player) Seek(delta int) error`
  - `(*Player) SetVolume(v int) error`
  - `(*Player) State() State` (lectura cacheada, sin round-trip)
  - `(*Player) EndCh() <-chan struct{}` (señal end-file para auto-avance)
  - Internos testeables (en `ipc.go`): `cmdLoad`, `cmdSetPause`, `cmdSeek`, `cmdSetVolume`, `cmdObserve`, `applyLine`.

- [ ] **Step 1: Añadir la dependencia go-winio**

Run:
```bash
go get github.com/Microsoft/go-winio@latest
```
Expected: actualiza `go.mod`/`go.sum`.

- [ ] **Step 2: Escribir los tests que fallan**

`internal/player/ipc_test.go`:
```go
package player

import (
	"strings"
	"testing"
)

func TestCmdBuilders(t *testing.T) {
	cases := []struct {
		got  []byte
		want string
	}{
		{cmdLoad("http://x"), `{"command":["loadfile","http://x","replace"]}`},
		{cmdSetPause(true), `{"command":["set_property","pause",true]}`},
		{cmdSetPause(false), `{"command":["set_property","pause",false]}`},
		{cmdSeek(-10), `{"command":["seek",-10,"relative"]}`},
		{cmdSetVolume(80), `{"command":["set_property","volume",80]}`},
		{cmdObserve(1, "time-pos"), `{"command":["observe_property",1,"time-pos"]}`},
	}
	for _, c := range cases {
		got := strings.TrimRight(string(c.got), "\n")
		if got != c.want {
			t.Fatalf("cmd = %s; want %s", got, c.want)
		}
		if !strings.HasSuffix(string(c.got), "\n") {
			t.Fatalf("command must end with newline: %q", string(c.got))
		}
	}
}

func TestApplyLineProperties(t *testing.T) {
	var st State
	applyLine([]byte(`{"event":"property-change","name":"time-pos","data":42.7}`), &st)
	if st.Position != 42 {
		t.Fatalf("position = %d; want 42", st.Position)
	}
	applyLine([]byte(`{"event":"property-change","name":"duration","data":211.0}`), &st)
	applyLine([]byte(`{"event":"property-change","name":"volume","data":75.0}`), &st)
	applyLine([]byte(`{"event":"property-change","name":"pause","data":true}`), &st)
	applyLine([]byte(`{"event":"property-change","name":"media-title","data":"Song"}`), &st)
	if st.Duration != 211 || st.Volume != 75 || !st.Paused || st.Title != "Song" {
		t.Fatalf("state = %+v", st)
	}
}

func TestApplyLineReturnsEvent(t *testing.T) {
	var st State
	ev, err := applyLine([]byte(`{"event":"end-file"}`), &st)
	if err != nil || ev != "end-file" {
		t.Fatalf("ev = %q, err = %v; want end-file,nil", ev, err)
	}
}
```

- [ ] **Step 3: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/player/
```
Expected: FALLA (símbolos indefinidos).

- [ ] **Step 4: Implementar los helpers IPC**

`internal/player/ipc.go`:
```go
package player

import "encoding/json"

func cmdJSON(parts ...interface{}) []byte {
	b, _ := json.Marshal(map[string]interface{}{"command": parts})
	return append(b, '\n')
}

func cmdLoad(url string) []byte  { return cmdJSON("loadfile", url, "replace") }
func cmdSetPause(p bool) []byte  { return cmdJSON("set_property", "pause", p) }
func cmdSeek(delta int) []byte   { return cmdJSON("seek", delta, "relative") }
func cmdSetVolume(v int) []byte  { return cmdJSON("set_property", "volume", v) }
func cmdObserve(id int, name string) []byte {
	return cmdJSON("observe_property", id, name)
}

type eventMsg struct {
	Event string          `json:"event"`
	Name  string          `json:"name"`
	Data  json.RawMessage `json:"data"`
}

// applyLine updates st from an mpv property-change line and returns the event name.
func applyLine(line []byte, st *State) (string, error) {
	var m eventMsg
	if err := json.Unmarshal(line, &m); err != nil {
		return "", err
	}
	if m.Event == "property-change" {
		switch m.Name {
		case "time-pos":
			var f float64
			if json.Unmarshal(m.Data, &f) == nil {
				st.Position = int(f)
			}
		case "duration":
			var f float64
			if json.Unmarshal(m.Data, &f) == nil {
				st.Duration = int(f)
			}
		case "volume":
			var f float64
			if json.Unmarshal(m.Data, &f) == nil {
				st.Volume = int(f)
			}
		case "pause":
			var b bool
			if json.Unmarshal(m.Data, &b) == nil {
				st.Paused = b
			}
		case "media-title":
			var s string
			if json.Unmarshal(m.Data, &s) == nil {
				st.Title = s
			}
		}
	}
	return m.Event, nil
}
```

- [ ] **Step 5: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/player/
```
Expected: PASA.

- [ ] **Step 6: Implementar el `Player` (proceso + pipe)**

`internal/player/player.go`:
```go
// Package player runs an mpv process and drives it over mpv's JSON IPC on a
// Windows named pipe. Audio only; mpv resolves YouTube via yt-dlp internally.
package player

import (
	"bufio"
	"errors"
	"fmt"
	"net"
	"os/exec"
	"sync"
	"time"

	winio "github.com/Microsoft/go-winio"
)

const PipeName = `\\.\pipe\ytcli-mpv`

type State struct {
	Position int
	Duration int
	Volume   int
	Paused   bool
	Title    string
}

type Player struct {
	mpvPath  string
	pipeName string
	cmd      *exec.Cmd
	conn     net.Conn
	mu       sync.Mutex
	state    State
	endCh    chan struct{}
}

func New(mpvPath string) *Player {
	return &Player{mpvPath: mpvPath, pipeName: PipeName, endCh: make(chan struct{}, 1)}
}

func (p *Player) Start() error {
	p.cmd = exec.Command(p.mpvPath,
		"--no-video", "--idle=yes", "--no-terminal",
		"--input-ipc-server="+p.pipeName,
		"--volume=80",
	)
	if err := p.cmd.Start(); err != nil {
		return fmt.Errorf("lanzando mpv: %w", err)
	}

	var conn net.Conn
	var err error
	for i := 0; i < 50; i++ { // mpv tarda un momento en crear el pipe
		timeout := 500 * time.Millisecond
		conn, err = winio.DialPipe(p.pipeName, &timeout)
		if err == nil {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}
	if err != nil {
		return fmt.Errorf("conectando al IPC de mpv: %w", err)
	}
	p.conn = conn

	for i, name := range []string{"time-pos", "duration", "volume", "pause", "media-title"} {
		if _, err := p.conn.Write(cmdObserve(i+1, name)); err != nil {
			return err
		}
	}
	go p.readLoop()
	return nil
}

func (p *Player) readLoop() {
	sc := bufio.NewScanner(p.conn)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Bytes()
		p.mu.Lock()
		ev, _ := applyLine(line, &p.state)
		p.mu.Unlock()
		if ev == "end-file" {
			select {
			case p.endCh <- struct{}{}:
			default:
			}
		}
	}
}

func (p *Player) send(b []byte) error {
	if p.conn == nil {
		return errors.New("player no iniciado")
	}
	_, err := p.conn.Write(b)
	return err
}

func (p *Player) Load(url string) error { return p.send(cmdLoad(url)) }
func (p *Player) SetPaused(v bool) error { return p.send(cmdSetPause(v)) }
func (p *Player) Seek(d int) error       { return p.send(cmdSeek(d)) }
func (p *Player) SetVolume(v int) error  { return p.send(cmdSetVolume(v)) }

func (p *Player) TogglePause() error {
	p.mu.Lock()
	paused := p.state.Paused
	p.mu.Unlock()
	return p.SetPaused(!paused)
}

func (p *Player) State() State {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.state
}

func (p *Player) EndCh() <-chan struct{} { return p.endCh }

func (p *Player) Close() error {
	if p.conn != nil {
		p.conn.Close()
	}
	if p.cmd != nil && p.cmd.Process != nil {
		return p.cmd.Process.Kill()
	}
	return nil
}
```

- [ ] **Step 7: Verificar que todo el paquete compila y los tests pasan**

Run:
```bash
go build ./... && go test ./internal/player/
```
Expected: build sin errores; tests PASA.

- [ ] **Step 8: Prueba manual (integración con mpv real)**

Se validará end-to-end en la Task 10 (arranque real con una URL). **Nota:** el arranque de mpv, la conexión al pipe y el auto-avance por `end-file` no se cubren con tests automáticos.

- [ ] **Step 9: Commit**

```bash
git add internal/player/ go.mod go.sum
git commit -m "feat: paquete player (proceso mpv + IPC JSON por named pipe)"
```

---

### Task 7: Helpers de formato de la TUI

**Files:**
- Create: `internal/tui/format.go`
- Test: `internal/tui/format_test.go`

**Interfaces:**
- Produces: `fmtTime(sec int) string`, `progressBar(pos, dur, width int) string`, `truncate(s string, max int) string`.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/tui/format_test.go`:
```go
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
}
```

- [ ] **Step 2: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/tui/
```
Expected: FALLA (símbolos indefinidos).

- [ ] **Step 3: Implementar los helpers**

`internal/tui/format.go`:
```go
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
```

- [ ] **Step 4: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/tui/
```
Expected: PASA.

- [ ] **Step 5: Commit**

```bash
git add internal/tui/format.go internal/tui/format_test.go
git commit -m "feat: helpers de formato de la TUI (tiempo, barra, truncado)"
```

---

### Task 8: Modelo Bubbletea y lógica de `Update`

**Files:**
- Create: `internal/tui/model.go`
- Create: `internal/tui/update.go`
- Test: `internal/tui/update_test.go`

**Interfaces:**
- Consumes: `queue.Queue`, `player.State`, `track.Track`; puertos (interfaces) para player/youtube/store definidos aquí.
- Produces:
  - Puertos: `playerPort`, `searchPort`, `storePort` (interfaces).
  - `type Model struct{ ... }` con `func New(q *queue.Queue, p playerPort, yt searchPort, st storePort, vol int) Model`.
  - `(Model) Init() tea.Cmd`, `(Model) Update(tea.Msg) (tea.Model, tea.Cmd)`.
  - Comandos/mensajes: `tickMsg`, `endFileMsg`, `searchResultMsg`.

- [ ] **Step 1: Añadir Bubbletea/Lipgloss**

Run:
```bash
go get github.com/charmbracelet/bubbletea@latest github.com/charmbracelet/lipgloss@latest
```
Expected: actualiza `go.mod`/`go.sum`.

- [ ] **Step 2: Escribir los tests que fallan**

`internal/tui/update_test.go`:
```go
package tui

import (
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
}

func newFakePlayer() *fakePlayer { return &fakePlayer{endCh: make(chan struct{}, 1)} }

func (f *fakePlayer) Load(u string) error      { f.loaded = u; return nil }
func (f *fakePlayer) TogglePause() error       { f.toggled++; return nil }
func (f *fakePlayer) Seek(d int) error         { f.seeked += d; return nil }
func (f *fakePlayer) SetVolume(v int) error    { f.volume = v; return nil }
func (f *fakePlayer) State() player.State      { return f.curState }
func (f *fakePlayer) EndCh() <-chan struct{}   { return f.endCh }

type fakeSearch struct{ result []track.Track }

func (f *fakeSearch) Search(string, int) ([]track.Track, error) { return f.result, nil }
func (f *fakeSearch) Resolve(string) ([]track.Track, error)     { return f.result, nil }

type fakeStore struct{ appended, favToggled int }

func (f *fakeStore) AppendHistory(track.Track) error         { f.appended++; return nil }
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
```

- [ ] **Step 3: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/tui/
```
Expected: FALLA (símbolos indefinidos).

- [ ] **Step 4: Implementar el modelo y sus puertos**

`internal/tui/model.go`:
```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AikonCWD/ytcli/internal/player"
	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/track"
)

// Ports keep the model testable with fakes and decouple it from concrete deps.
type playerPort interface {
	Load(url string) error
	TogglePause() error
	Seek(delta int) error
	SetVolume(v int) error
	State() player.State
	EndCh() <-chan struct{}
}

type searchPort interface {
	Search(query string, n int) ([]track.Track, error)
	Resolve(url string) ([]track.Track, error)
}

type storePort interface {
	AppendHistory(t track.Track) error
	ToggleFavorite(t track.Track) (bool, error)
	LoadHistory() ([]track.Track, error)
	LoadFavorites() ([]track.Track, error)
}

type mode int

const (
	modeCompact mode = iota
	modeExpanded
)

type tab int

const (
	tabQueue tab = iota
	tabSearch
	tabHistory
	tabFavorites
)

const (
	seekStep   = 10
	volumeStep = 5
	searchN    = 15
)

type Model struct {
	q      *queue.Queue
	player playerPort
	yt     searchPort
	store  storePort

	st      player.State
	vol     int
	prevVol int // for mute restore

	mode      mode
	tab       tab
	searching bool
	query     string
	cursor    int

	results   []track.Track
	history   []track.Track
	favorites []track.Track

	status string
	width  int
	height int
	quit   bool
}

func New(q *queue.Queue, p playerPort, yt searchPort, st storePort, vol int) Model {
	return Model{q: q, player: p, yt: yt, store: st, vol: vol, prevVol: vol, width: 46}
}
```

- [ ] **Step 5: Implementar `Init`, `Update` y comandos**

`internal/tui/update.go`:
```go
package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/track"
)

type tickMsg time.Time
type endFileMsg struct{}
type searchResultMsg struct {
	tracks []track.Track
	err    error
}

func tickCmd() tea.Cmd {
	return tea.Tick(250*time.Millisecond, func(t time.Time) tea.Msg { return tickMsg(t) })
}

func listenEnd(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg { <-ch; return endFileMsg{} }
}

func (m Model) searchCmd(query string) tea.Cmd {
	yt := m.yt
	return func() tea.Msg {
		tracks, err := yt.Search(query, searchN)
		return searchResultMsg{tracks: tracks, err: err}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), listenEnd(m.player.EndCh()))
}

func nextRepeat(r queue.RepeatMode) queue.RepeatMode {
	switch r {
	case queue.RepeatOff:
		return queue.RepeatAll
	case queue.RepeatAll:
		return queue.RepeatOne
	default:
		return queue.RepeatOff
	}
}

// playCurrent loads the queue's current track and records it in history.
func (m *Model) playCurrent() {
	if t, ok := m.q.Current(); ok {
		m.player.Load(t.URL)
		m.store.AppendHistory(t)
	}
}

func (m Model) activeList() []track.Track {
	switch m.tab {
	case tabSearch:
		return m.results
	case tabHistory:
		return m.history
	case tabFavorites:
		return m.favorites
	default:
		return m.q.Tracks()
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		m.st = m.player.State()
		return m, tickCmd()

	case endFileMsg:
		if _, ok := m.q.Next(); ok {
			m.playCurrent()
		}
		return m, listenEnd(m.player.EndCh())

	case searchResultMsg:
		if msg.err != nil {
			m.status = "Error de búsqueda: " + msg.err.Error()
			return m, nil
		}
		m.results = msg.tracks
		m.cursor = 0
		m.status = ""
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}
```

- [ ] **Step 6: Implementar el manejo de teclas**

`internal/tui/keys.go`:
```go
package tui

import (
	tea "github.com/charmbracelet/bubbletea"

	"github.com/AikonCWD/ytcli/internal/queue"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.searching {
		return m.handleSearchKey(msg)
	}

	switch msg.Type {
	case tea.KeySpace:
		m.player.TogglePause()
		return m, nil
	case tea.KeyLeft:
		m.player.Seek(-seekStep)
		return m, nil
	case tea.KeyRight:
		m.player.Seek(seekStep)
		return m, nil
	case tea.KeyUp:
		if m.mode == modeExpanded {
			if m.cursor > 0 {
				m.cursor--
			}
			return m, nil
		}
		return m.changeVolume(volumeStep)
	case tea.KeyDown:
		if m.mode == modeExpanded {
			if m.cursor < len(m.activeList())-1 {
				m.cursor++
			}
			return m, nil
		}
		return m.changeVolume(-volumeStep)
	case tea.KeyTab:
		if m.mode == modeCompact {
			m.mode = modeExpanded
		} else {
			m.mode = modeCompact
		}
		return m, nil
	case tea.KeyEnter:
		return m.playSelection()
	case tea.KeyCtrlC:
		m.quit = true
		return m, tea.Quit
	}

	if len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case 'q':
			m.quit = true
			return m, tea.Quit
		case 'n':
			if _, ok := m.q.Next(); ok {
				m.playCurrent()
			}
			return m, nil
		case 'p':
			if _, ok := m.q.Prev(); ok {
				m.playCurrent()
			}
			return m, nil
		case '+':
			return m.changeVolume(volumeStep)
		case '-':
			return m.changeVolume(-volumeStep)
		case 'm':
			return m.toggleMute()
		case 's':
			m.q.SetShuffle(!m.q.Shuffle())
			return m, nil
		case 'r':
			m.q.SetRepeat(nextRepeat(m.q.Repeat()))
			return m, nil
		case 'f':
			if t, ok := m.q.Current(); ok {
				m.store.ToggleFavorite(t)
			}
			return m, nil
		case '/':
			m.mode = modeExpanded
			m.tab = tabSearch
			m.searching = true
			m.query = ""
			return m, nil
		}
	}
	return m, nil
}

func (m Model) changeVolume(delta int) (tea.Model, tea.Cmd) {
	m.vol += delta
	if m.vol > 100 {
		m.vol = 100
	}
	if m.vol < 0 {
		m.vol = 0
	}
	m.player.SetVolume(m.vol)
	return m, nil
}

func (m Model) toggleMute() (tea.Model, tea.Cmd) {
	if m.vol > 0 {
		m.prevVol = m.vol
		m.vol = 0
	} else {
		m.vol = m.prevVol
	}
	m.player.SetVolume(m.vol)
	return m, nil
}

// playSelection plays the highlighted item of the active list (expanded mode).
func (m Model) playSelection() (tea.Model, tea.Cmd) {
	if m.mode != modeExpanded {
		return m, nil
	}
	list := m.activeList()
	if m.cursor < 0 || m.cursor >= len(list) {
		return m, nil
	}
	if m.tab == tabQueue {
		m.q.JumpTo(m.cursor)
		m.playCurrent()
		return m, nil
	}
	m.q.Add(list[m.cursor])
	m.q.JumpTo(m.q.Len() - 1)
	m.playCurrent()
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.searching = false
		return m, m.searchCmd(m.query)
	case tea.KeyEsc:
		m.searching = false
		m.query = ""
		return m, nil
	case tea.KeyBackspace:
		if r := []rune(m.query); len(r) > 0 {
			m.query = string(r[:len(r)-1])
		}
		return m, nil
	case tea.KeyRunes, tea.KeySpace:
		if msg.Type == tea.KeySpace {
			m.query += " "
		} else {
			m.query += string(msg.Runes)
		}
		return m, nil
	}
	return m, nil
}

var _ = queue.RepeatOff // keep queue import used if keys.go trimmed
```

> Nota de implementación: si `keys.go` deja de referenciar el paquete `queue`, elimina la línea `var _ = queue.RepeatOff` y el import. Está solo para que el archivo compile tal cual si se pega aislado.

- [ ] **Step 7: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/tui/
```
Expected: PASA (todos los tests de teclas y mensajes).

- [ ] **Step 8: Commit**

```bash
git add internal/tui/model.go internal/tui/update.go internal/tui/keys.go internal/tui/update_test.go go.mod go.sum
git commit -m "feat: modelo Bubbletea, Update y manejo de teclas"
```

---

### Task 9: Vistas compacta y expandida

**Files:**
- Create: `internal/tui/view.go`
- Test: `internal/tui/view_test.go`

**Interfaces:**
- Consumes: `Model` y sus helpers de formato.
- Produces: `(Model) View() string`, con render compacto (~4-5 líneas con borde Lipgloss) y panel expandido con pestañas.

- [ ] **Step 1: Escribir los tests que fallan**

`internal/tui/view_test.go`:
```go
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
```

- [ ] **Step 2: Ejecutar los tests para ver que fallan**

Run:
```bash
go test ./internal/tui/ -run TestCompactView
go test ./internal/tui/ -run TestExpandedView
```
Expected: FALLA (no existe `View`).

- [ ] **Step 3: Implementar las vistas**

`internal/tui/view.go`:
```go
package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	selStyle    = lipgloss.NewStyle().Reverse(true)
	tabActive   = lipgloss.NewStyle().Bold(true).Underline(true)
)

func (m Model) repeatLabel() string {
	switch m.q.Repeat() {
	case queue.RepeatAll:
		return "all"
	case queue.RepeatOne:
		return "one"
	default:
		return "off"
	}
}

func (m Model) View() string {
	if m.quit {
		return ""
	}
	compact := m.compactView()
	if m.mode == modeExpanded {
		return compact + "\n" + m.expandedView()
	}
	return compact
}

func (m Model) compactView() string {
	inner := m.width - 4
	if inner < 20 {
		inner = 20
	}

	title := "—"
	if t, ok := m.q.Current(); ok {
		title = t.Title
		if title == "" {
			title = m.st.Title
		}
	}
	line1 := "♪  " + truncate(title, inner-3)

	playIcon := "▶"
	if m.st.Paused {
		playIcon = "⏸"
	}
	bar := progressBar(m.st.Position, m.st.Duration, inner-20)
	line2 := fmt.Sprintf("%s  %s  %s / %s", playIcon, bar,
		fmtTime(m.st.Position), fmtTime(m.st.Duration))

	line3 := fmt.Sprintf("🔊 %d%%   🔁 %s   ⧉ %d/%d",
		m.vol, m.repeatLabel(), m.q.CurrentIndex()+1, m.q.Len())

	status := m.status
	if status == "" {
		status = dimStyle.Render("space ⏯  ←→ seek  n/p  / buscar  tab ⤢  q salir")
	}

	body := strings.Join([]string{line1, line2, line3, status}, "\n")
	return borderStyle.Width(inner).Render(body)
}

func (m Model) expandedView() string {
	labels := []string{"Cola", "Buscar", "Historial", "Favoritos"}
	var tabs []string
	for i, l := range labels {
		if tab(i) == m.tab {
			tabs = append(tabs, tabActive.Render(l))
		} else {
			tabs = append(tabs, dimStyle.Render(l))
		}
	}
	header := strings.Join(tabs, "   ")

	var rows []string
	if m.tab == tabSearch {
		prompt := "🔎 " + m.query
		if m.searching {
			prompt += "▏"
		}
		rows = append(rows, prompt)
	}

	list := m.activeList()
	maxRows := 10
	for i, t := range list {
		if i >= maxRows {
			rows = append(rows, dimStyle.Render(fmt.Sprintf("… y %d más", len(list)-maxRows)))
			break
		}
		line := truncate(t.Title, m.width-6)
		if i == m.cursor && !m.searching {
			line = selStyle.Render("› " + line)
		} else {
			line = "  " + line
		}
		rows = append(rows, line)
	}
	if len(list) == 0 && m.tab != tabSearch {
		rows = append(rows, dimStyle.Render("(vacío)"))
	}

	body := header + "\n" + strings.Join(rows, "\n")
	return borderStyle.Width(m.width - 4).Render(body)
}
```

> Nota: este archivo referencia el paquete `queue` (en `repeatLabel`). Añade el import `"github.com/AikonCWD/ytcli/internal/queue"` al bloque de imports de `view.go`.

- [ ] **Step 4: Corregir imports y compilar**

Añade `"github.com/AikonCWD/ytcli/internal/queue"` a los imports de `view.go`. Luego:

Run:
```bash
go build ./...
```
Expected: compila sin errores.

- [ ] **Step 5: Ejecutar los tests para ver que pasan**

Run:
```bash
go test ./internal/tui/
```
Expected: PASA (incluidos los tests de vista).

- [ ] **Step 6: Commit**

```bash
git add internal/tui/view.go internal/tui/view_test.go
git commit -m "feat: vistas compacta y expandida de la TUI"
```

---

### Task 10: `main` — CLI, bootstrap y arranque end-to-end

**Files:**
- Create: `main.go`
- Create: `README.md`

**Interfaces:**
- Consumes: `deps`, `store`, `youtube`, `player`, `queue`, `tui`.
- Produces: binario `ytcli.exe`.

- [ ] **Step 1: Implementar `main`**

`main.go`:
```go
// Command ytcli is a terminal UI that streams audio from YouTube via mpv + yt-dlp.
package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AikonCWD/ytcli/internal/deps"
	"github.com/AikonCWD/ytcli/internal/player"
	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/store"
	"github.com/AikonCWD/ytcli/internal/tui"
	"github.com/AikonCWD/ytcli/internal/youtube"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ytcli:", err)
		os.Exit(1)
	}
}

func run(urls []string) error {
	binDir, err := deps.BinDir()
	if err != nil {
		return err
	}
	fmt.Println("ytcli: preparando dependencias…")
	paths, err := deps.Ensure(binDir, func(msg string) { fmt.Println("  " + msg) })
	if err != nil {
		return err
	}

	dataDir, err := store.DefaultDir()
	if err != nil {
		return err
	}
	st := store.New(dataDir)
	yt := youtube.New(paths.YtDlp)

	p := player.New(paths.Mpv)
	if err := p.Start(); err != nil {
		return err
	}
	defer p.Close()

	q := queue.New()
	for _, u := range urls {
		tracks, err := yt.Resolve(u)
		if err != nil {
			fmt.Fprintln(os.Stderr, "  aviso: no se pudo resolver", u, "-", err)
			continue
		}
		q.Add(tracks...)
	}
	if t, ok := q.Current(); ok {
		p.Load(t.URL)
		st.AppendHistory(t)
	}

	m := tui.New(q, p, yt, st, 80)
	prog := tea.NewProgram(m) // modo inline (sin alt-screen): footprint mínimo
	_, err = prog.Run()
	return err
}
```

- [ ] **Step 2: Compilar el binario**

Run:
```bash
go build -o ytcli.exe .
```
Expected: genera `ytcli.exe` sin errores.

- [ ] **Step 3: Verificar el conjunto de tests**

Run:
```bash
go test ./...
```
Expected: PASA en todos los paquetes.

- [ ] **Step 4: Prueba manual end-to-end**

Ejecuta con una URL real (requiere red; la primera vez descargará mpv/yt-dlp):
```bash
./ytcli.exe "https://www.youtube.com/watch?v=dQw4w9WgXcQ"
```
Verifica:
- Se descargan mpv/yt-dlp la primera vez y suena el audio.
- `space` pausa/reanuda; `←`/`→` saltan ±10s; `↑`/`↓` cambian volumen.
- `/` abre búsqueda, se escribe una consulta, `Enter` lista resultados, flechas + `Enter` reproducen.
- `tab` alterna compacto/expandido; `f` marca favorito; `q` sale.
- Con una playlist, al terminar una pista avanza sola a la siguiente.

- [ ] **Step 5: Escribir el README**

`README.md`:
```markdown
# ytcli

Reproductor de música TUI para YouTube (solo audio). Escrito en Go, usa mpv
(streaming) y yt-dlp (búsqueda y metadatos), que se auto-descargan en el primer
arranque en Windows.

## Uso

    ytcli                      # abre vacío; usa / para buscar
    ytcli <url>                # reproduce un vídeo
    ytcli <url-playlist>       # encola y reproduce una playlist
    ytcli <url1> <url2> ...    # encola varias

## Controles

| Tecla | Acción |
|-------|--------|
| Espacio | Play / Pausa |
| ← / → | Seek ∓10s |
| n / p | Siguiente / anterior |
| ↑ / ↓ (o + / -) | Volumen |
| m | Mute |
| s | Shuffle |
| r | Repeat (off/all/one) |
| / | Buscar |
| Enter | Reproducir selección |
| f | Favorito |
| Tab | Compacto / expandido |
| q | Salir |

## Construir

    go build -o ytcli.exe .
```

- [ ] **Step 6: Commit**

```bash
git add main.go README.md
git commit -m "feat: main con bootstrap de dependencias y arranque de la TUI"
```

---

## Self-Review

**1. Cobertura de la spec:**
- Interfaz compacta/expandida → Task 9. ✓
- Controles (play/pausa, seek, next/prev, volumen, mute, shuffle, repeat, buscar, favorito, tab, salir) → Task 8 (`keys.go`). ✓
- Uso desde CLI (URLs/playlists/varias) → Task 10 (`run`). ✓
- Streaming directo vía mpv → Task 6. ✓
- Búsqueda por texto (yt-dlp `ytsearch`, sin API key) → Task 4 + Task 8 (`searchCmd`). ✓
- Persistencia historial + favoritos → Task 3, consumida en Tasks 8 y 10. ✓
- Auto-descarga de dependencias → Task 5. ✓
- Manejo de errores: vídeo no resoluble se salta con aviso (Task 10 `run`); error de búsqueda a barra de estado (Task 8); descarga fallida aborta con mensaje (Task 5). ✓
- Pruebas TDD por paquete (`queue`, `store`, `youtube`, `deps`, `player`, `tui`) → cubierto; integración de mpv/red marcada como manual. ✓
- Modo comando vs texto (teclas no interfieren al escribir en buscador) → Task 8 (`handleSearchKey` captura runas; `handleKey` delega si `m.searching`). ✓

**2. Escaneo de placeholders:** Sin "TBD/TODO/implementar luego". Las dos notas de implementación (import de `queue` en `view.go`, línea `var _` en `keys.go`) son instrucciones concretas, no placeholders.

**3. Consistencia de tipos:** `track.Track` uniforme en todos los paquetes. `player.State` con los mismos campos en definición (Task 6) y en el puerto/tests (Task 8). `queue.RepeatMode` y sus constantes usadas igual en Tasks 2, 8, 9. Firmas de `store` idénticas entre Task 3 y el `storePort` de Task 8. `deps.Paths{Mpv, YtDlp}` consistente entre Tasks 5 y 10.

**Ajuste aplicado durante la revisión:** el test `TestNextLoadsAndRecordsHistory` asume que `n` avanza y carga; `playCurrent` carga `Current()` tras `Next()`, coherente. `TestVolumeKeys` espera 85 (80+5), coherente con `volumeStep=5` y `vol` inicial 80.
