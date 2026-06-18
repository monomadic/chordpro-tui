package render

import (
	"strings"
	"testing"

	"chordpro-tui/internal/chordpro"
)

func chartSong() *chordpro.Song {
	return &chordpro.Song{
		Title: "Test",
		Key:   "G",
		Sections: []chordpro.Section{{
			Lines: []chordpro.Line{{Segments: []chordpro.Segment{
				{Chord: "G", Text: "one"},
				{Chord: "C", Text: "two"},
				{Chord: "G", Text: "again"}, // duplicate, should appear once
				{Chord: "D", Text: "three"},
			}}},
		}},
	}
}

func TestRenderChordSheet(t *testing.T) {
	out := RenderChordSheet(chartSong(), 80, 30, DefaultTheme())
	plain := stripANSI(out)

	for _, want := range []string{"Test", "Chord shapes · key of G", "transpose"} {
		if !strings.Contains(plain, want) {
			t.Errorf("sheet missing %q", want)
		}
	}
	// Three distinct chords (G appears twice but should render once): count the
	// fretboard top borders.
	if got := strings.Count(plain, "┍"); got != 3 {
		t.Errorf("expected 3 diagrams, found %d", got)
	}
}

func TestRenderChordSheetNarrow(t *testing.T) {
	// On a narrow screen the diagrams must stack one-per-row rather than spill
	// past the edge. The fretboard grid (lines carrying box-drawing glyphs)
	// must stay within the width; long header/footer hint text may overflow,
	// just as the main song renderer's footer does.
	const w = 24
	out := RenderChordSheet(chartSong(), w, 30, DefaultTheme())
	for _, ln := range strings.Split(out, "\n") {
		plain := stripANSI(ln)
		if !strings.ContainsAny(plain, "┍│└├") {
			continue // header/footer text, exempt
		}
		if runeLen(plain) > w {
			t.Errorf("diagram line exceeds width %d: %d %q", w, runeLen(plain), plain)
		}
	}
}

func stripANSI(s string) string {
	var b strings.Builder
	inEsc := false
	for _, r := range s {
		switch {
		case r == 0x1b:
			inEsc = true
		case inEsc && r == 'm':
			inEsc = false
		case inEsc:
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}

func TestShapeForPrefersDefine(t *testing.T) {
	song := &chordpro.Song{
		Defines: map[string]chordpro.ChordDefinition{
			"G": {Name: "G", BaseFret: 1, Frets: []int{3, 2, 0, 0, 3, 3}},
		},
	}
	sh, ok := shapeFor(song, "G")
	if !ok {
		t.Fatal("define not used")
	}
	if sh.BaseFret != 1 || len(sh.Frets) != 6 || sh.Frets[5] != 3 {
		t.Errorf("custom shape not returned: %+v", sh)
	}
	// Chords without a define fall back to the built-in library.
	if _, ok := shapeFor(song, "C"); !ok {
		t.Error("library fallback failed for C")
	}
}
