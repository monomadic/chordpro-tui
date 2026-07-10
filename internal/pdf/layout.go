package pdf

import (
	"strings"

	"chordpro-tui/internal/chordpro"
	"chordpro-tui/internal/chords"

	"github.com/go-pdf/fpdf"
)

// All layout arithmetic happens in "em" units: multiples of the body font
// size. Because every font size, gap, and gutter below is defined relative to
// the body size, a layout measured once at scale 1 fits the page at exactly
// scale = min(availW/width, availH/height) — the fit search just picks the
// column count that maximizes that scale.
const (
	lyricSize = 1.0 // the body size; everything else is relative to it

	// A chord row and its lyric row form one visual unit: the chord baseline
	// sits low in a short chord row (hugging the lyric), while the lyric row
	// carries the slack below its baseline, so the gap to the next chord/lyric
	// pair is clearly larger than the gap inside the pair.
	lyricLH     = 1.45
	lyricBase   = 0.82 // lyric baseline within its row
	chordSize   = 0.88
	chordLH     = 0.95
	chordBase   = 0.74 // chord baseline within its row
	commentSize = 0.92
	tabSize     = 0.95
	tabLH       = 1.15
	labelSize   = 0.78
	labelLH     = 1.6
	blankLH     = 0.72
	sectionGap  = 1.05 // vertical gap between section blocks in a column
	gutterW     = 2.8  // horizontal gap between columns
	insetW      = 0.95 // left inset inside every block (the chorus bar's lane)
	chordGap    = 0.38 // minimum space between adjacent chords in a row

	titleSize = 2.2
	titleLH   = 2.7
	subSize   = 0.85
	subLH     = 1.5
	metaSize = 0.72
	metaLH   = 1.9  // row holding the outlined metadata pills
	pillH    = 1.25 // outlined pill box height
	pillPadX = 0.42 // horizontal padding inside a pill box
	pillGap  = 0.55 // gap between metadata pills
	kvGap    = 0.3  // gap between a pill's label and value
	headerGap = 1.7 // gap between the header and the song body

	// Chord diagram geometry (sheet-music style fingering grids in the header).
	dStrGap    = 0.54 // horizontal gap between strings
	dFretGap   = 0.60 // vertical gap between frets
	dFretRows  = 4    // fret rows shown (the dataset uses 4-fret windows)
	dNameLH    = 1.1  // row holding the chord name above the grid
	dMarkLH    = 0.48 // row holding the open/muted (o/x) markers
	dFingerLH  = 0.65 // row holding the finger numbers below the grid
	dNameSize  = 0.78
	markSize   = 0.42 // tiny type: o/x markers, finger numbers, "3fr"
	dCellGap   = 1.1  // gap between diagrams in a row
	dRowGap    = 0.55 // gap between wrapped diagram rows
	dTopGap    = 0.65 // gap between the metadata row and the diagrams
	diagPerRow = 8    // wrap the diagram row for chord-heavy songs

	diagH = dNameLH + dMarkLH + dFretRows*dFretGap + dFingerLH
)

// fontSet maps each text role to a core-font (family, style) pair. Core PDF
// fonts need no embedding, so output stays small and works everywhere.
type fontSet struct {
	lyric, chord, annot, comment, tab, label, title, meta, mark [2]string
}

// newFonts builds the font mapping: sans-serif (Helvetica) throughout by
// default for a clean sheet-music look; serif swaps the lyric voice to Times
// while headings, chords, and labels stay sans.
func newFonts(serif bool) fontSet {
	body := "Helvetica"
	if serif {
		body = "Times"
	}
	return fontSet{
		lyric:   [2]string{body, ""},
		chord:   [2]string{"Helvetica", "B"},
		annot:   [2]string{body, "I"},
		comment: [2]string{body, "I"},
		tab:     [2]string{"Courier", ""},
		label:   [2]string{"Helvetica", "BI"},
		title:   [2]string{"Helvetica", "B"},
		meta:    [2]string{"Helvetica", "B"},
		mark:    [2]string{"Helvetica", ""},
	}
}

// measurer returns text widths in em units. It owns a throwaway fpdf document
// so measuring never disturbs the font state of the document being drawn
// (core-font metrics are identical across documents).
type measurer struct {
	doc *fpdf.Fpdf
	tr  func(string) string
}

func newMeasurer(tr func(string) string) measurer {
	doc := fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "pt",
		Size:    fpdf.SizeType{Wd: 1000, Ht: 1000},
	})
	doc.AddPage()
	return measurer{doc: doc, tr: tr}
}

// width measures text set in font f at ratio × body size, in em units.
func (m measurer) width(f [2]string, ratio float64, text string) float64 {
	m.doc.SetFont(f[0], f[1], 1000)
	return m.doc.GetStringWidth(m.tr(text)) / 1000 * ratio
}

// run is one piece of text positioned within a line, in em units from the
// block's content origin.
type run struct {
	text   string
	x      float64
	italic bool // an annotation in the chord row: italic body font, not bold
}

// layLine is one measured source line: a chord row over a lyric row, or a
// comment / tab / blank line.
type layLine struct {
	comment string // non-empty: a {comment} line
	tab     string // non-empty: a verbatim tablature line
	blank   bool
	chords  []run
	lyrics  []run
	width   float64
	height  float64
}

// layBlock is one measured section, sized at scale 1.
type layBlock struct {
	label  string
	chorus bool
	lines  []layLine
	width  float64
	height float64
}

// diagram is one chord fingering grid in the header, with its cell width.
type diagram struct {
	shape chords.Shape
	w     float64
}

// headerLay is the measured title block: title, byline, metadata row, and
// rows of chord fingering diagrams.
type headerLay struct {
	title, sub string
	meta       [][2]string
	diagRows   [][]diagram
	width      float64
	height     float64
}

// layout is a fitted page: the chosen column split and the body font size (in
// points) at which it fills the available area.
type layout struct {
	header headerLay
	cols   [][]layBlock
	scale  float64
	capped bool    // scale was clamped by MaxFont (a short song): center, don't justify
	bodyW  float64 // em, at the chosen column split
	bodyH  float64 // em
}

// fit measures the song and returns the column split and scale that yield the
// largest body font within availW × availH (points). forceCols pins the
// column count; maxFont caps the body size so short songs don't balloon.
func fit(song *chordpro.Song, m measurer, f fontSet, availW, availH, maxFont float64, forceCols int, noDiags bool) layout {
	blocks := buildBlocks(song, m, f)
	hdr := buildHeader(song, m, f, noDiags)

	lo, hi := 1, 4
	if forceCols > 0 {
		lo, hi = forceCols, forceCols
	}

	best := layout{header: hdr, scale: -1}
	for n := lo; n <= hi; n++ {
		cols := packInto(blocks, n, sectionGap)
		w := max(layoutW(cols), hdr.width)
		h := hdr.height + headerGap + maxColH(cols, sectionGap)
		if w <= 0 || h <= 0 {
			continue
		}
		s := min(availW/w, availH/h)
		if s > best.scale {
			best = layout{header: hdr, cols: cols, scale: s,
				bodyW: layoutW(cols), bodyH: maxColH(cols, sectionGap)}
		}
		if len(cols) < n {
			break // blocks can't split further; more columns won't differ
		}
	}
	best.capped = best.scale > maxFont
	best.scale = min(best.scale, maxFont)
	return best
}

// buildBlocks measures each song section into a block at scale 1.
func buildBlocks(song *chordpro.Song, m measurer, f fontSet) []layBlock {
	var blocks []layBlock
	for _, sec := range song.Sections {
		b := layBlock{label: sec.Label, chorus: sec.Kind == chordpro.KindChorus}
		for _, ln := range sec.Lines {
			b.lines = append(b.lines, buildLine(ln, sec.Kind, m, f))
		}
		b.lines = tidyBlankLines(b.lines)
		if len(b.lines) == 0 && b.label == "" {
			continue
		}
		if b.label != "" {
			b.height += labelLH
			b.width = m.width(f.label, labelSize, b.label)
		}
		for _, l := range b.lines {
			b.height += l.height
			b.width = max(b.width, l.width)
		}
		b.width += insetW
		blocks = append(blocks, b)
	}
	return blocks
}

// buildLine measures one source line, positioning each chord over the syllable
// it belongs to. When a chord is wider than its syllable, the following lyric
// runs are pushed right so later chords still land over their own syllables —
// the same rule the terminal renderer applies, with float widths instead of
// character cells.
func buildLine(ln chordpro.Line, kind chordpro.SectionKind, m measurer, f fontSet) layLine {
	if ln.Comment != "" {
		return layLine{comment: ln.Comment, height: lyricLH,
			width: m.width(f.comment, commentSize, ln.Comment)}
	}
	if kind == chordpro.KindTab {
		t := ln.PlainText()
		return layLine{tab: t, height: tabLH, width: m.width(f.tab, tabSize, t)}
	}
	if ln.IsBlank() {
		return layLine{blank: true, height: blankLH}
	}

	var l layLine
	chordX, lyricX := 0.0, 0.0
	for _, seg := range ln.Segments {
		if mk, annot := seg.Marker(); mk != "" {
			chordX = max(chordX, lyricX)
			font := f.chord
			if annot {
				font = f.annot
			}
			l.chords = append(l.chords, run{text: mk, x: chordX, italic: annot})
			chordX += m.width(font, chordSize, mk) + chordGap
		}
		if seg.Text != "" {
			l.lyrics = append(l.lyrics, run{text: seg.Text, x: lyricX})
			lyricX += m.width(f.lyric, lyricSize, seg.Text)
		}
		lyricX = max(lyricX, chordX-chordGap)
	}
	l.width = max(lyricX, chordX-chordGap)
	l.height = lyricLH
	if len(l.chords) > 0 {
		l.height += chordLH
	}
	return l
}

// tidyBlankLines collapses runs of blank lines and trims them from both ends,
// so stray whitespace in a song doesn't open gaps on the page.
func tidyBlankLines(lines []layLine) []layLine {
	var out []layLine
	prevBlank := false
	for _, l := range lines {
		if l.blank && prevBlank {
			continue
		}
		out = append(out, l)
		prevBlank = l.blank
	}
	for len(out) > 0 && out[0].blank {
		out = out[1:]
	}
	for len(out) > 0 && out[len(out)-1].blank {
		out = out[:len(out)-1]
	}
	return out
}

// buildHeader measures the title block: title, byline, the metadata row (key,
// capo, tempo, ...), and fingering diagrams for every chord the song uses.
func buildHeader(song *chordpro.Song, m measurer, f fontSet, noDiags bool) headerLay {
	h := headerLay{title: song.Title, sub: subtitleLine(song), meta: song.Meta()}
	if h.title == "" {
		h.title = "Untitled"
	}
	h.width = m.width(f.title, titleSize, h.title)
	h.height = titleLH
	if h.sub != "" {
		h.width = max(h.width, m.width(f.meta, subSize, h.sub))
		h.height += subLH
	}
	if len(h.meta) > 0 {
		h.width = max(h.width, metaWidth(m, f, h.meta))
		h.height += metaLH
	}
	if !noDiags {
		h.diagRows = buildDiagrams(song, m, f)
		for _, row := range h.diagRows {
			h.width = max(h.width, diagRowW(row))
		}
		if n := len(h.diagRows); n > 0 {
			h.height += dTopGap + float64(n)*diagH + float64(n-1)*dRowGap
		}
	}
	return h
}

// buildDiagrams resolves a fingering for each distinct chord in first-use
// order (a song's {define} wins over the built-in library) and wraps them
// into rows. Chords with no catalogued shape are silently skipped.
func buildDiagrams(song *chordpro.Song, m measurer, f fontSet) [][]diagram {
	var diags []diagram
	for _, name := range uniqueChords(song) {
		sh, ok := shapeFor(song, name)
		if !ok || len(sh.Frets) < 2 {
			continue
		}
		w := max(float64(len(sh.Frets)-1)*dStrGap, m.width(f.chord, dNameSize, sh.Name))
		if sh.BaseFret > 1 {
			w += m.width(f.mark, markSize, frLabel(sh.BaseFret)) + 0.15
		}
		diags = append(diags, diagram{shape: sh, w: w})
	}
	var rows [][]diagram
	for i := 0; i < len(diags); i += diagPerRow {
		rows = append(rows, diags[i:min(i+diagPerRow, len(diags))])
	}
	return rows
}

// diagRowW is the total width of one row of diagrams, in em units.
func diagRowW(row []diagram) float64 {
	w := 0.0
	for i, d := range row {
		if i > 0 {
			w += dCellGap
		}
		w += d.w
	}
	return w
}

// shapeFor resolves a chord's fingering, preferring the song's own {define}
// over the built-in library so custom voicings win when present.
func shapeFor(song *chordpro.Song, name string) (chords.Shape, bool) {
	if d, ok := song.Defines[name]; ok {
		return chords.Shape{Name: name, Frets: d.Frets, BaseFret: d.BaseFret}, true
	}
	return chords.Lookup(name)
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

// subtitleLine merges artist, subtitle, and album into one bullet-joined line.
func subtitleLine(song *chordpro.Song) string {
	var parts []string
	for _, p := range []string{song.Artist, song.Subtitle, song.Album} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "  •  ")
}

// metaWidth is the total width of the metadata row in em units: each pill is
// an outlined box padded around its label and value.
func metaWidth(m measurer, f fontSet, meta [][2]string) float64 {
	w := 0.0
	for i, kv := range meta {
		if i > 0 {
			w += pillGap
		}
		w += pillW(m, f, kv)
	}
	return w
}

// pillW is the outlined box width of one metadata pill, in em units.
func pillW(m measurer, f fontSet, kv [2]string) float64 {
	return pillPadX + m.width(f.meta, metaSize, kv[0]) + kvGap + m.width(f.mark, metaSize, kv[1]) + pillPadX
}

// packInto distributes blocks into at most n balanced columns, keeping blocks
// whole. It targets an even height per column, growing the target until the
// blocks actually fit in n columns.
func packInto(blocks []layBlock, n int, gap float64) [][]layBlock {
	if len(blocks) == 0 {
		return nil
	}
	if n <= 1 {
		return [][]layBlock{blocks}
	}
	target := totalH(blocks, gap) / float64(n)
	for {
		cols := greedyFill(blocks, target, gap)
		if len(cols) <= n {
			return cols
		}
		target *= 1.05
	}
}

// greedyFill packs blocks top-to-bottom, starting a new column whenever adding
// the next block (plus its separator) would exceed target.
func greedyFill(blocks []layBlock, target, gap float64) [][]layBlock {
	var cols [][]layBlock
	var col []layBlock
	colH := 0.0
	for _, b := range blocks {
		add := b.height
		if len(col) > 0 {
			add += gap
		}
		if len(col) > 0 && colH+add > target+1e-9 {
			cols = append(cols, col)
			col, colH = nil, 0
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

func totalH(blocks []layBlock, gap float64) float64 {
	t := 0.0
	for i, b := range blocks {
		if i > 0 {
			t += gap
		}
		t += b.height
	}
	return t
}

func maxColH(cols [][]layBlock, gap float64) float64 {
	h := 0.0
	for _, c := range cols {
		h = max(h, totalH(c, gap))
	}
	return h
}

func colW(col []layBlock) float64 {
	w := 0.0
	for _, b := range col {
		w = max(w, b.width)
	}
	return w
}

// layoutW is the rendered width of a column split: each column sized to its
// widest block, joined by gutters.
func layoutW(cols [][]layBlock) float64 {
	w := 0.0
	for i, c := range cols {
		if i > 0 {
			w += gutterW
		}
		w += colW(c)
	}
	return w
}
