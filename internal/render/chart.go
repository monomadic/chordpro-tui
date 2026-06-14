package render

import (
	"strconv"
	"strings"

	"chordpro-tui/internal/chords"
	"chordpro-tui/internal/chordpro"

	"github.com/charmbracelet/lipgloss"
)

// fretRows is how many fret spaces each diagram shows below the nut. The
// dataset is built around four-fret windows, so four keeps every shape visible.
const fretRows = 4

// RenderChordSheet lays out a fingering diagram for every distinct chord used
// in the song, packed into as many per-row as the width allows, with a title
// and a footer hint. Chords appear in order of first use, deduplicated, so the
// sheet mirrors how the song unfolds. The song is expected to already be
// transposed, so the diagrams match what's on screen.
func RenderChordSheet(song *chordpro.Song, width, height int, th *Theme) string {
	if th == nil {
		th = DefaultTheme()
	}
	if width < 20 {
		width = 20
	}
	if height < 6 {
		height = 6
	}

	title := th.Title.Render(song.Title)
	sub := th.Subtitle.Render("Chord shapes" + keySuffix(song))
	header := lipgloss.JoinVertical(lipgloss.Center, title, sub)

	names := uniqueChords(song)
	var diagrams []string
	var missing []string
	for _, name := range names {
		if shape, ok := chords.Lookup(name); ok {
			diagrams = append(diagrams, chordDiagram(shape, th))
		} else {
			missing = append(missing, name)
		}
	}

	headerH := lipgloss.Height(header)
	availH := height - headerH - 2 // blank line below header + footer
	if availH < 1 {
		availH = 1
	}

	body := gridView(diagrams, width)
	if note := missingNote(missing, th); note != "" {
		body = lipgloss.JoinVertical(lipgloss.Left, body, "", note)
	}

	// Clip to the available height so the footer stays pinned, or center
	// vertically when there's room to spare.
	lines := strings.Split(body, "\n")
	truncated := false
	if len(lines) > availH {
		lines = lines[:availH]
		truncated = true
	} else if pad := availH - len(lines); pad > 0 {
		top := pad / 2
		lines = append(append(make([]string, top), lines...), make([]string, pad-top)...)
	}
	body = strings.Join(lines, "\n")

	header = lipgloss.PlaceHorizontal(width, lipgloss.Center, header)
	body = lipgloss.PlaceHorizontal(width, lipgloss.Center, body)
	footer := lipgloss.PlaceHorizontal(width, lipgloss.Center, chartFooter(th, truncated))

	return header + "\n\n" + body + "\n" + footer
}

// keySuffix renders " · key of G" when the song states a key, else "".
func keySuffix(song *chordpro.Song) string {
	if song.Key == "" {
		return ""
	}
	return " · key of " + song.Key
}

// uniqueChords collects every chord name in the song in first-use order,
// skipping blanks and duplicates.
func uniqueChords(song *chordpro.Song) []string {
	seen := make(map[string]bool)
	var out []string
	for _, sec := range song.Sections {
		for _, ln := range sec.Lines {
			for _, seg := range ln.Segments {
				if seg.Chord == "" || seen[seg.Chord] {
					continue
				}
				seen[seg.Chord] = true
				out = append(out, seg.Chord)
			}
		}
	}
	return out
}

// chordDiagram renders one fingering as a fixed-height block: a name, an
// open/muted marker row, and a fretboard grid. Every block is the same height
// so they tile cleanly into a grid.
func chordDiagram(s chords.Shape, th *Theme) string {
	dot := th.Chord.Foreground(th.P.Chord).Background(lipgloss.NoColor{}).Bold(true)
	grid := th.Muted
	open := th.Section
	muted := th.Muted

	var b strings.Builder

	// Name, centered over the 11-cell-wide grid.
	b.WriteString(center(th.Chord.Render(" "+s.Name+" "), 11))
	b.WriteByte('\n')

	// Marker row: × muted, ○ open, blank when the string is fretted.
	for i := 0; i < 6; i++ {
		if i > 0 {
			b.WriteByte(' ')
		}
		switch {
		case fretAt(s, i) < 0:
			b.WriteString(muted.Render("×"))
		case fretAt(s, i) == 0:
			b.WriteString(open.Render("○"))
		default:
			b.WriteByte(' ')
		}
	}
	b.WriteByte('\n')

	// Top border. A diagram sitting at the nut (BaseFret 1) gets a heavy bar.
	if s.BaseFret <= 1 {
		b.WriteString(grid.Render("┍━┯━┯━┯━┯━┑"))
	} else {
		b.WriteString(grid.Render("┌─┬─┬─┬─┬─┐"))
	}
	b.WriteByte('\n')

	for row := 1; row <= fretRows; row++ {
		// String cells: a dot where this string is fretted at this row.
		var line strings.Builder
		for i := 0; i < 6; i++ {
			if i > 0 {
				line.WriteString(grid.Render(" "))
			}
			if fretRowFor(s, i) == row {
				line.WriteString(dot.Render("●"))
			} else {
				line.WriteString(grid.Render("│"))
			}
		}
		// Label the starting fret beside the first row of a moved diagram.
		if row == 1 && s.BaseFret > 1 {
			line.WriteString(th.Muted.Render("  " + strconv.Itoa(s.BaseFret) + "fr"))
		}
		b.WriteString(line.String())
		b.WriteByte('\n')

		if row < fretRows {
			b.WriteString(grid.Render("├─┼─┼─┼─┼─┤"))
			b.WriteByte('\n')
		} else {
			b.WriteString(grid.Render("└─┴─┴─┴─┴─┘"))
		}
	}

	return b.String()
}

// fretAt is the raw fret value for string i, or -1 when out of range.
func fretAt(s chords.Shape, i int) int {
	if i < 0 || i >= len(s.Frets) {
		return -1
	}
	return s.Frets[i]
}

// fretRowFor returns which displayed row (1..fretRows) string i is fretted on,
// or 0 when the string is open, muted, or out of the window. Positive fret
// values are already row numbers relative to BaseFret in the dataset.
func fretRowFor(s chords.Shape, i int) int {
	v := fretAt(s, i)
	if v <= 0 {
		return 0
	}
	if v > fretRows {
		return 0
	}
	return v
}

// gridView packs equal-height diagram blocks left-to-right, wrapping to a new
// row of blocks whenever the next one would overflow the width.
func gridView(blocks []string, width int) string {
	if len(blocks) == 0 {
		return ""
	}
	const gap = "   "
	blockW := lipgloss.Width(blocks[0])
	perRow := (width + lipgloss.Width(gap)) / (blockW + lipgloss.Width(gap))
	if perRow < 1 {
		perRow = 1
	}

	var rows []string
	for i := 0; i < len(blocks); i += perRow {
		end := i + perRow
		if end > len(blocks) {
			end = len(blocks)
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, withGutters(blocks[i:end])...))
	}
	// A blank line between rows of diagrams keeps them from crowding.
	return strings.Join(rows, "\n\n")
}

// missingNote summarizes chords with no catalogued shape, so the user knows
// they were intentionally skipped rather than lost.
func missingNote(missing []string, th *Theme) string {
	if len(missing) == 0 {
		return ""
	}
	return th.Muted.Render("no shape for: " + strings.Join(missing, " "))
}

func chartFooter(th *Theme, truncated bool) string {
	hint := "c/esc back · [ ] transpose · 0 reset · t theme · q quit"
	if truncated {
		hint = "↓ more chords than fit · " + hint
	}
	return th.Muted.Render(hint)
}

// center pads s to width w on both sides, accounting for ANSI styling.
func center(s string, w int) string {
	return lipgloss.PlaceHorizontal(w, lipgloss.Center, s)
}
