# chordpro-tui

A colorful, modern terminal renderer for [ChordPro](https://www.chordpro.org/)
song files. It lays a whole song out to fill **one screen** — chords stacked
over lyrics, a framed title card, metadata pills — flowing into balanced
newspaper columns so nothing scrolls off the page when it doesn't have to.

It also has a **teleprompter mode** that auto-scrolls at the song's tempo.

![two-column fit layout](#) <!-- run it to see -->

## Why Go + Charm

The renderer is built on [Lipgloss](https://github.com/charmbracelet/lipgloss)
for styling and layout and [Bubbletea](https://github.com/charmbracelet/bubbletea)
for the interactive loop. That combination is the lowest-friction path to a
genuinely good-looking TUI: truecolor styles, rounded borders, and column
composition come for free, and the same render code powers both the static
"fit to page" view and the animated scroll view.

## Install / build

```sh
go build -o chordpro-tui .
```

Requires Go 1.21+ and a truecolor terminal for the full palette.

## Usage

```sh
# Interactive (default when stdout is a terminal)
chordpro-tui testdata/wagon_wheel.cho

# Start straight into auto-scroll teleprompter mode
chordpro-tui -scroll testdata/wagon_wheel.cho

# Transpose up 2 semitones, pick a theme
chordpro-tui -transpose 2 -theme "Tokyo Night" testdata/wagon_wheel.cho

# Render once and exit (good for piping / screenshots)
chordpro-tui -print testdata/wagon_wheel.cho
chordpro-tui -print -width 120 -height 40 testdata/wagon_wheel.cho

# Read from stdin
chordpro-tui < testdata/wagon_wheel.cho
```

### Keys (interactive)

| Key              | Action                                            |
| ---------------- | ------------------------------------------------- |
| `s`              | cycle view mode: **fit → scroll → sync**          |
| `t`              | cycle color theme                                 |
| `[` / `]`        | transpose down / up (fit mode)                    |
| `0`              | reset transpose                                   |
| `space`          | pause/resume scroll · play/pause sync             |
| `r`              | restart the sync timeline                         |
| `+` / `-`        | scroll speed (scroll) · song length (sync)        |
| `↑`/`↓`, `j`/`k` | scroll a line / seek the timeline                 |
| `f`/`b`, PgDn/PgUp | scroll a page                                   |
| `g` / `G`        | jump to top / bottom                              |
| `q`              | quit                                              |

### View modes

- **Fit** — the whole song laid out to fill one screen (see below).
- **Scroll** — a teleprompter that auto-scrolls at a constant, tempo-derived
  speed you can nudge with `+`/`-`.
- **Sync** — scrolls so the last line lands exactly at the end of the song.
  Reads a `{duration: mm:ss}` directive (defaults to 3:30, adjustable with
  `+`/`-`); `space` plays/pauses and a progress bar shows elapsed / total.

### Transpose & themes

`[` / `]` shift every chord (and the key) by a semitone, choosing the
conventional spelling for the resulting key (e.g. transposing into B♭ spells
flats, into A spells sharps); slash-chord bass notes move too. `t` cycles the
bundled themes — **Mocha, Tokyo Night, Gruvbox, Dracula, Nord** — and the
footer shows the active theme and transpose offset.

## Layout behaviour

- **Fits when it can.** The song is split into atomic section blocks (verses,
  choruses, …) that flow top-to-bottom into as many columns as needed to stay
  within the screen height — but never more columns than fit the width, so it
  never overflows sideways.
- **Centers when it's small.** A short song is centered vertically and
  horizontally on the page.
- **Scrolls when it can't.** A song too big for any single-screen layout falls
  back gracefully; press `s` for the auto-scrolling teleprompter.

## Supported ChordPro

Directives: `title`/`t`, `subtitle`/`st`, `artist`, `composer`, `album`, `key`,
`capo`, `tempo`, `time`, `year`, `duration`/`length`, `comment`/`c`, and the
`start_of_*`/`end_of_*` (and `soc`/`eoc`/`sov`/`sob`/`sot`) environments for
choruses, verses, bridges, and tab blocks. Inline `[chord]` markup is positioned
over the syllable that follows it. Unknown directives are ignored; `#` lines are
source comments.

## Project layout

```
main.go                      CLI entry point, TTY detection, flags
internal/chordpro/           parser + song model + transpose
internal/render/             themes, chord/lyric alignment, column packing
internal/tui/                Bubbletea model (fit / scroll / sync modes)
testdata/                    example songs
```

## Theming

The palette lives in `internal/render/theme.go` (default: Catppuccin Mocha).
Swap the `Palette` values to reskin every style at once.
```
