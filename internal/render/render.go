package render

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

	"chordpro-tui/internal/chordpro"

	"github.com/charmbracelet/lipgloss"
)

const (
	gutter    = "   " // horizontal space between columns
	chorusBar = "▍ "  // left accent for chorus lines
	colPad    = 1     // extra left padding inside each column
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

// RenderOpts tweaks how a song is rendered.
type RenderOpts struct {
	HideHeader bool   // omit the title/metadata header block
	HideTabs   bool   // fold away tab (tablature) sections
	ViewMode   string // view-mode label shown in the footer badge (fit mode only)
}

// ViewBadge renders the always-on view-mode indicator (e.g. "auto-scroll")
// shown in the bottom-right corner of every view, styled as a metadata pill.
func ViewBadge(th *Theme, label string) string {
	if th == nil {
		th = DefaultTheme()
	}
	return th.PillVal.Render(label)
}

// Render lays the whole song out to fit within width x height and returns the
// composed screen string. When the song is shorter than the screen it is
// centered; when taller it flows into balanced newspaper columns.
func Render(song *chordpro.Song, width, height int, th *Theme) string {
	return RenderWith(song, width, height, th, RenderOpts{})
}

// spacingPlan is a candidate set of vertical paddings. gapBelow is the blank
// rows between the header and the body, and sectionGap the blank rows between
// stacked section blocks within a column.
type spacingPlan struct{ gapBelow, sectionGap int }

// RenderWith is Render with display options.
func RenderWith(song *chordpro.Song, width, height int, th *Theme, opts RenderOpts) string {
	if th == nil {
		th = DefaultTheme()
	}
	if width < 20 {
		width = 20
	}
	if height < 6 {
		height = 6
	}

	header := ""
	headerH := 0
	if !opts.HideHeader {
		header = buildHeader(song, width, th)
		headerH = lipgloss.Height(header)
	}

	blocks := buildBlocks(song, th, opts)

	// Choose vertical spacing. We prefer a roomy layout (a blank line below the
	// header, a blank line between sections), but give those rows back to the
	// song before clipping anything: a layout that's only a line or two too tall
	// otherwise loses content off the bottom. Plans run roomiest → tightest; we
	// take the first that fits, else the tightest (whose body is then clipped,
	// with the footer flagging "more").
	plans := []spacingPlan{{0, 1}, {0, 0}}
	if !opts.HideHeader {
		plans = []spacingPlan{{1, 1}, {0, 1}, {0, 0}}
	}

	var cols [][]block
	var sel spacingPlan
	var budget int
	for _, p := range plans {
		b := height - headerH - p.gapBelow - 1 // -1 for footer
		if b < 1 {
			b = 1
		}
		c := packColumns(blocks, width, b, p.sectionGap)
		cols, sel, budget = c, p, b
		if maxColHeight(c, p.sectionGap) <= b {
			break // fits at this spacing; roomiest wins
		}
	}

	// Fit the body to exactly `budget` rows: clip an over-tall song (so it never
	// buries the footer) or center a short one, keeping the footer pinned.
	bodyLines := strings.Split(renderCols(cols, sel.sectionGap), "\n")
	truncated := false
	if len(bodyLines) > budget {
		bodyLines = bodyLines[:budget]
		truncated = true
	} else if pad := budget - len(bodyLines); pad > 0 {
		top := pad / 2
		bodyLines = append(make([]string, top), bodyLines...)
		bodyLines = append(bodyLines, make([]string, pad-top)...)
	}
	body := lipgloss.PlaceHorizontal(width, lipgloss.Center, strings.Join(bodyLines, "\n"))
	footer := lipgloss.PlaceHorizontal(width, lipgloss.Center, buildFooter(song, width, th, truncated, opts.ViewMode))

	var lines []string
	if !opts.HideHeader {
		lines = append(lines, strings.Split(lipgloss.PlaceHorizontal(width, lipgloss.Center, header), "\n")...)
		for i := 0; i < sel.gapBelow; i++ {
			lines = append(lines, "")
		}
	}
	lines = append(lines, strings.Split(body, "\n")...)
	lines = append(lines, footer)
	return strings.Join(lines, "\n")
}

// buildBlocks turns each song section into a styled block. Tab sections are
// folded away when opts.HideTabs is set.
func buildBlocks(song *chordpro.Song, th *Theme, opts RenderOpts) []block {
	var blocks []block
	for _, sec := range song.Sections {
		if opts.HideTabs && sec.Kind == chordpro.KindTab {
			continue
		}
		var lines []string
		if sec.Label != "" {
			lines = append(lines, th.Section.Render(strings.ToUpper(sec.Label)))
		}
		for _, ln := range sec.Lines {
			lines = append(lines, renderLine(ln, sec.Kind, th)...)
		}
		lines = tidyBlanks(lines)
		// Chorus lines carry a 2-column accent bar; give every other section a
		// matching 2-space indent so all body text lines up under it.
		if sec.Kind == chordpro.KindChorus {
			lines = decorateChorus(lines, th)
		} else {
			lines = indentLines(lines, 2)
		}
		if len(lines) > 0 {
			blocks = append(blocks, newBlock(lines))
		}
	}
	return blocks
}

// tidyBlanks collapses any run of blank lines down to a single blank and trims
// blank lines from the start and end, so stray whitespace in a song doesn't
// open up gaps in the rendered output.
func tidyBlanks(lines []string) []string {
	var out []string
	prevBlank := false
	for _, l := range lines {
		blank := lipgloss.Width(l) == 0
		if blank && prevBlank {
			continue
		}
		out = append(out, l)
		prevBlank = blank
	}
	for len(out) > 0 && lipgloss.Width(out[0]) == 0 {
		out = out[1:]
	}
	for len(out) > 0 && lipgloss.Width(out[len(out)-1]) == 0 {
		out = out[:len(out)-1]
	}
	return out
}

// renderLine produces the styled rows for a single source line: a chord row
// stacked above the lyric row, a comment, or a verbatim tab row.
func renderLine(ln chordpro.Line, kind chordpro.SectionKind, th *Theme) []string {
	if ln.Comment != "" {
		return []string{th.Comment.Render(ln.Comment)}
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
	if ln.HasMarkers() {
		out = append(out, styleChordRow(ln.Segments, th))
	}
	out = append(out, th.Lyric.Render(lyricRow))
	return out
}

// styleChordRow builds the chord row with each chord individually styled, so
// the background hugs each chord as a pill. Annotations ([*...]) sit in the same
// row but are rendered as plain italic text, not pills. It reproduces
// alignChords' spacing exactly, so markers stay aligned with the lyric row.
func styleChordRow(segs []chordpro.Segment, th *Theme) string {
	var b strings.Builder
	chordVis, lyricVis := 0, 0
	for _, seg := range segs {
		if mk, annot := seg.Marker(); mk != "" {
			if gap := lyricVis - chordVis; gap > 0 {
				b.WriteString(strings.Repeat(" ", gap))
				chordVis += gap
			}
			if annot {
				b.WriteString(th.Annotation.Render(mk))
			} else {
				b.WriteString(th.Chord.Render(mk))
			}
			chordVis += runeLen(mk)
			b.WriteByte(' ') // unstyled separator between markers
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
// positioning each marker (chord or annotation) directly above the syllable it
// applies to.
func alignChords(segs []chordpro.Segment) (chordRow, lyricRow string) {
	var chord, lyric strings.Builder
	for _, seg := range segs {
		if mk, _ := seg.Marker(); mk != "" {
			// Pad the chord row out to the current lyric position.
			if gap := runeLen(lyric.String()) - runeLen(chord.String()); gap > 0 {
				chord.WriteString(strings.Repeat(" ", gap))
			}
			chord.WriteString(mk)
			chord.WriteByte(' ') // keep adjacent markers from touching
		}
		lyric.WriteString(seg.Text)
		// If the marker overhangs its syllable, push following lyrics right so
		// the next marker still lands in the right place.
		if gap := runeLen(chord.String()) - runeLen(lyric.String()); gap > 0 {
			lyric.WriteString(strings.Repeat(" ", gap))
		}
	}
	return chord.String(), lyric.String()
}

// indentLines prefixes n spaces to every line of a block, shifting it right to
// line up with the chorus body (whose accent bar occupies the same columns).
func indentLines(lines []string, n int) []string {
	pad := strings.Repeat(" ", n)
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = pad + l
	}
	return out
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

// packColumns flows blocks into balanced newspaper columns and returns the
// chosen column groups. It searches for the fewest columns that keep every
// column within maxH (using gap blank rows between stacked blocks), while never
// letting the real layout exceed width — so the result never overflows
// horizontally. Each candidate is measured for its actual width (columns size
// to their own content), so legitimate multi-column layouts are not rejected by
// one long line elsewhere in the song.
func packColumns(blocks []block, width, maxH, gap int) [][]block {
	if len(blocks) == 0 {
		return nil
	}

	var chosen [][]block
	for n := 1; n <= len(blocks); n++ {
		cols := packInto(blocks, n, gap)
		if layoutWidth(cols) > width {
			if chosen == nil {
				chosen = cols // even a single column is too wide; render anyway
			}
			break // adding columns only gets wider
		}
		chosen = cols
		if maxColHeight(cols, gap) <= maxH {
			break // fits the page height with the fewest columns that fit width
		}
		if len(cols) < n {
			break // blocks can't be split into more columns; this is as tall as it gets
		}
	}
	return chosen
}

// renderCols stacks each column (with gap blank rows between blocks) and joins
// them side by side with gutters.
func renderCols(cols [][]block, gap int) string {
	if len(cols) == 0 {
		return ""
	}
	rendered := make([]string, len(cols))
	for i, c := range cols {
		rendered[i] = renderColumn(c, gap)
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

// maxColHeight is the height of the tallest column, counting gap blank rows
// between stacked blocks.
func maxColHeight(cols [][]block, gap int) int {
	h := 0
	for _, c := range cols {
		if ch := totalHeight(c, gap); ch > h {
			h = ch
		}
	}
	return h
}

func colWidth(blocks []block) int { return maxWidth(blocks) }

// packInto distributes blocks into at most n balanced columns, keeping blocks
// whole. It targets an even height per column, bumping the target until the
// blocks actually fit in n columns.
func packInto(blocks []block, n, gap int) [][]block {
	if n <= 1 {
		return [][]block{blocks}
	}
	total := totalHeight(blocks, gap)
	target := ceilDiv(total, n)
	for {
		cols := greedyFill(blocks, target, gap)
		if len(cols) <= n {
			return cols
		}
		target++ // too many columns at this target; allow taller columns
	}
}

// greedyFill packs blocks top-to-bottom, starting a new column whenever adding
// the next block (plus its separator) would exceed target.
func greedyFill(blocks []block, target, gap int) [][]block {
	var cols [][]block
	var col []block
	colH := 0
	for _, b := range blocks {
		add := b.height
		if len(col) > 0 {
			add += gap // blank separator
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

func totalHeight(blocks []block, gap int) int {
	t := 0
	for i, b := range blocks {
		if i > 0 {
			t += gap // separator
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

// renderColumn stacks a column's blocks with gap blank separators between them,
// left-aligned to the column's widest block.
func renderColumn(blocks []block, gap int) string {
	w := 0
	for _, b := range blocks {
		if b.width > w {
			w = b.width
		}
	}
	var lines []string
	for i, b := range blocks {
		if i > 0 {
			for j := 0; j < gap; j++ {
				lines = append(lines, "")
			}
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

// ApplyBackground tints every cell of an already-rendered screen with bg,
// padding each line out to width w. It re-asserts the background after every
// ANSI reset, so foreground-styled text keeps the themed fill while element
// backgrounds (chord/metadata pills) still paint over it. Returns the input
// unchanged if bg is not a parseable "#rrggbb" color.
func ApplyBackground(screen string, w int, bg lipgloss.Color) string {
	r, g, b, ok := hexRGB(string(bg))
	if !ok {
		return screen
	}
	set := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)
	const reset = "\x1b[0m"
	lines := strings.Split(screen, "\n")
	for i, ln := range lines {
		if pad := w - lipgloss.Width(ln); pad > 0 {
			ln += strings.Repeat(" ", pad)
		}
		lines[i] = set + strings.ReplaceAll(ln, reset, reset+set) + reset
	}
	return strings.Join(lines, "\n")
}

// hexRGB parses a "#rrggbb" color into 8-bit components.
func hexRGB(s string) (r, g, b int, ok bool) {
	s = strings.TrimPrefix(s, "#")
	if len(s) != 6 {
		return 0, 0, 0, false
	}
	v, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return int(v >> 16), int(v>>8) & 0xff, int(v) & 0xff, true
}

// RenderLong lays the entire song out as a single tall column (header on top,
// every section stacked). It is the content the scroll mode windows over, so it
// never drops lines to fit the screen. Returns the styled lines.
func RenderLong(song *chordpro.Song, width int, th *Theme) []string {
	return RenderLongWith(song, width, th, RenderOpts{})
}

// RenderLongWith is RenderLong with display options.
func RenderLongWith(song *chordpro.Song, width int, th *Theme, opts RenderOpts) []string {
	if th == nil {
		th = DefaultTheme()
	}
	if width < 20 {
		width = 20
	}
	// Center the body so the song sits in the middle of wide screens with
	// margins either side; text inside each block stays left-aligned.
	cols := packColumns(buildBlocks(song, th, opts), width, 1<<30, 1) // huge cap => one column
	body := lipgloss.PlaceHorizontal(width, lipgloss.Center, renderCols(cols, 1))

	var lines []string
	if !opts.HideHeader {
		header := lipgloss.PlaceHorizontal(width, lipgloss.Center, buildHeader(song, width, th))
		lines = append(lines, strings.Split(header, "\n")...)
		lines = append(lines, "") // blank line between header and body
	}
	lines = append(lines, strings.Split(body, "\n")...)
	return lines
}
