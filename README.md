# ytcli

Reproductor de música TUI para YouTube (solo audio). Escrito en Go, usa mpv
(streaming) y yt-dlp (búsqueda y metadatos), que se auto-descargan en el primer
arranque en Windows.

## Uso

    ytcli                      # abre vacío; usa / para buscar
    ytcli <url>                # reproduce un vídeo
    ytcli <url-playlist>       # encola y reproduce una playlist
    ytcli <url1> <url2> ...    # encola varias

## Controles

| Tecla | Acción |
|-------|--------|
| Espacio | Play / Pausa |
| ← / → | Seek ∓10s |
| n / p | Siguiente / anterior |
| ↑ / ↓ (o + / -) | Volumen |
| m | Mute |
| s | Shuffle |
| r | Repeat (off/all/one) |
| / | Buscar |
| Enter | Reproducir selección |
| f | Favorito |
| Tab | Compacto / expandido |
| q | Salir |

## Construir

    go build -o ytcli.exe .
