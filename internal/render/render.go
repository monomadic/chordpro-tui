package render

import (
	"strings"
	"unicode/utf8"

	"chordpro-tui/internal/chordpro"

	"github.com/charmbracelet/lipgloss"
)

const (
	gutter      = "   " // horizontal space between columns
	chorusBar = "▍ " // left accent for chorus lines
	colPad    = 1    // extra left padding inside each column
)

// block is a self-contained, already-styled rectangle of text (one section).
type block struct {
	lines  []string
	width  int
	height int
}

func newBlock(lines []string) block {
	w := 0
	for _, l := range lines {
		if x := lipgloss.Width(l); x > w {
			w = x
		}
	}
	return block{lines: lines, width: w, height: len(lines)}
}

// Render lays the whole song out to fit within width x height and returns the
// composed screen string. When the song is shorter than the screen it is
// centered; when taller it flows into balanced newspaper columns.
func Render(song *chordpro.Song, width, height int, th *Theme) string {
	if th == nil {
		th = DefaultTheme()
	}
	if width < 20 {
		width = 20
	}
	if height < 6 {
		height = 6
	}

	header := buildHeader(song, width, th)

	headerH := lipgloss.Height(header)
	// Reserve a blank line below the header and one line for the footer.
	availH := height - headerH - 2
	if availH < 1 {
		availH = 1
	}

	blocks := buildBlocks(song, th)
	body := packColumns(blocks, width, availH)

	// If the song can't fit the page, clip it so it never spills past the
	// screen (which would bury the footer), and flag that there's more.
	bodyLines := strings.Split(body, "\n")
	truncated := false
	if len(bodyLines) > availH {
		bodyLines = bodyLines[:availH]
		truncated = true
	} else if pad := availH - len(bodyLines); pad > 0 {
		// Push the body down so its last line sits just above the footer.
		bodyLines = append(make([]string, pad), bodyLines...)
	}
	body = strings.Join(bodyLines, "\n")

	footer := buildFooter(song, width, th, truncated)

	body = lipgloss.PlaceHorizontal(width, lipgloss.Center, body)
	header = lipgloss.PlaceHorizontal(width, lipgloss.Center, header)
	footer = lipgloss.PlaceHorizontal(width, lipgloss.Center, footer)

	return header + "\n\n" + body + "\n" + footer
}

// buildBlocks turns each song section into a styled block.
func buildBlocks(song *chordpro.Song, th *Theme) []block {
	var blocks []block
	for _, sec := range song.Sections {
		var lines []string
		if sec.Label != "" {
			lines = append(lines, th.Section.Render(strings.ToUpper(sec.Label)))
		}
		for _, ln := range sec.Lines {
			lines = append(lines, renderLine(ln, sec.Kind, th)...)
		}
		if sec.Kind == chordpro.KindChorus {
			lines = decorateChorus(lines, th)
		}
		if len(lines) > 0 {
			lines = append(lines, "") // breathing room after each section
			blocks = append(blocks, newBlock(lines))
		}
	}
	return blocks
}

// renderLine produces the styled rows for a single source line: a chord row
// stacked above the lyric row, a comment, or a verbatim tab row.
func renderLine(ln chordpro.Line, kind chordpro.SectionKind, th *Theme) []string {
	if ln.Comment != "" {
		return []string{th.Comment.Render("✦ " + ln.Comment)}
	}
	if kind == chordpro.KindTab {
		return []string{th.Tab.Render(ln.PlainText())}
	}
	if ln.IsBlank() {
		return []string{""}
	}

	_, lyricRow := alignChords(ln.Segments)
	lyricRow = strings.TrimRight(lyricRow, " ")

	var out []string
	if ln.HasChords() {
		out = append(out, styleChordRow(ln.Segments, th))
	}
	out = append(out, th.Lyric.Render(lyricRow))
	return out
}

// styleChordRow builds the chord row with each chord individually styled, so
// the background hugs each chord as a pill. It reproduces alignChords' spacing
// exactly, so chords stay aligned with the lyric row below.
func styleChordRow(segs []chordpro.Segment, th *Theme) string {
	var b strings.Builder
	chordVis, lyricVis := 0, 0
	for _, seg := range segs {
		if seg.Chord != "" {
			if gap := lyricVis - chordVis; gap > 0 {
				b.WriteString(strings.Repeat(" ", gap))
				chordVis += gap
			}
			b.WriteString(th.Chord.Render(seg.Chord))
			chordVis += runeLen(seg.Chord)
			b.WriteByte(' ') // unstyled separator between pills
			chordVis++
		}
		lyricVis += runeLen(seg.Text)
		if gap := chordVis - lyricVis; gap > 0 {
			lyricVis += gap // mirror lyric overhang padding
		}
	}
	return strings.TrimRight(b.String(), " ")
}

// alignChords builds the plain (unstyled) chord and lyric rows for a line,
// positioning each chord directly above the syllable it applies to.
func alignChords(segs []chordpro.Segment) (chordRow, lyricRow string) {
	var chord, lyric strings.Builder
	for _, seg := range segs {
		if seg.Chord != "" {
			// Pad the chord row out to the current lyric position.
			if gap := runeLen(lyric.String()) - runeLen(chord.String()); gap > 0 {
				chord.WriteString(strings.Repeat(" ", gap))
			}
			chord.WriteString(seg.Chord)
			chord.WriteByte(' ') // keep adjacent chords from touching
		}
		lyric.WriteString(seg.Text)
		// If the chord overhangs its syllable, push following lyrics right so
		// the next chord still lands in the right place.
		if gap := runeLen(chord.String()) - runeLen(lyric.String()); gap > 0 {
			lyric.WriteString(strings.Repeat(" ", gap))
		}
	}
	return chord.String(), lyric.String()
}

// decorateChorus prefixes a colored accent bar to every line of a chorus block.
func decorateChorus(lines []string, th *Theme) []string {
	bar := th.ChorusBar.Render(chorusBar)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = bar + l
	}
	return out
}

// packColumns flows blocks into balanced newspaper columns and joins them side
// by side. It searches for the fewest columns that keep every column within
// maxH, while never letting the real layout exceed width — so the result never
// overflows horizontally. Each candidate is measured for its actual width
// (columns size to their own content), so legitimate multi-column layouts are
// not rejected by one long line elsewhere in the song.
func packColumns(blocks []block, width, maxH int) string {
	if len(blocks) == 0 {
		return ""
	}

	var chosen [][]block
	for n := 1; n <= len(blocks); n++ {
		cols := packInto(blocks, n)
		if layoutWidth(cols) > width {
			if chosen == nil {
				chosen = cols // even a single column is too wide; render anyway
			}
			break // adding columns only gets wider
		}
		chosen = cols
		if maxColHeight(cols) <= maxH {
			break // fits the page height with the fewest columns that fit width
		}
		if len(cols) < n {
			break // blocks can't be split into more columns; this is as tall as it gets
		}
	}

	rendered := make([]string, len(chosen))
	for i, c := range chosen {
		rendered[i] = renderColumn(c)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, withGutters(rendered)...)
}

// layoutWidth is the rendered width of a set of columns: each column sized to
// its widest block plus padding, joined by gutters.
func layoutWidth(cols [][]block) int {
	if len(cols) == 0 {
		return 0
	}
	w := 0
	for _, c := range cols {
		w += colWidth(c) + colPad
	}
	w += lipgloss.Width(gutter) * (len(cols) - 1)
	return w
}

// maxColHeight is the height of the tallest column.
func maxColHeight(cols [][]block) int {
	h := 0
	for _, c := range cols {
		if ch := totalHeight(c); ch > h {
			h = ch
		}
	}
	return h
}

func colWidth(blocks []block) int { return maxWidth(blocks) }

// packInto distributes blocks into at most n balanced columns, keeping blocks
// whole. It targets an even height per column, bumping the target until the
// blocks actually fit in n columns.
func packInto(blocks []block, n int) [][]block {
	if n <= 1 {
		return [][]block{blocks}
	}
	total := totalHeight(blocks)
	target := ceilDiv(total, n)
	for {
		cols := greedyFill(blocks, target)
		if len(cols) <= n {
			return cols
		}
		target++ // too many columns at this target; allow taller columns
	}
}

// greedyFill packs blocks top-to-bottom, starting a new column whenever adding
// the next block (plus its separator) would exceed target.
func greedyFill(blocks []block, target int) [][]block {
	var cols [][]block
	var col []block
	colH := 0
	for _, b := range blocks {
		add := b.height
		if len(col) > 0 {
			add++ // blank separator
		}
		if colH+add > target && len(col) > 0 {
			cols = append(cols, col)
			col = nil
			colH = 0
			add = b.height
		}
		col = append(col, b)
		colH += add
	}
	if len(col) > 0 {
		cols = append(cols, col)
	}
	return cols
}

func totalHeight(blocks []block) int {
	t := 0
	for i, b := range blocks {
		if i > 0 {
			t++ // separator
		}
		t += b.height
	}
	return t
}

func maxWidth(blocks []block) int {
	w := 1
	for _, b := range blocks {
		if b.width > w {
			w = b.width
		}
	}
	return w
}

func ceilDiv(a, b int) int {
	if b <= 0 {
		return a
	}
	return (a + b - 1) / b
}

// renderColumn stacks a column's blocks with blank separators, left-aligned to
// the column's widest block.
func renderColumn(blocks []block) string {
	w := 0
	for _, b := range blocks {
		if b.width > w {
			w = b.width
		}
	}
	var lines []string
	for i, b := range blocks {
		if i > 0 {
			lines = append(lines, "")
		}
		lines = append(lines, b.lines...)
	}
	// lipgloss includes padding within Width, so widen the box by colPad to
	// keep the content area exactly w wide (otherwise long lines wrap).
	col := lipgloss.NewStyle().Width(w + colPad).PaddingLeft(colPad)
	return col.Render(strings.Join(lines, "\n"))
}

// withGutters interleaves gutter spacers between rendered columns.
func withGutters(cols []string) []string {
	if len(cols) <= 1 {
		return cols
	}
	out := make([]string, 0, len(cols)*2-1)
	for i, c := range cols {
		if i > 0 {
			out = append(out, gutter)
		}
		out = append(out, c)
	}
	return out
}

func runeLen(s string) int { return utf8.RuneCountInString(s) }

// RenderLong lays the entire song out as a single tall column (header on top,
// every section stacked). It is the content the scroll mode windows over, so it
// never drops lines to fit the screen. Returns the styled lines.
func RenderLong(song *chordpro.Song, width int, th *Theme) []string {
	if th == nil {
		th = DefaultTheme()
	}
	if width < 20 {
		width = 20
	}
	header := buildHeader(song, width, th)
	body := packColumns(buildBlocks(song, th), width, 1<<30) // huge cap => one column

	// Center both blocks so the song sits in the middle of wide screens with
	// margins either side; text inside each block stays left-aligned.
	header = lipgloss.PlaceHorizontal(width, lipgloss.Center, header)
	body = lipgloss.PlaceHorizontal(width, lipgloss.Center, body)

	var lines []string
	lines = append(lines, strings.Split(header, "\n")...)
	lines = append(lines, "")
	lines = append(lines, strings.Split(body, "\n")...)
	return lines
}
