package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/AikonCWD/ytcli/internal/queue"
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
