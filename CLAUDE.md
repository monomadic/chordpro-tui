# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## What this is

A terminal ChordPro song renderer (Go + Bubbletea/Lipgloss). It lays a whole song out to fill one screen — chords stacked over lyrics, newspaper columns — plus auto-scroll teleprompter and duration-synced player views. A sibling binary, `chordpro-pdf`, exports the same one-page fit layout as a device-sized PDF.

## Commands

```sh
go build -o chordpro-tui ./cmd/chordpro-tui   # build the TUI
go build -o chordpro-pdf ./cmd/chordpro-pdf   # build the PDF exporter
go test ./...                       # all tests
go test ./internal/chordpro/ -run TestParse -v   # single test
go vet ./...                        # lint (no other linter configured)

# Run without a TTY / verify rendering (deterministic, good for eyeballing changes):
go run ./cmd/chordpro-tui --print --width 100 --height 40 testdata/wagon_wheel.cho
CHORDPRO_TUI_FORCE_COLOR=1 go run ./cmd/chordpro-tui --print ... | less -R   # force truecolor when piped

scripts/gallery.sh [--bg] [song.cho]   # render a song in every theme back-to-back

# PDF exporter; eyeball output by rasterizing: sips -s format png out.pdf --out out.png
go run ./cmd/chordpro-pdf --preset ipad-mini testdata/wagon_wheel.cho
```

`--print` mode is the fastest feedback loop: it exercises the full parse → render pipeline with a fixed size and exits, no interactive TUI needed. Sample songs live in `testdata/`.

## Architecture

Data flows one way: **parse → Song model → render → TUI** (or **→ pdf → file** for the exporter). `internal/chordpro` and `internal/chords` are the shared core both binaries reuse; keep them free of rendering concerns.

- `internal/chordpro/` — parser and the `Song` data model (no rendering concerns). `Parse`/`ParseString` produce a `Song`: metadata fields plus `[]Section` → `[]Line` → `[]Segment` (a segment is a chord *or* annotation plus the lyric text it sits over). `transpose.go` has two distinct jobs: `Song.Transposed(n)` for the in-memory view, and `TransposeSource` which rewrites raw file text preserving formatting (used by the `w` save-a-copy feature).
- `internal/render/` — pure functions from `(Song, width, height, Theme)` to styled strings; owns all Lipgloss styling. Two entry points: `RenderWith` (fit mode: tries spacing plans and column counts to fill one screen, centers small songs) and `RenderLongWith` (single tall column returned as lines, scrolled by the TUI). `theme.go` holds the `Palette` structs for all 9 themes — a `Theme` is just `NewTheme(Palette)`, so adding a theme means adding a Palette and registering it in `Themes()`. `chart.go` renders the chord-shape sheet (`c` key).
- `internal/chords/` — static chord fingering database; `{define}` directives in a song override it.
- `internal/pdf/` — one-page PDF export (go-pdf/fpdf, core fonts only, monochrome). All layout math is in "em" units (multiples of the body font size) measured at scale 1 in `layout.go`; `fit` then picks the column count that maximizes the body size for the page, and `pdf.go` draws it. `preset.go` holds device page sizes (device presets = the device's logical point resolution, so aspect ratio matches the screen). Entry point: `Export(song, Options, io.Writer)`.
- `internal/config/` — optional `chordpro-tui.toml` settings: a hand-rolled flat-TOML parser (no dependency), `Default()`, `Marshal` (drives `--print-config`), and `Load` (searches `./` then the user config dir, or an explicit `--config` path). Display options resolve to `render.Tri` values in a `render.RenderOpts`; the fit renderer resolves `auto` by re-rendering leaner until the song fits (`internal/render/display.go`). Keep the zero value == "everything off" so a missing config changes nothing.
- `internal/tui/` — the Bubbletea `Model` (view modes fit/scroll/sync, key handling, theme cycling, `$EDITOR` round-trip) and the fuzzy file picker (`o` key, folder browsing via `n`/`p`/`r`, ordered by the `sort-songs` config). Holds both `base` (untransposed) and `song` (transposed) so transpose is always re-derived from source.
- `cmd/chordpro-tui/` — TUI flags (`--print-config`, `--config`, …), TTY detection (non-TTY or `--print` renders once to stdout), stdin/file/directory input resolution.
- `cmd/chordpro-pdf/` — exporter flags (preset/orientation/custom page size), output-path derivation.

Parser/model changes usually ripple: a new directive touches `parser.go`, possibly `model.go`, then rendering in `render.go`/`header.go`, and the README's "Supported ChordPro" section.

## Reference

- `doc/chordpro-spec.md` — the ChordPro spec; check it before changing directive parsing.
- README documents exact key bindings, supported directives, and transpose spelling rules (fixed per note: Eb/Bb flats, C#/F#/G# sharps) — keep it in sync with behavior changes.
