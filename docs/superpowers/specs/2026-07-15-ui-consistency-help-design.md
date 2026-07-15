# ytcli — consistencia visual, colores por pestaña y --help

Fecha: 2026-07-15 · Estado: aprobado implícitamente (petición directa del usuario, sesión autónoma)

## Problema

- Mezcla sin criterio de emojis (🔊 🔁 🔀 🔴 🔎) y glifos sueltos (♪ ▶ ⏸ ⧉ ⏯ ⤢ ›).
- Un único color (rojo 167) para todo; las pestañas del modo expandido no se
  distinguen al pulsar TAB.
- Barra de progreso ▓░ monocroma.
- No hay `--help` ni `--version`; `ytcli --loquesea` se busca en YouTube.

## Diseño

### 1. Sistema visual unificado

Principio: **emoji = icono de estado/categoría** (aporta color propio y
semántica), **glifo fino coloreado = elemento estructural** (barra, marcadores,
cursor). Nada de símbolos intermedios raros (⧉ ⏯ ⤢ desaparecen).

| Elemento | Antes | Ahora |
|---|---|---|
| Título en reproducción | `♪ ` | `🎵 ` |
| Estado transporte | `▶` / `⏸` | igual (glifo estructural, junto a la barra) |
| Barra de progreso | `▓▓░░` | `━━●──` — tramo recorrido en acento, resto atenuado, knob `●` |
| Volumen | `🔊 80%` | `🔊 80%` · `🔇 0%` si mute |
| Repeat | `🔁 all/one/off` | `🔁 all` / `🔂 one` / `🔁 off` (atenuado si off) |
| Shuffle | `🔀 on/off` | igual, atenuado si off |
| Posición en cola | `⧉ 3/12` | `🎶 3/12` |
| Directo | `🔴 EN DIRECTO` | igual |
| Búsqueda | `🔎 ` | igual |
| Ayudas de teclas | `space ⏯ … tab ⤢` | texto plano atenuado con separador `·` |
| Marcador pista actual (lista) | `♪ ` | `♪ ` en color acento |
| Cursor de selección | `› ` + Reverse | igual (Reverse es seguro en cualquier tema) |

Los valores on/activos se pintan con el color de acento; off/inactivos en faint.

### 2. Colores por pestaña (frames al pulsar TAB)

Paleta ANSI-256 (degrada bien, coherente con Windows Terminal):

- Cola → rojo 167 (el acento actual)
- Buscar → azul 75
- Historial → ámbar 179
- Favoritos → rosa 213

El borde del frame expandido y la etiqueta de la pestaña activa (bold +
subrayado) usan el color de su pestaña; las inactivas van en faint. El frame
compacto conserva el rojo 167.

### 3. CLI: --help / --version

- `-h`, `--help`, `help` → ayuda completa por stdout y exit 0: uso, ejemplos,
  opciones, tabla de atajos (compacto y expandido), ficheros de datos,
  dependencias auto-descargadas.
- `-v`, `--version` → `ytcli <versión>`.
- Cualquier otro argumento que empiece por `-` → error a stderr sugiriendo
  `--help` (antes se enviaba a la búsqueda de YouTube).
- El resto del contrato CLI no cambia (URLs / palabras de búsqueda).

### 4. Fuera de alcance

Overlay de ayuda dentro de la TUI, temas configurables, reconexión IPC.

## Testing

- `progressBar` testeado sobre `ansi.Strip` (el color vive en la función).
- Tests de vista existentes se mantienen (♪ marcador, EN DIRECTO, anchos);
  se añaden casos para 🔇 mute, 🔂 repeat-one y ayuda de --help en main.
- `go test ./...`, `go vet`, build.
