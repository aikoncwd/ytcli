package tui

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/track"
)

type tickMsg time.Time
type endFileMsg struct{}
type playerLostMsg struct{}
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

func listenLost(ch <-chan struct{}) tea.Cmd {
	return func() tea.Msg { <-ch; return playerLostMsg{} }
}

func (m Model) searchCmd(query string) tea.Cmd {
	yt := m.yt
	return func() tea.Msg {
		tracks, err := yt.Search(query, searchN)
		return searchResultMsg{tracks: tracks, err: err}
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), listenEnd(m.player.EndCh()), listenLost(m.player.LostCh()))
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
	t, ok := m.q.Current()
	if !ok {
		return
	}
	if err := m.player.Load(t.URL); err != nil {
		m.status = "Error al reproducir: " + err.Error()
		return
	}
	m.store.AppendHistory(t)
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

// selectTab switches to tab t in expanded mode, loading store-backed lists.
func (m Model) selectTab(t tab) (tea.Model, tea.Cmd) {
	m.mode = modeExpanded
	m.tab = t
	m.cursor = 0
	switch t {
	case tabHistory:
		m.history, _ = m.store.LoadHistory()
	case tabFavorites:
		m.favorites, _ = m.store.LoadFavorites()
	}
	return m, nil
}

// toggleFavorite favorites the selected list item (expanded, non-queue tab) or
// else the current track, and refreshes the favorites list if it's showing.
func (m Model) toggleFavorite() (tea.Model, tea.Cmd) {
	var t track.Track
	var ok bool
	list := m.activeList()
	if m.mode == modeExpanded && m.tab != tabQueue && m.cursor >= 0 && m.cursor < len(list) {
		t, ok = list[m.cursor], true
	} else {
		t, ok = m.q.Current()
	}
	if !ok {
		return m, nil
	}
	if _, err := m.store.ToggleFavorite(t); err != nil {
		m.status = "Error al marcar favorito: " + err.Error()
		return m, nil
	}
	if m.tab == tabFavorites {
		m.favorites, _ = m.store.LoadFavorites()
		if m.cursor >= len(m.favorites) {
			m.cursor = len(m.favorites) - 1
		}
		if m.cursor < 0 {
			m.cursor = 0
		}
	}
	return m, nil
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

	case playerLostMsg:
		m.status = "Conexión con mpv perdida"
		return m, nil

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
