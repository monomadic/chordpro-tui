package pdf

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"chordpro-tui/internal/chordpro"
)

func loadSong(t *testing.T) *chordpro.Song {
	t.Helper()
	f, err := os.Open("../../testdata/wagon_wheel.cho")
	if err != nil {
		t.Fatalf("open test song: %v", err)
	}
	defer f.Close()
	song, err := chordpro.Parse(f)
	if err != nil {
		t.Fatalf("parse test song: %v", err)
	}
	return song
}

func TestExportProducesSinglePagePDF(t *testing.T) {
	song := loadSong(t)
	var buf bytes.Buffer
	res, err := Export(song, Options{PageW: 744, PageH: 1133}, &buf)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	out := buf.String()
	if !strings.HasPrefix(out, "%PDF-") {
		t.Fatalf("output does not start with %%PDF-, got %q", out[:min(len(out), 16)])
	}
	if !strings.Contains(out, "/Count 1") {
		t.Errorf("expected exactly one page (/Count 1) in output")
	}
	if res.Columns < 1 {
		t.Errorf("Columns = %d, want >= 1", res.Columns)
	}
	if res.BodyPt <= 0 || res.BodyPt > 20 {
		t.Errorf("BodyPt = %g, want in (0, 20]", res.BodyPt)
	}
}

func TestLongerSongGetsSmallerType(t *testing.T) {
	song := loadSong(t)
	long := *song
	long.Sections = nil
	for i := 0; i < 6; i++ {
		long.Sections = append(long.Sections, song.Sections...)
	}

	var a, b bytes.Buffer
	short, err := Export(song, Options{PageW: 744, PageH: 1133}, &a)
	if err != nil {
		t.Fatalf("Export short: %v", err)
	}
	big, err := Export(&long, Options{PageW: 744, PageH: 1133}, &b)
	if err != nil {
		t.Fatalf("Export long: %v", err)
	}
	if big.BodyPt >= short.BodyPt {
		t.Errorf("6x-long song body %g pt, want smaller than original %g pt", big.BodyPt, short.BodyPt)
	}
}

func TestInvertedExport(t *testing.T) {
	song := loadSong(t)
	var buf bytes.Buffer
	if _, err := Export(song, Options{PageW: 744, PageH: 1133, Inverted: true}, &buf); err != nil {
		t.Fatalf("Export inverted: %v", err)
	}
	if !strings.HasPrefix(buf.String(), "%PDF-") {
		t.Fatal("inverted output is not a PDF")
	}
}

func TestForcedColumns(t *testing.T) {
	song := loadSong(t)
	var buf bytes.Buffer
	res, err := Export(song, Options{PageW: 1440, PageH: 900, Columns: 2}, &buf)
	if err != nil {
		t.Fatalf("Export: %v", err)
	}
	if res.Columns != 2 {
		t.Errorf("Columns = %d, want 2", res.Columns)
	}
}

// A chord wider than its syllable must push the following lyrics right, so the
// next chord still lands over its own syllable — the float-width version of
// the terminal renderer's overhang rule.
func TestChordOverhangPushesLyrics(t *testing.T) {
	m := newMeasurer(func(s string) string { return s })
	f := newFonts(false)
	ln := chordpro.Line{Segments: []chordpro.Segment{
		{Chord: "Gmaj7sus4", Text: "a "},
		{Chord: "D", Text: "long tail"},
	}}
	l := buildLine(ln, chordpro.KindVerse, m, f)
	if len(l.chords) != 2 || len(l.lyrics) != 2 {
		t.Fatalf("got %d chords, %d lyric runs; want 2 and 2", len(l.chords), len(l.lyrics))
	}
	firstChordW := m.width(f.chord, chordSize, "Gmaj7sus4")
	if l.lyrics[1].x < firstChordW-1e-9 {
		t.Errorf("second lyric run at %g, want pushed past the wide chord (%g)", l.lyrics[1].x, firstChordW)
	}
	if l.chords[1].x < l.lyrics[1].x-1e-9 {
		t.Errorf("second chord at %g, want at or after its syllable (%g)", l.chords[1].x, l.lyrics[1].x)
	}
	if l.width < l.lyrics[1].x {
		t.Errorf("line width %g smaller than a run inside it (%g)", l.width, l.lyrics[1].x)
	}
}

func TestHeaderIncludesChordDiagrams(t *testing.T) {
	song := loadSong(t)
	m := newMeasurer(func(s string) string { return s })
	f := newFonts(false)

	hdr := buildHeader(song, m, f, false)
	if len(hdr.diagRows) == 0 {
		t.Fatal("no diagram rows for a song whose chords are all in the database")
	}
	got := 0
	for _, row := range hdr.diagRows {
		got += len(row)
	}
	if want := len(uniqueChords(song)); got != want {
		t.Errorf("diagram count = %d, want %d (one per distinct chord)", got, want)
	}

	plain := buildHeader(song, m, f, true)
	if len(plain.diagRows) != 0 {
		t.Errorf("noDiags header still has %d diagram rows", len(plain.diagRows))
	}
	if plain.height >= hdr.height {
		t.Errorf("noDiags header height %g, want smaller than %g", plain.height, hdr.height)
	}
}

func TestPresetLookup(t *testing.T) {
	p, ok := PresetByName("iPad-Mini")
	if !ok || p.W != 744 || p.H != 1133 {
		t.Errorf("PresetByName(iPad-Mini) = %+v, %v", p, ok)
	}
	if _, ok := PresetByName("betamax"); ok {
		t.Errorf("unknown preset unexpectedly found")
	}
}
