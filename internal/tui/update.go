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
