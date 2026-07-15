package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.showHelp {
		return m.handleHelpKey(msg)
	}
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
			return m, m.setMode(modeExpanded)
		}
		return m, m.setMode(modeCompact)
	case tea.KeyEsc:
		return m, m.setMode(modeCompact)
	case tea.KeyDelete:
		return m.removeSelection()
	case tea.KeyEnter:
		return m.playSelection()
	case tea.KeyCtrlC:
		m.quit = true
		return m, tea.Quit
	}

	if len(msg.Runes) == 1 {
		switch msg.Runes[0] {
		case ' ':
			m.player.TogglePause()
			return m, nil
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
			return m.toggleFavorite()
		case 'd':
			return m.removeSelection()
		case '/':
			cmd := m.setMode(modeExpanded)
			m.tab = tabSearch
			m.searching = true
			m.query = ""
			return m, cmd
		case '?':
			return m.openHelp()
		case '1':
			return m.selectTab(tabQueue)
		case '2':
			return m.selectTab(tabSearch)
		case '3':
			return m.selectTab(tabHistory)
		case '4':
			return m.selectTab(tabFavorites)
		}
	}
	return m, nil
}

// openHelp shows the modal help panel; it lives in the alt screen, so the
// previous mode is remembered and restored on close.
func (m Model) openHelp() (tea.Model, tea.Cmd) {
	m.helpFrom = m.mode
	m.showHelp = true
	return m, m.setMode(modeExpanded)
}

// handleHelpKey makes the help panel modal: q/ctrl+c still quit, any other
// key closes it and restores the mode it was opened from.
func (m Model) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.Type == tea.KeyCtrlC || (len(msg.Runes) == 1 && msg.Runes[0] == 'q') {
		m.quit = true
		return m, tea.Quit
	}
	m.showHelp = false
	return m, m.setMode(m.helpFrom)
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
// Items from search/history/favorites already in the queue are jumped to
// instead of re-added, so playlist.txt never accumulates duplicates.
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
	t := list[m.cursor]
	if idx := m.q.IndexOfID(t.ID); idx >= 0 {
		m.q.JumpTo(idx)
		m.playCurrent()
		return m, nil
	}
	m.q.Add(t)
	m.q.JumpTo(m.q.Len() - 1)
	m.playCurrent()
	m.savePlaylist()
	return m, nil
}

// removeSelection deletes the highlighted queue item (expanded Cola tab only).
// Removing the playing track loads the next one (or stops mpv on an empty
// queue) so the player never keeps playing a track the queue no longer has.
func (m Model) removeSelection() (tea.Model, tea.Cmd) {
	if m.mode != modeExpanded || m.tab != tabQueue {
		return m, nil
	}
	wasCurrent := m.cursor == m.q.CurrentIndex()
	if !m.q.RemoveAt(m.cursor) {
		return m, nil
	}
	if wasCurrent {
		if m.q.Len() > 0 {
			m.playCurrent()
		} else {
			m.player.Stop()
		}
	}
	m.clampCursor(m.q.Len())
	m.savePlaylist()
	return m, nil
}

func (m Model) handleSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyCtrlC:
		m.quit = true
		return m, tea.Quit
	case tea.KeyEnter:
		m.searching = false
		if strings.TrimSpace(m.query) == "" {
			return m, nil
		}
		m.status = "Buscando…"
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
