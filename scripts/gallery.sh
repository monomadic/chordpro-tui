#!/usr/bin/env bash
#
# gallery.sh — render a ChordPro song in every bundled theme, one after another,
# for a quick visual comparison. Colors are forced on so it looks right even
# when piped (e.g. `scripts/gallery.sh | less -R`).
#
# Usage:
#   scripts/gallery.sh [--bg] [song.cho]
#
#   --bg        also fill each card with the theme's background color
#   song.cho    song to render (default: testdata/wagon_wheel.cho)
#
# Env overrides:
#   GALLERY_W   render width  (default: terminal width, else 100)
#   GALLERY_H   render height (default: 22)
#
# Note: avoids `set -u` for compatibility with macOS's stock bash 3.2.
set -eo pipefail

cd "$(dirname "$0")/.."

bg=()
if [ "${1:-}" = "--bg" ]; then
  bg=(--bg)
  shift
fi
song="${1:-testdata/wagon_wheel.cho}"

if [ ! -f "$song" ]; then
  echo "gallery: no such song: $song" >&2
  exit 1
fi

bin="$(mktemp -t cptui-gallery)"
trap 'rm -f "$bin"' EXIT
go build -o "$bin" ./cmd/chordpro-tui

cols="${GALLERY_W:-$(tput cols 2>/dev/null || echo 100)}"
rows="${GALLERY_H:-22}"

themes=(Mocha "Tokyo Night" Gruvbox Dracula Nord Synthwave Cyberpunk Laser Vapor)

for t in "${themes[@]}"; do
  printf '\n\033[1;38;2;255;106;213m▌ %s\033[0m\n\n' "$t"
  CHORDPRO_TUI_FORCE_COLOR=1 "$bin" --print --theme "$t" "${bg[@]}" \
    --width "$cols" --height "$rows" "$song"
done
