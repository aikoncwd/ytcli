package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"

	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/track"
)

// Sistema visual: emoji = icono de estado (🎵 🔊 🔇 🔁 🔂 🔀 🎶 🔎 🔴);
// glifo fino coloreado = elemento estructural (barra ━●─, marcador ♪, cursor ›).
// Los valores activos van en color de acento y los inactivos atenuados.
var (
	// accent is a muted red (ANSI 256 #d75f5f): fits the YouTube theme without
	// shouting, and degrades gracefully on limited color profiles.
	accent = lipgloss.Color("167")
	// tabColors gives each expanded tab its own accent; the frame border and
	// the active tab label follow it, so TAB/1-4 recolor the whole box.
	tabColors = map[tab]lipgloss.Color{
		tabQueue:     lipgloss.Color("167"), // rojo
		tabSearch:    lipgloss.Color("75"),  // azul
		tabHistory:   lipgloss.Color("179"), // ámbar
		tabFavorites: lipgloss.Color("213"), // rosa
	}
	borderStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).
			BorderForeground(accent).Padding(0, 1)
	dimStyle    = lipgloss.NewStyle().Faint(true)
	selStyle    = lipgloss.NewStyle().Reverse(true)
	liveStyle   = lipgloss.NewStyle().Bold(true).Foreground(accent)
	accentStyle = lipgloss.NewStyle().Foreground(accent)
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

// innerWidth is the lipgloss block width (padding included, borders excluded);
// content must fit innerWidth-2 cells or lines wrap and the box height becomes
// unstable, which breaks bubbletea's inline renderer.
func (m Model) innerWidth() int {
	inner := m.width - 4
	if inner < 26 {
		inner = 26
	}
	return inner
}

func (m Model) View() string {
	if m.quit {
		return ""
	}
	compact := m.compactView()
	if m.showHelp {
		return compact + "\n" + m.helpView()
	}
	if m.mode == modeExpanded {
		return compact + "\n" + m.expandedView()
	}
	return compact
}

// onOff styles a status value: accent when active, dimmed when off.
func onOff(label string, active bool) string {
	if active {
		return accentStyle.Render(label)
	}
	return dimStyle.Render(label)
}

func (m Model) compactView() string {
	inner := m.innerWidth()
	contentW := inner - 2

	cur, hasCur := m.q.Current()
	title := "—"
	if hasCur {
		title = cur.Title
		if title == "" {
			title = m.st.Title
		}
		if title == "" {
			title = cur.URL
		}
	}
	line1 := "🎵 " + truncate(title, contentW-3)

	playIcon := "▶"
	if m.st.Paused {
		playIcon = "⏸"
	}
	var line2 string
	if hasCur && cur.Live {
		line2 = playIcon + "  " + liveStyle.Render("🔴 EN DIRECTO")
	} else {
		times := fmtTime(m.st.Position) + " / " + fmtTime(m.st.Duration)
		barW := contentW - ansi.StringWidth(playIcon) - 4 - ansi.StringWidth(times)
		line2 = fmt.Sprintf("%s  %s  %s", accentStyle.Render(playIcon),
			progressBar(m.st.Position, m.st.Duration, barW), dimStyle.Render(times))
	}
	line2 = truncate(line2, contentW) // barW can go negative on tiny widths

	volIcon := "🔊"
	if m.vol == 0 {
		volIcon = "🔇"
	}
	repIcon := "🔁"
	if m.q.Repeat() == queue.RepeatOne {
		repIcon = "🔂"
	}
	line3 := truncate(strings.Join([]string{
		volIcon + " " + onOff(fmt.Sprintf("%d%%", m.vol), m.vol > 0),
		repIcon + " " + onOff(m.repeatLabel(), m.q.Repeat() != queue.RepeatOff),
		"🔀 " + onOff(map[bool]string{true: "on", false: "off"}[m.q.Shuffle()], m.q.Shuffle()),
		fmt.Sprintf("🎶 %d/%d", m.q.CurrentIndex()+1, m.q.Len()),
	}, "   "), contentW)

	status := truncate(m.status, contentW)
	if status == "" {
		// Adaptive hint: fall back to the essentials when the long one doesn't fit.
		hint := "space pausa · ←/→ avanzar · n/p pista · / buscar · tab lista · ? ayuda · q salir"
		if ansi.StringWidth(hint) > contentW {
			hint = "space pausa · / buscar · ? ayuda · q salir"
		}
		status = dimStyle.Render(truncate(hint, contentW))
	}

	body := strings.Join([]string{line1, line2, line3, status}, "\n")
	return borderStyle.Width(inner).Render(body)
}

// listRows returns how many list entries fit the expanded box for the current
// terminal height (compact box + borders, header, prompt, footer ≈ 12 lines).
// Capped because legacy conhost reports the scrollback-buffer height (e.g.
// 9001) in resize events, not the window height.
func (m Model) listRows() int {
	if m.height <= 0 {
		return 10
	}
	rows := m.height - 12
	if rows < 4 {
		return 4
	}
	if rows > 40 {
		return 40
	}
	return rows
}

// footerHelp picks the key hint for tab t, falling back to a condensed
// variant when the full one doesn't fit contentW.
func footerHelp(t tab, contentW int) string {
	long, short := "↑↓ mover · enter añadir y reproducir · f fav · esc/tab volver",
		"↑↓ · enter reproducir · f fav · esc volver"
	if t == tabQueue {
		long, short = "↑↓ mover · enter reproducir · d quitar · f fav · esc/tab volver",
			"↑↓ · enter · d quitar · f fav · esc volver"
	}
	if ansi.StringWidth(long) > contentW {
		return short
	}
	return long
}

// listRow renders one entry: title plus dim channel/duration metadata.
func (m Model) listRow(t track.Track, i, contentW int) string {
	textW := contentW - 2
	title := t.Title
	if title == "" {
		title = t.URL
	}
	var parts []string
	if t.Channel != "" {
		parts = append(parts, t.Channel)
	}
	if t.Live {
		parts = append(parts, "🔴 directo")
	} else if t.Duration > 0 {
		parts = append(parts, fmtTime(t.Duration))
	}
	meta := ""
	if len(parts) > 0 {
		meta = " · " + strings.Join(parts, " · ")
	}
	metaW := ansi.StringWidth(meta)
	if metaW > textW/2 { // keep room for the title on narrow widths
		meta = ""
		metaW = 0
	}
	star := "" // favorites marker; redundant inside the Favoritos tab itself
	if m.tab != tabFavorites && m.favIDs[t.ID] {
		star = " ⭐"
	}
	title = truncate(title, textW-metaW-ansi.StringWidth(star))

	if i == m.cursor && !m.searching {
		return selStyle.Render("› " + title + star + meta)
	}
	prefix := "  "
	if m.tab == tabQueue && i == m.q.CurrentIndex() {
		prefix = accentStyle.Render("♪ ")
	}
	return prefix + title + star + dimStyle.Render(meta)
}

func (m Model) expandedView() string {
	inner := m.innerWidth()
	contentW := inner - 2
	color := tabColors[m.tab]

	labels := []string{"1 Cola", "2 Buscar", "3 Historial", "4 Favoritos"}
	var tabs []string
	for i, l := range labels {
		if tab(i) == m.tab {
			active := lipgloss.NewStyle().Bold(true).Underline(true).Foreground(color)
			tabs = append(tabs, active.Render(l))
		} else {
			tabs = append(tabs, dimStyle.Render(l))
		}
	}
	header := truncate(strings.Join(tabs, "   "), contentW)

	var rows []string
	if m.tab == tabSearch {
		prompt := "🔎 " + m.query
		if m.searching {
			prompt += "▏"
		}
		rows = append(rows, truncate(prompt, contentW))
	}

	list := m.activeList()
	maxRows := m.listRows()
	for i, t := range list {
		if i >= maxRows {
			rows = append(rows, dimStyle.Render(fmt.Sprintf("… y %d más", len(list)-maxRows)))
			break
		}
		rows = append(rows, m.listRow(t, i, contentW))
	}
	if len(list) == 0 {
		switch {
		case m.tab == tabSearch && !m.searching:
			rows = append(rows, dimStyle.Render("(sin resultados — pulsa / para buscar)"))
		case m.tab != tabSearch:
			rows = append(rows, dimStyle.Render("(vacío)"))
		}
	}

	footer := dimStyle.Render(truncate(footerHelp(m.tab, contentW), contentW))
	body := header + "\n" + strings.Join(rows, "\n") + "\n" + footer
	return borderStyle.BorderForeground(color).Width(inner).Render(body)
}

// helpKeys is the in-TUI cheat sheet; column one is the key, column two the action.
var helpKeys = [][2]string{
	{"espacio", "play / pausa"},
	{"← / →", "retroceder / avanzar 10 s"},
	{"↑ / ↓  + / -", "volumen · mover selección (expandido)"},
	{"n / p", "pista siguiente / anterior"},
	{"m", "silenciar (mute)"},
	{"s", "shuffle on/off"},
	{"r", "repeat off → all → one"},
	{"f", "favorito ⭐"},
	{"/", "buscar en YouTube"},
	{"1 2 3 4", "pestañas Cola · Buscar · Historial · Favoritos"},
	{"enter", "reproducir la selección"},
	{"d / supr", "quitar de la cola (pestaña Cola)"},
	{"tab", "compacto ⇄ expandido"},
	{"esc", "volver"},
	{"q · ctrl+c", "salir"},
}

// helpView renders the modal cheat sheet in the same frame as the expanded view.
func (m Model) helpView() string {
	inner := m.innerWidth()
	contentW := inner - 2

	header := lipgloss.NewStyle().Bold(true).Underline(true).Foreground(accent).
		Render(truncate("Ayuda · atajos de teclado", contentW))
	maxRows := m.listRows() + 3 // no prompt/meta rows: the cheat sheet can run taller
	var rows []string
	for i, kv := range helpKeys {
		if i >= maxRows {
			rows = append(rows, dimStyle.Render(fmt.Sprintf("… y %d más", len(helpKeys)-maxRows)))
			break
		}
		key := accentStyle.Render(fmt.Sprintf("%-13s", kv[0]))
		rows = append(rows, truncate(key+kv[1], contentW))
	}
	footer := dimStyle.Render(truncate("pulsa cualquier tecla para volver · q salir", contentW))
	body := header + "\n" + strings.Join(rows, "\n") + "\n" + footer
	return borderStyle.Width(inner).Render(body)
}
