// Command ytcli is a terminal UI that streams audio from YouTube via mpv + yt-dlp.
package main

import (
	"fmt"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/AikonCWD/ytcli/internal/deps"
	"github.com/AikonCWD/ytcli/internal/player"
	"github.com/AikonCWD/ytcli/internal/queue"
	"github.com/AikonCWD/ytcli/internal/store"
	"github.com/AikonCWD/ytcli/internal/track"
	"github.com/AikonCWD/ytcli/internal/tui"
	"github.com/AikonCWD/ytcli/internal/youtube"
)

const (
	initialVolume = 80
	version       = "1.2.0"
)

const helpText = `ytcli — reproductor de música TUI para YouTube (solo audio)

USO
  ytcli [opciones] [url | palabras de búsqueda]...

  Sin argumentos abre la playlist guardada (playlist.txt). Las URLs se
  encolan (las playlists de YouTube se expanden); cualquier otra palabra
  se busca en YouTube y se reproduce el primer resultado.

EJEMPLOS
  ytcli                                reanuda la playlist guardada
  ytcli https://youtu.be/xxxxxxxxxxx   encola y reproduce un vídeo o directo
  ytcli <url-playlist>                 encola una playlist completa
  ytcli <url1> <url2> ...              encola varias URLs
  ytcli lofi girl radio                busca y reproduce el primer resultado

OPCIONES
  -h, --help      muestra esta ayuda y sale
  -v, --version   muestra la versión y sale

ATAJOS · modo compacto
  espacio         play / pausa
  ← / →           retroceder / avanzar 10 s
  ↑ / ↓  ó  + / - subir / bajar volumen
  n / p           pista siguiente / anterior
  m               silenciar (mute)
  s               shuffle on/off
  r               repeat off → all → one
  f               favorito de la pista actual
  /               buscar en YouTube
  ?               ayuda (esta chuleta, dentro de la TUI)
  tab             abrir la vista expandida
  q · ctrl+c      salir

ATAJOS · vista expandida
  1 2 3 4         pestañas Cola · Buscar · Historial · Favoritos
  ↑ / ↓           mover la selección
  enter           reproducir la selección
  d / supr        quitar de la cola (pestaña Cola)
  f               favorito de la selección
  esc / tab       volver al modo compacto

FICHEROS (junto al ejecutable; texto editable, una pista por línea)
  playlist.txt    la cola: se carga al arrancar y se guarda al cambiarla
  history.txt     historial (máx. 200, más reciente primero)
  favorites.txt   favoritos

  Basta con pegar una URL de YouTube por línea; el título, canal y
  duración se rellenan solos al reproducirla.

DEPENDENCIAS
  mpv y yt-dlp se descargan automáticamente en el primer arranque
  (quedan en %LOCALAPPDATA%\ytcli\bin).
`

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "ytcli:", err)
		os.Exit(1)
	}
}

// resolveArgs turns CLI args into tracks: URLs (with or without scheme) are
// resolved (playlists expand), everything else is joined into a single search
// query playing the top match.
func resolveArgs(yt *youtube.Client, args []string, warn func(string)) []track.Track {
	var out []track.Track
	var words []string
	for _, a := range args {
		if store.LooksLikeURL(a) {
			ts, err := yt.Resolve(a)
			if err != nil {
				warn("no se pudo resolver " + a + " - " + err.Error())
				continue
			}
			out = append(out, ts...)
			continue
		}
		words = append(words, a)
	}
	if len(words) > 0 {
		q := strings.Join(words, " ")
		ts, err := yt.Search(q, 1)
		if err != nil || len(ts) == 0 {
			warn("sin resultados para «" + q + "»")
		} else {
			out = append(out, ts[0])
		}
	}
	return out
}

func run(args []string) error {
	for _, a := range args {
		switch a {
		case "-h", "--help", "help":
			fmt.Print(helpText)
			return nil
		case "-v", "--version":
			fmt.Println("ytcli " + version)
			return nil
		}
	}
	// Reject unknown flags instead of sending them to the YouTube search.
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return fmt.Errorf("opción desconocida %q (usa --help para ver la ayuda)", a)
		}
	}

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
	// Import history/favorites from the old %APPDATA% JSON store, if any.
	if appData := os.Getenv("APPDATA"); appData != "" {
		_ = st.MigrateJSON(appData + `\ytcli`)
	}
	yt := youtube.New(paths.YtDlp)

	q := queue.New()
	saved, err := st.LoadPlaylist()
	if err != nil {
		fmt.Fprintln(os.Stderr, "  aviso: no se pudo leer playlist.txt -", err)
	}
	q.Add(saved...)

	if len(args) > 0 {
		fmt.Println("ytcli: resolviendo enlaces…")
	}
	warn := func(msg string) { fmt.Fprintln(os.Stderr, "  aviso: "+msg) }
	jumpTo := -1
	for _, t := range resolveArgs(yt, args, warn) {
		idx := q.IndexOfID(t.ID)
		if idx == -1 {
			q.Add(t)
			idx = q.Len() - 1
		}
		if jumpTo == -1 {
			jumpTo = idx
		}
	}
	if jumpTo >= 0 {
		q.JumpTo(jumpTo)
	}
	if q.Len() > 0 {
		if err := st.SavePlaylist(q.OriginalTracks()); err != nil {
			fmt.Fprintln(os.Stderr, "  aviso: no se pudo guardar playlist.txt -", err)
		}
	}

	p := player.New(paths.Mpv)
	if err := p.Start(); err != nil {
		return err
	}
	defer p.Close()

	if t, ok := q.Current(); ok {
		if err := p.Load(t.URL); err != nil {
			fmt.Fprintln(os.Stderr, "  aviso: no se pudo iniciar la reproducción:", err)
		} else {
			st.AppendHistory(t)
		}
	}

	m := tui.New(q, p, yt, st, initialVolume)
	prog := tea.NewProgram(m) // modo inline (sin alt-screen): footprint mínimo
	_, err = prog.Run()
	return err
}
