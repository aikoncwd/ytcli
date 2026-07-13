package tui

import (
	tea "github.com/charmbracelet/bubbletea"
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
