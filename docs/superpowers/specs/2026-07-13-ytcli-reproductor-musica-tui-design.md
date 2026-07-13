# ytcli — Reproductor de música TUI para YouTube

**Fecha:** 2026-07-13
**Estado:** Diseño aprobado
**Plataforma objetivo:** Windows

## Resumen

`ytcli` es una aplicación de terminal (TUI) que reproduce **solo el audio** de vídeos y
playlists de YouTube. Está pensada para verse bien y ocupar poco: en su estado normal de
reproducción muestra una interfaz **compacta** (pocas líneas) ideal para dejarla corriendo
en un panel spliteado mientras se usa el resto de la ventana para otras tareas. Cuando hace
falta buscar, ver la cola, el historial o los favoritos, se despliega un modo **expandido**
que luego se colapsa de nuevo.

La app no reimplementa audio ni la extracción de YouTube: **delega** en dos herramientas
externas ya existentes y probadas:

- **mpv** — motor de audio, controlado por su interfaz **IPC JSON**. Da streaming, seek,
  control de volumen y cambio de pista sin que tengamos que escribir código de audio.
- **yt-dlp** — usado por mpv por debajo para resolver YouTube, y usado por nosotros
  directamente para la **búsqueda por texto** (`ytsearch`, sin API key), la expansión de
  playlists y la obtención de metadatos.

## Decisiones de alcance (v1)

| Decisión | Elección |
|---|---|
| Plataforma | Solo Windows |
| Obtención de audio | Streaming directo (sin descarga a disco) |
| Entrada de contenido | URLs, URLs de playlist **y** búsqueda por texto |
| Persistencia | Historial + favoritos guardados en disco |
| Gestión de dependencias | Auto-descarga de mpv/yt-dlp en el primer arranque |
| Significado de "favorito" | Pista individual (URL + título + canal) |
| Salto de seek | ±10 segundos por defecto |

## Stack tecnológico

- **Lenguaje:** Go (compila a un único `.exe` sin runtime).
- **TUI:** Bubbletea + Lipgloss (Charm) para el bucle de eventos, el modelo y el estilo.
- **Audio:** proceso `mpv` lanzado por la app y controlado vía **named pipe** de Windows
  (`--input-ipc-server=\\.\pipe\ytcli-mpv`) con su protocolo IPC JSON.
- **YouTube:** binario `yt-dlp` invocado como subproceso.

### Por qué este stack

mpv resuelve streaming, seek, volumen, negociación de formatos y reproducción sin cortes;
construir eso a mano (Opción B: librería de audio nativa) sería mucho más código y más
frágil. Go + Bubbletea da la mejor relación calidad visual / facilidad de distribución
(single-binary) frente a Rust/Ratatui (desarrollo más lento, bindings de mpv más engorrosos)
o Python/Textual (arrastra runtime).

## Interfaz de usuario

### Modo compacto (por defecto, ~4-5 líneas)

Estado normal mientras suena música. Diseñado para un panel estrecho.

```
┌─ ytcli ───────────────────────────────────┐
│ ♪  Daft Punk - Harder Better Faster        │
│ ▶  ▓▓▓▓▓▓▓▓░░░░░░░░░░  1:47 / 3:44          │
│ 🔊 80%   🔁 all   ⧉ 3/12                    │
└────────────────── space ⏯  ←→ seek  tab ⤢ ─┘
```

- Línea 1: título de la pista actual (con desplazamiento si no cabe).
- Línea 2: icono de estado (play/pausa) + barra de progreso + tiempo transcurrido/total.
- Línea 3: volumen, modo de repetición, y posición en la cola (`actual/total`).
- Pie: recordatorio de las teclas más usadas.

### Modo expandido (tecla `Tab`)

Añade debajo del bloque compacto un panel con pestañas navegables:

- **Cola** — lista de pistas encoladas; la actual resaltada.
- **Buscar** — campo de texto (`/`) + lista de resultados de `ytsearch`.
- **Historial** — pistas reproducidas recientemente (desde `store`).
- **Favoritos** — pistas marcadas con `f`.

Selección con flechas, `Enter` para reproducir. `Tab` vuelve a colapsar a compacto.

**Modo comando vs. modo texto:** los atajos de una letra (`n`, `p`, `s`, `f`, `m`...) actúan
como comandos salvo cuando el campo de búsqueda tiene el foco (tras pulsar `/`): ahí las
teclas escriben texto y solo `Enter` (buscar/confirmar) y `Esc` (cancelar) tienen efecto de
control. `Space`, flechas de volumen y seek siguen controlando la reproducción en cualquier
contexto que no sea el de escritura.

### Controles (teclado)

| Tecla | Acción |
|---|---|
| `Space` | Play / Pausa |
| `←` / `→` | Seek −10s / +10s |
| `n` / `p` | Siguiente / anterior pista |
| `↑` / `↓` (o `+` / `-`) | Subir / bajar volumen |
| `m` | Mute |
| `s` | Alternar shuffle |
| `r` | Ciclar repeat (off → all → one) |
| `/` | Buscar en YouTube |
| `Enter` | Reproducir la selección |
| `f` | Marcar/desmarcar favorito (pista actual o seleccionada) |
| `Tab` | Expandir / colapsar interfaz |
| `q` | Salir |

### Uso desde la CLI

- `ytcli <url>` — arranca reproduciendo ese vídeo y lo encola.
- `ytcli <url-playlist>` — expande la playlist a la cola y arranca.
- `ytcli <url1> <url2> ...` — encola varias.
- `ytcli` (sin argumentos) — abre vacío; se usa `/` para buscar.

## Arquitectura

### Paquetes

- **`player`** — envuelve el proceso mpv y su cliente IPC.
  - Responsabilidad: lanzar/terminar mpv, enviar comandos y observar propiedades.
  - Interfaz (aprox.): `Load(url)`, `Toggle()`, `Seek(delta)`, `SetVolume(v)`,
    `Next()`, `Prev()`, y consulta de estado (posición, duración, título, pausa, volumen).
  - Depende de: named pipe de Windows hacia mpv (`deps` le da la ruta del binario).

- **`youtube`** — envuelve yt-dlp.
  - Responsabilidad: buscar (`ytsearch<N>:<query>`), expandir playlists y extraer
    metadatos parseando `--dump-json` / `--flat-playlist`.
  - Interfaz: `Search(query, n) -> []Track`, `Resolve(url) -> []Track`.
  - Depende de: ruta del binario yt-dlp (de `deps`).

- **`deps`** — bootstrap de dependencias.
  - Responsabilidad: localizar mpv y yt-dlp; si faltan, descargarlos a
    `%LOCALAPPDATA%\ytcli\bin`, verificarlos y devolver sus rutas.
  - Muestra una pantalla de progreso en el primer arranque.
  - Interfaz: `Ensure() -> Paths{Mpv, YtDlp}`.

- **`store`** — persistencia.
  - Responsabilidad: leer/escribir historial y favoritos en JSON bajo `%APPDATA%\ytcli\`
    (`history.json`, `favorites.json`).
  - Interfaz: `LoadHistory()/AppendHistory(track)`, `LoadFavorites()/ToggleFavorite(track)`.

- **`queue`** — cola de reproducción en memoria.
  - Responsabilidad: mantener la lista de pistas, el índice actual y los modos shuffle/repeat.
  - Interfaz: `Add(tracks)`, `Current()`, `Next()`, `Prev()`, `SetShuffle(b)`, `SetRepeat(mode)`.
  - Sin dependencias externas (lógica pura → fácil de testear).

- **`tui`** — modelo Bubbletea que orquesta todo.
  - Responsabilidad: estado de UI, manejo de teclas, render de ambos modos, y un **ticker**
    (~4 Hz) que consulta el estado a `player` para refrescar la barra de progreso.

- **`main`** — parseo de argumentos CLI y arranque (`deps.Ensure()` → construir modelo → correr).

### Modelo de datos: `Track`

```
Track {
  ID       string  // id de vídeo de YouTube
  URL      string
  Title    string
  Channel  string
  Duration int     // segundos
}
```

### Flujo de datos

```
URL/búsqueda → youtube.Resolve/Search → []Track → queue.Add
   → player.Load(actual) → mpv hace streaming
   → ticker consulta estado a mpv por IPC → tui renderiza
   → teclas → comandos IPC a mpv
   → al cambiar/terminar pista → store.AppendHistory
```

## Manejo de errores

- **Falta mpv/yt-dlp:** `deps.Ensure()` los auto-descarga. Si la descarga falla, mensaje
  claro (con la URL/instrucción manual como alternativa) y salida con código != 0.
- **Vídeo no disponible / error de red:** se salta a la siguiente pista de la cola y se
  informa en la barra de estado (no bloquea la app).
- **yt-dlp obsoleto (YouTube cambió):** se detecta el patrón de error de yt-dlp y se sugiere
  ejecutar `yt-dlp -U` (o se ofrece re-descargar la última versión vía `deps`).
- **Caída de la conexión IPC:** se reintenta relanzar el proceso mpv y reconectar el pipe;
  si no se logra tras varios intentos, se avisa en la barra de estado.

## Estrategia de pruebas (TDD)

- **`queue`** — unitarias de `Next`/`Prev`/`shuffle`/`repeat` en todos los modos y bordes
  (cola vacía, un solo elemento, wrap-around con repeat all/one).
- **`store`** — round-trip de guardar/cargar JSON; toggle de favorito idempotente.
- **`youtube`** — parseo de metadatos con **fixtures** de salida real de yt-dlp
  (`--dump-json`), sin llamar a la red.
- **`deps`** — lógica de detección (binario presente/ausente, elección de ruta) con un
  sistema de archivos simulado; la descarga real queda como prueba manual.
- **`player`** — pruebas a nivel de **serialización de comandos IPC** (que el JSON enviado a
  mpv sea correcto). La integración real con mpv es prueba manual/etiquetada con build tag.

## Fuera de alcance (v1 — YAGNI)

- Renderizado de vídeo (solo audio, por diseño).
- Caché/descarga de audio para offline (se decidió streaming directo).
- Favoritos como playlists guardadas (v1: favoritos = pistas individuales).
- Soporte multiplataforma explícito (v1: solo Windows; el stack no lo impide en el futuro).
- Ecualizador, letras, integración con otras plataformas.
