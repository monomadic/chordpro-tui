// Package pdf renders a parsed ChordPro song onto a single PDF page —
// chords stacked over lyrics in balanced newspaper columns, scaled so the
// whole song exactly fills the page. Output is monochrome sheet-music style
// (bold sans title, fingering diagrams, italic section labels) using core PDF
// fonts, so files are small and open anywhere offline.
package pdf

import (
	"fmt"
	"io"
	"strconv"

	"chordpro-tui/internal/chordpro"

	"github.com/go-pdf/fpdf"
)

// Options controls page geometry and typography for Export.
type Options struct {
	PageW, PageH float64 // page size in points (required)
	Margin       float64 // page margin in points; 0 = 4.5% of the short edge
	Columns      int     // exact column count; 0 = choose automatically
	MaxFont      float64 // cap on the body font size in points; 0 = 20
	Serif        bool    // Times lyrics instead of the default Helvetica
	NoDiagrams   bool    // omit the chord fingering diagrams in the header
	Inverted     bool    // white text on a black page (screen night mode)
}

// rgb is an ink color.
type rgb struct{ r, g, b int }

// palette holds the page's inks. The default is black-on-white; inverted
// swaps to white-on-black with grays lifted to stay readable on the dark
// ground.
type palette struct {
	bg       rgb // page fill (inverted only)
	fg       rgb // primary text, grids, rules
	gray     rgb // secondary text: comments, diagram marks, fingers
	bar      rgb // chorus accent rule
	inverted bool
}

func newPalette(inverted bool) palette {
	if inverted {
		return palette{
			bg:       rgb{0, 0, 0},
			fg:       rgb{255, 255, 255},
			gray:     rgb{175, 175, 175},
			bar:      rgb{145, 145, 145},
			inverted: true,
		}
	}
	return palette{
		fg:   rgb{0, 0, 0},
		gray: rgb{90, 90, 90},
		bar:  rgb{120, 120, 120},
	}
}

func setText(doc *fpdf.Fpdf, c rgb) { doc.SetTextColor(c.r, c.g, c.b) }
func setDraw(doc *fpdf.Fpdf, c rgb) { doc.SetDrawColor(c.r, c.g, c.b) }
func setFill(doc *fpdf.Fpdf, c rgb) { doc.SetFillColor(c.r, c.g, c.b) }

// Result reports what Export chose, for CLI feedback.
type Result struct {
	PageW, PageH float64
	Columns      int
	BodyPt       float64 // body font size the song was set at
}

// Export lays song out on one PDF page and writes the document to w.
func Export(song *chordpro.Song, opts Options, w io.Writer) (Result, error) {
	if song == nil {
		return Result{}, fmt.Errorf("no song to export")
	}
	if opts.PageW <= 0 || opts.PageH <= 0 {
		return Result{}, fmt.Errorf("page size must be positive, got %g×%g", opts.PageW, opts.PageH)
	}
	margin := opts.Margin
	if margin <= 0 {
		margin = 0.045 * min(opts.PageW, opts.PageH)
	}
	maxFont := opts.MaxFont
	if maxFont <= 0 {
		maxFont = 20
	}

	doc := fpdf.NewCustom(&fpdf.InitType{
		UnitStr: "pt",
		Size:    fpdf.SizeType{Wd: opts.PageW, Ht: opts.PageH},
	})
	doc.SetAutoPageBreak(false, 0)
	doc.SetTitle(song.Title, true)
	if song.Artist != "" {
		doc.SetAuthor(song.Artist, true)
	}
	doc.SetCreator("chordpro-pdf", true)
	doc.AddPage()

	pal := newPalette(opts.Inverted)
	if pal.inverted {
		setFill(doc, pal.bg)
		doc.Rect(0, 0, opts.PageW, opts.PageH, "F")
	}

	f := newFonts(opts.Serif)
	m := newMeasurer(doc.UnicodeTranslatorFromDescriptor(""))
	lay := fit(song, m, f, opts.PageW-2*margin, opts.PageH-2*margin, maxFont, opts.Columns, opts.NoDiagrams)
	draw(doc, m, f, pal, lay, opts.PageW, opts.PageH, margin)

	if err := doc.Error(); err != nil {
		return Result{}, err
	}
	return Result{opts.PageW, opts.PageH, len(lay.cols), lay.scale}, doc.Output(w)
}

// baseline gives the text baseline offset within a row, in em units, for a
// row of height rowLH holding a font of the given size ratio.
func baseline(rowLH, ratio float64) float64 {
	return 0.78*ratio + 0.45*(rowLH-ratio)
}

func setFont(doc *fpdf.Fpdf, f [2]string, size float64) {
	doc.SetFont(f[0], f[1], size)
}

// draw paints a fitted layout onto the document's current page. The content is
// centered horizontally and sits slightly above vertical center when the song
// is smaller than the page (scale capped by MaxFont).
func draw(doc *fpdf.Fpdf, m measurer, f fontSet, pal palette, lay layout, pageW, pageH, margin float64) {
	s := lay.scale
	availH := pageH - 2*margin
	contentH := (lay.header.height + headerGap + lay.bodyH) * s
	y := margin + max(0, (availH-contentH)*0.4)
	cx := pageW / 2

	// Header: bold title, bold byline, metadata row, chord diagrams.
	hdr := lay.header
	tw := m.width(f.title, titleSize, hdr.title) * s
	setFont(doc, f.title, titleSize*s)
	setText(doc, pal.fg)
	doc.Text(cx-tw/2, y+baseline(titleLH, titleSize)*s, m.tr(hdr.title))
	y += titleLH * s
	if hdr.sub != "" {
		sw := m.width(f.meta, subSize, hdr.sub) * s
		setFont(doc, f.meta, subSize*s)
		doc.Text(cx-sw/2, y+baseline(subLH, subSize)*s, m.tr(hdr.sub))
		y += subLH * s
	}
	if len(hdr.meta) > 0 {
		drawMeta(doc, m, f, pal, hdr.meta, cx, y, s)
		y += metaLH * s
	}
	if len(hdr.diagRows) > 0 {
		y += dTopGap * s
		for i, row := range hdr.diagRows {
			if i > 0 {
				y += dRowGap * s
			}
			x := cx - diagRowW(row)*s/2
			for _, d := range row {
				drawDiagram(doc, m, f, pal, d, x, y, s)
				x += (d.w + dCellGap) * s
			}
			y += diagH * s
		}
	}
	y += headerGap * s

	// Body: columns of section blocks. When the page is full (scale not capped)
	// horizontal slack is spread into the gutters, so the outer columns sit at
	// the page margin and the sides match the top. A short, capped song stays
	// centered instead.
	slack := pageW - 2*margin - lay.bodyW*s
	x, extraGap := margin, 0.0
	if !lay.capped && len(lay.cols) > 1 && slack > 0 {
		extraGap = slack / float64(len(lay.cols)-1)
	} else {
		x += max(0, slack/2)
	}
	for _, col := range lay.cols {
		cy := y
		for _, b := range col {
			drawBlock(doc, m, f, pal, b, x, cy, s)
			cy += (b.height + sectionGap) * s
		}
		x += (colW(col)+gutterW)*s + extraGap
	}
}

// drawMeta paints the metadata row (KEY C, CAPO 2, ...) centered on cx: each
// pill is an outlined transparent box holding a bold black label and a black
// value.
func drawMeta(doc *fpdf.Fpdf, m measurer, f fontSet, pal palette, meta [][2]string, cx, y, s float64) {
	x := cx - metaWidth(m, f, meta)*s/2
	boxTop := y + (metaLH-pillH)/2*s
	by := boxTop + (pillH/2+0.28)*s // baseline that optically centers the caps
	setDraw(doc, pal.fg)
	doc.SetLineWidth(0.045 * s)
	setText(doc, pal.fg)
	for i, kv := range meta {
		if i > 0 {
			x += pillGap * s
		}
		doc.Rect(x, boxTop, pillW(m, f, kv)*s, pillH*s, "D")
		tx := x + pillPadX*s
		setFont(doc, f.meta, metaSize*s)
		doc.Text(tx, by, m.tr(kv[0]))
		tx += (m.width(f.meta, metaSize, kv[0]) + kvGap) * s
		setFont(doc, f.mark, metaSize*s)
		doc.Text(tx, by, m.tr(kv[1]))
		x += pillW(m, f, kv) * s
	}
}

// frLabel formats the starting-fret note beside a moved diagram, e.g. "3fr".
func frLabel(baseFret int) string { return strconv.Itoa(baseFret) + "fr" }

// drawDiagram paints one chord fingering grid at (x, y): the chord name, an
// o/x open-muted marker row, the fretboard with dots, a starting-fret note
// for shapes above the nut, and finger numbers under the strings.
func drawDiagram(doc *fpdf.Fpdf, m measurer, f fontSet, pal palette, d diagram, x, y, s float64) {
	sh := d.shape
	gw := float64(len(sh.Frets)-1) * dStrGap
	gx := x + (d.w-gw)/2*s // grid left edge; the cell centers its grid
	strX := func(i int) float64 { return gx + float64(i)*dStrGap*s }

	// Chord name, centered over the grid.
	nw := m.width(f.chord, dNameSize, sh.Name) * s
	setFont(doc, f.chord, dNameSize*s)
	setText(doc, pal.fg)
	doc.Text(gx+gw*s/2-nw/2, y+baseline(dNameLH, dNameSize)*s, m.tr(sh.Name))
	y += dNameLH * s

	// o/x markers above the nut.
	setFont(doc, f.mark, markSize*s)
	setText(doc, pal.gray)
	for i, fr := range sh.Frets {
		mk := ""
		switch {
		case fr < 0:
			mk = "x"
		case fr == 0:
			mk = "o"
		default:
			continue
		}
		doc.Text(strX(i)-m.width(f.mark, markSize, mk)*s/2, y+0.34*s, m.tr(mk))
	}
	y += dMarkLH * s

	// Fretboard grid.
	gh := dFretRows * dFretGap * s
	setDraw(doc, pal.fg)
	doc.SetLineWidth(0.025 * s)
	for i := range sh.Frets {
		doc.Line(strX(i), y, strX(i), y+gh)
	}
	for r := 0; r <= dFretRows; r++ {
		ry := y + float64(r)*dFretGap*s
		doc.Line(gx, ry, gx+gw*s, ry)
	}
	if sh.BaseFret <= 1 {
		doc.SetLineWidth(0.09 * s) // heavy nut
		doc.Line(gx, y, gx+gw*s, y)
	} else {
		setFont(doc, f.mark, markSize*s)
		setText(doc, pal.gray)
		doc.Text(gx+gw*s+0.15*s, y+dFretGap*0.65*s, m.tr(frLabel(sh.BaseFret)))
	}

	// Fretted-string dots. Positive fret values are rows relative to BaseFret.
	setFill(doc, pal.fg)
	for i, fr := range sh.Frets {
		if fr > 0 && fr <= dFretRows {
			doc.Circle(strX(i), y+(float64(fr)-0.5)*dFretGap*s, 0.18*s, "F")
		}
	}
	y += gh

	// Finger numbers under the strings ({define} shapes carry none).
	setFont(doc, f.mark, markSize*s)
	setText(doc, pal.gray)
	for i, fg := range sh.Fingers {
		if fg <= 0 || i >= len(sh.Frets) {
			continue
		}
		t := strconv.Itoa(fg)
		doc.Text(strX(i)-m.width(f.mark, markSize, t)*s/2, y+0.45*s, m.tr(t))
	}
}

// drawBlock paints one section at (x, y): optional label, then each line's
// chord row over its lyric row. Chorus blocks get a vertical accent rule in
// the inset lane, mirroring the terminal renderer's accent bar.
func drawBlock(doc *fpdf.Fpdf, m measurer, f fontSet, pal palette, b layBlock, x, y, s float64) {
	if b.chorus {
		setDraw(doc, pal.bar)
		doc.SetLineWidth(max(0.6, 0.09*s))
		bx := x + 0.12*s
		doc.Line(bx, y+0.15*s, bx, y+b.height*s-0.15*s)
	}
	tx := x + insetW*s
	if b.label != "" {
		setFont(doc, f.label, labelSize*s)
		setText(doc, pal.fg)
		doc.Text(tx, y+baseline(labelLH, labelSize)*s, m.tr(b.label))
		y += labelLH * s
	}
	for _, ln := range b.lines {
		switch {
		case ln.blank:
		case ln.comment != "":
			setFont(doc, f.comment, commentSize*s)
			setText(doc, pal.gray)
			doc.Text(tx, y+lyricBase*s, m.tr(ln.comment))
		case ln.tab != "":
			setFont(doc, f.tab, tabSize*s)
			setText(doc, pal.fg)
			doc.Text(tx, y+baseline(tabLH, tabSize)*s, m.tr(ln.tab))
		default:
			ly := y
			if len(ln.chords) > 0 {
				for _, r := range ln.chords {
					font := f.chord
					if r.italic {
						font = f.annot
					}
					setFont(doc, font, chordSize*s)
					setText(doc, pal.fg)
					doc.Text(tx+r.x*s, ly+chordBase*s, m.tr(r.text))
				}
				ly += chordLH * s
			}
			setFont(doc, f.lyric, lyricSize*s)
			setText(doc, pal.fg)
			for _, r := range ln.lyrics {
				doc.Text(tx+r.x*s, ly+lyricBase*s, m.tr(r.text))
			}
		}
		y += ln.height * s
	}
}
