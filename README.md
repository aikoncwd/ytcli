# ytcli

> Reproductor de música TUI para YouTube — solo audio, directo en tu terminal.

[![Go](https://img.shields.io/badge/Go-1.26-00ADD8?logo=go&logoColor=white)](https://go.dev/)
[![Platform](https://img.shields.io/badge/plataforma-Windows-0078D6?logo=windows&logoColor=white)](#requisitos)
[![License: MIT](https://img.shields.io/badge/licencia-MIT-green.svg)](LICENSE)

**ytcli** convierte tu terminal en un reproductor de música alimentado por
YouTube. Busca, encola y reproduce cualquier vídeo, playlist o directo — sin
navegador, sin anuncios visuales y sin gastar ancho de banda en vídeo: solo
se descarga el audio en streaming.

Bajo el capó usa [mpv](https://mpv.io/) para reproducir y
[yt-dlp](https://github.com/yt-dlp/yt-dlp) para buscar y resolver metadatos.
**No necesitas instalar nada**: ambos se descargan automáticamente la primera
vez que arrancas.

## Características

- 🎵 **Solo audio** — streaming eficiente, sin decodificar vídeo
- 🔍 **Búsqueda integrada** — busca en YouTube sin salir del terminal (`/`)
- 📃 **Playlists de YouTube** — pásale la URL y se expande entera en la cola
- 📻 **Directos** — los livestreams también funcionan
- ⭐ **Favoritos e historial** — persistentes entre sesiones
- 🔀 **Shuffle y repeat** — off / all / one
- 🗂️ **Datos en texto plano** — playlist, historial y favoritos son ficheros
  `.txt` editables a mano y portables junto al ejecutable
- 🖥️ **Dos vistas** — modo compacto de una línea o vista expandida a pantalla
  completa con pestañas coloreadas (Cola · Buscar · Historial · Favoritos)
- 📦 **Cero fricción** — un único binario; mpv y yt-dlp se auto-descargan

## Requisitos

- Windows 10/11 (usa named pipes y `%LOCALAPPDATA%` para las dependencias)
- Para compilar desde código: [Go](https://go.dev/dl/) 1.26+

## Instalación

### Con `go install`

```
go install github.com/AikonCWD/ytcli@latest
```

### Desde el código

```
git clone https://github.com/AikonCWD/ytcli.git
cd ytcli
go build -o ytcli.exe .
```

En el primer arranque, ytcli descarga mpv y yt-dlp en
`%LOCALAPPDATA%\ytcli\bin` (si ya los tienes en el `PATH`, usa esos).

## Uso

```
ytcli                      # abre la playlist guardada; usa / para buscar
ytcli <url>                # reproduce un vídeo (o un directo)
ytcli <url-playlist>       # encola y reproduce una playlist completa
ytcli <url1> <url2> ...    # encola varias URLs
ytcli lofi girl radio      # busca en YouTube y reproduce el primer resultado
ytcli --help               # ayuda completa: uso, atajos y ficheros
ytcli --version            # versión
```

## Atajos de teclado

Pulsa `?` dentro de la aplicación para ver esta chuleta en cualquier momento.

### Modo compacto

| Tecla | Acción |
|-------|--------|
| `Espacio` | Play / pausa |
| `←` / `→` | Retroceder / avanzar 10 s |
| `↑` / `↓` ó `+` / `-` | Subir / bajar volumen |
| `n` / `p` | Pista siguiente / anterior |
| `m` | Silenciar (mute) |
| `s` | Shuffle on/off |
| `r` | Repeat off → all → one |
| `f` | Favorito de la pista actual |
| `/` | Buscar en YouTube |
| `?` | Ayuda |
| `Tab` | Abrir la vista expandida |
| `q` · `Ctrl+C` | Salir |

### Vista expandida

| Tecla | Acción |
|-------|--------|
| `1` `2` `3` `4` | Pestañas Cola · Buscar · Historial · Favoritos |
| `↑` / `↓` | Mover la selección |
| `Enter` | Reproducir la selección |
| `d` / `Supr` | Quitar de la cola (pestaña Cola) |
| `f` | Favorito de la selección (marcado con ⭐) |
| `Esc` / `Tab` | Volver al modo compacto |

Cada pestaña tiene su propio color (Cola roja, Buscar azul, Historial ámbar,
Favoritos rosa): el borde del panel cambia con la pestaña activa.

## Ficheros de datos

Todo vive en ficheros de texto **junto al ejecutable** — cópialos con el
binario a un USB y te llevas tu música:

| Fichero | Contenido |
|---------|-----------|
| `playlist.txt` | La cola de reproducción (se carga al arrancar, se guarda al cambiarla) |
| `history.txt` | Historial (máx. 200 entradas, la más reciente primero) |
| `favorites.txt` | Favoritos |

El formato es una pista por línea con campos separados por tabulador, pero
basta con **pegar una URL de YouTube por línea**: el título, canal y duración
se rellenan solos al reproducirla.

## Cómo funciona

```
┌─────────┐   Bubble Tea    ┌────────┐  JSON IPC (named pipe)  ┌─────┐
│ Terminal│ ◄────────────►  │ ytcli  │ ◄─────────────────────► │ mpv │ ── audio
└─────────┘                 └────┬───┘                         └─────┘
                                 │ exec
                                 ▼
                            ┌────────┐
                            │ yt-dlp │ ── búsqueda y metadatos
                            └────────┘
```

- La TUI está construida con [Bubble Tea](https://github.com/charmbracelet/bubbletea)
  y [Lip Gloss](https://github.com/charmbracelet/lipgloss), en modo inline
  (sin alt-screen) para un footprint mínimo en el terminal.
- mpv corre como proceso hijo controlado por su
  [IPC JSON](https://mpv.io/manual/stable/#json-ipc) sobre un named pipe.
- yt-dlp se invoca bajo demanda para búsquedas y resolución de URLs/playlists.

## Desarrollo

```
go build ./...    # compilar
go test ./...     # ejecutar los tests
go vet ./...      # análisis estático
```

Estructura del código:

```
main.go              CLI, bootstrap de dependencias y arranque de la TUI
internal/deps/       localización/descarga automática de mpv y yt-dlp
internal/player/     control de mpv vía IPC JSON
internal/queue/      cola de reproducción, shuffle y repeat
internal/store/      persistencia en texto plano (playlist/historial/favoritos)
internal/track/      modelo de pista
internal/tui/        interfaz de terminal (Bubble Tea)
internal/youtube/    búsqueda y resolución de metadatos con yt-dlp
```

## Contribuir

Los issues y pull requests son bienvenidos. Si quieres proponer un cambio
grande, abre antes un issue para comentarlo.

## Licencia

[MIT](LICENSE) © [AikonCWD](https://github.com/AikonCWD)
