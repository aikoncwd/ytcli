# ytcli

Reproductor de música TUI para YouTube (solo audio). Escrito en Go, usa mpv
(streaming) y yt-dlp (búsqueda y metadatos), que se auto-descargan en el primer
arranque en Windows.

## Uso

    ytcli                      # abre la playlist guardada; usa / para buscar
    ytcli <url>                # reproduce un vídeo (o un directo)
    ytcli <url-playlist>       # encola y reproduce una playlist
    ytcli <url1> <url2> ...    # encola varias
    ytcli lofi girl radio      # busca en YouTube y reproduce el primer resultado
    ytcli --help               # ayuda completa: uso, atajos y ficheros
    ytcli --version            # versión

## Ficheros de datos

Todo vive en ficheros de texto **junto al ejecutable**, editables a mano
(una pista por línea, campos separados por tabulador; basta con pegar una URL):

    playlist.txt     la cola de reproducción (se carga al arrancar)
    history.txt      historial (máx. 200, más reciente primero)
    favorites.txt    favoritos

## Controles

| Tecla | Acción |
|-------|--------|
| Espacio | Play / Pausa |
| ← / → | Seek ∓10s |
| n / p | Siguiente / anterior |
| + / - | Volumen |
| ↑ / ↓ | Volumen (modo compacto) · mover selección (modo expandido) |
| 1 / 2 / 3 / 4 | Ir a pestaña Cola / Buscar / Historial / Favoritos |
| m | Mute |
| s | Shuffle |
| r | Repeat (off/all/one) |
| / | Buscar |
| Enter | Reproducir selección |
| f | Favorito (marcado con ⭐ en las listas) |
| ? | Ayuda (atajos) dentro de la TUI |
| d / Supr | Quitar de la cola (pestaña Cola) |
| Tab | Compacto / expandido (pantalla completa) |
| Esc | Volver al modo compacto |
| q | Salir |

En el modo expandido cada pestaña tiene su color (Cola roja, Buscar azul,
Historial ámbar, Favoritos rosa): el borde del panel cambia con la pestaña
activa.

## Construir

    go build -o ytcli.exe .
