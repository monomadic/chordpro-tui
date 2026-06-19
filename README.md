# chordpro-tui

A colorful, modern terminal renderer for [ChordPro](https://www.chordpro.org/)
song files. It lays a whole song out to fill **one screen** — chords stacked
over lyrics, a centered title, metadata pills — flowing into balanced
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

# Point at a folder to open its most recently modified song
chordpro-tui testdata/

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
| `?`              | show all keyboard shortcuts                       |
| `o`              | open another song from this folder (fuzzy finder) |
| `e`              | edit the current file in `$EDITOR`                |
| `n` / `p`        | load next / previous song in the folder           |
| `r`              | load a random song in the folder                  |
| `s`              | cycle view mode: **fit → scroll → sync**          |
| `c`              | chord-shape sheet for the current song            |
| `t`              | cycle color theme                                 |
| `B`              | toggle themed background fill                      |
| `h`              | toggle the title header                            |
| `[` / `]`        | transpose down / up (fit mode)                    |
| `0`              | reset transpose                                   |
| `space`          | pause/resume scroll · play/pause sync             |
| `+` / `-`        | scroll speed (scroll) · song length (sync)        |
| `↑`/`↓`, `j`/`k` | scroll a line / seek the timeline                 |
| `f`/`b`, PgDn/PgUp | scroll a page                                   |
| `g` / `G`        | jump to top / bottom (`g` restarts sync)          |
| `q`              | quit                                              |

Press `?` any time for an on-screen overlay of all of these.

### View modes

- **Fit** — the whole song laid out to fill one screen (see below).
- **Scroll** — a teleprompter that auto-scrolls at a constant, tempo-derived
  speed you can nudge with `+`/`-`.
- **Sync** — scrolls so the last line lands exactly at the end of the song.
  Reads a `{duration: mm:ss}` directive (defaults to 3:30, adjustable with
  `+`/`-`); `space` plays/pauses and a progress bar shows elapsed / total.

### Transpose & themes

`[` / `]` shift every chord (and the key) by a semitone; slash-chord bass notes
move too. Accidentals use the spelling most commonly seen on lead sheets, fixed
per note rather than by key: **E♭** and **B♭** are flats, while **C♯**, **F♯**
and **G♯** are sharps (so you get B♭ over A♯ but F♯ over G♭, and E/G♯ stays
sharp). The current key pill shows the transpose offset (e.g. `Bb +3`). `t`
cycles the bundled themes — **Mocha, Tokyo Night, Gruvbox, Dracula, Nord** plus
the neon set **Synthwave, Cyberpunk, Laser, Vapor** — and the footer shows the
active theme (with its position in the cycle, e.g. `Synthwave 6/9`).

`B` toggles a **themed background fill** (also `-bg`): instead of the terminal's
default background, the whole screen is painted with the theme's background
color, with chord and metadata pills still standing out on top. Best with a
truecolor terminal.

To preview every theme at once, run `scripts/gallery.sh` (add `--bg` to see the
backgrounds, pass a song path to use your own): it renders the song in each
theme back-to-back with colors forced on.

### Opening, browsing & editing songs

Pass a **folder** instead of a file and the most recently modified song in it
opens (handy for "show me the chart I just saved"); the rest of the folder is
then a keypress away.

`o` opens a fuzzy finder over every ChordPro file
(`.cho .chopro .chordpro .crd .pro .cp`) in the current song's folder. Rows are
laid out in labelled columns — **title, artist, key, capo, tempo, year** (the
metadata columns drop off, narrowest first, on small terminals) and file
extensions are hidden. Type to filter by title (matched characters are
highlighted), `↑`/`↓` to move, `enter` to open, `esc` to cancel.

Without opening the finder you can also jump straight between songs in the
folder: `n` / `p` for next / previous (alphabetical, wrapping) and `r` for a
random pick. Switching songs resets transpose and playback but keeps your theme.

`e` opens the current file in `$EDITOR` (falling back to `vi`); when you quit the
editor the song is reloaded automatically, preserving your transpose and theme.

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
`capo`, `tempo`, `bpm`, `time`/`time_signature`, `year`, `tuning`,
`duration`/`length`, `comment`/`c`/`comment_italic`/`comment_box`/`highlight`,
`define`, and the `start_of_*`/`end_of_*` (and `soc`/`sov`/`sob`/`sot`/`soi`/`soo`)
environments for choruses, verses, bridges, tab blocks, intros, and outros.
Inline `[chord]` markup is positioned over the syllable that follows it.

- A directive's argument may be a bare value (`{start_of_verse: Verse 1}`) or
  HTML-style attributes (`{start_of_verse: label="Verse 1"}`, single or double
  quotes, with or without the colon). Section `label=` attributes set the
  section heading.
- `[*…]` brackets are **annotations** (`[*Riff x2]`, `[*N.C.]`): the `*` is
  dropped and the text is shown in the chord position verbatim — never
  transposed or drawn as a chord shape.
- `bpm` drives the scroll/sync speed; when both are present `bpm` wins and the
  TEMPO pill shows it, so a word `tempo` like `Allegro` doesn't break pacing.
- `define` chord fingerings override the built-in shapes in the chord sheet
  (`c`); `tuning` is shown there too.
- Blank lines **inside** a `start_of_*`/`end_of_*` block are kept as spacing;
  loose (un-bracketed) paragraphs still split on blank lines.

Unknown directives are ignored; `#` lines are source comments. Conditional
selectors (`{comment-guitar: …}`), `{chorus}` recall, `{transpose}`, grids, and
non-European note systems are not yet interpreted.

## Project layout

```
main.go                      CLI entry point, TTY detection, flags
internal/chordpro/           parser + song model + transpose
internal/render/             themes, chord/lyric alignment, column packing
internal/tui/                Bubbletea model (fit / scroll / sync) + file picker
scripts/gallery.sh           render the sample song in every theme
testdata/                    example songs
```

## Theming

The palette lives in `internal/render/theme.go` (default: Catppuccin Mocha).
Swap the `Palette` values to reskin every style at once.
```
