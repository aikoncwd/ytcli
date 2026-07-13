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
