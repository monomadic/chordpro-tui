package chordpro

import (
	"strings"
	"testing"
	"time"
)

func TestTransposeChord(t *testing.T) {
	cases := []struct {
		chord string
		n     int
		want  string
	}{
		{"C", 2, "D"},
		{"G", 2, "A"},
		{"A", 3, "C"},
		{"C", 1, "C#"},
		{"E", 1, "F"},
		{"B", 1, "C"},
		{"Am", 2, "Bm"},
		{"G7", 2, "A7"},
		{"Cmaj7", 5, "Fmaj7"},
		{"F#m", 1, "Gm"},
		{"C", 12, "C"}, // octave
		{"C", -1, "B"}, // wrap down
		{"D", -2, "C"},
	}
	for _, c := range cases {
		if got := transposeChord(c.chord, c.n); got != c.want {
			t.Errorf("transpose(%q, %d) = %q, want %q", c.chord, c.n, got, c.want)
		}
	}
}

// TestTransposeFixedSpelling pins the chosen enharmonic spelling for every
// accidental pitch class: only Eb and Bb are flats; C#, F#, G# are sharps.
func TestTransposeFixedSpelling(t *testing.T) {
	cases := []struct {
		chord, want string
	}{
		{"C", "C#"}, // C#, not Db
		{"D", "Eb"}, // Eb, not D#
		{"F", "F#"}, // F#, not Gb
		{"G", "G#"}, // G#, not Ab
		{"A", "Bb"}, // Bb, not A#
		{"Db", "D"}, // up a semitone from Db
		{"A#", "B"}, // input sharp accidental still parses
		{"Gb", "G"}, // input flat accidental still parses
	}
	for _, c := range cases {
		if got := transposeChord(c.chord, 1); got != c.want {
			t.Errorf("transpose(%q, +1) = %q, want %q", c.chord, got, c.want)
		}
	}
	// Inputs already on a black key normalise to the fixed spelling (n=0).
	for in, want := range map[string]string{"A#": "Bb", "Db": "C#", "Gb": "F#", "D#": "Eb", "Ab": "G#"} {
		if got := transposeChord(in, 0); got != want {
			t.Errorf("normalise(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestTransposeSlashChord(t *testing.T) {
	if got := transposeChord("G/B", 2); got != "A/C#" {
		t.Errorf("got %q, want A/C#", got)
	}
	if got := transposeChord("D/F#", 1); got != "Eb/G" {
		t.Errorf("got %q, want Eb/G", got)
	}
	// E/G# stays sharp (G# chosen over Ab).
	if got := transposeChord("E/G#", 0); got != "E/G#" {
		t.Errorf("got %q, want E/G#", got)
	}
}

func TestTransposeNonChordUnchanged(t *testing.T) {
	for _, c := range []string{"", "N.C.", "%", "x4"} {
		if got := transposeChord(c, 3); got != c {
			t.Errorf("transpose(%q) = %q, want unchanged", c, got)
		}
	}
}

func TestTransposeKey(t *testing.T) {
	cases := map[string]string{
		"C": "C#", // +1
		"A": "Bb", // wraps to the fixed flat
		"D": "Eb",
		"G": "G#",
		"F": "F#",
	}
	for in, want := range cases {
		if got := TransposeKey(in, 1); got != want {
			t.Errorf("TransposeKey(%q, 1) = %q, want %q", in, got, want)
		}
	}
	if got := TransposeKey("Am", 3); got != "Cm" {
		t.Errorf("TransposeKey(Am, 3) = %q, want Cm", got)
	}
	if got := TransposeKey("G", 2); got != "A" {
		t.Errorf("TransposeKey(G, 2) = %q, want A", got)
	}
}

func TestTransposedSongPropagates(t *testing.T) {
	song, err := ParseString("{title: X}\n{key: G}\n[G]hi [D]there\n")
	if err != nil {
		t.Fatal(err)
	}
	tr := song.Transposed(2)
	if tr.Key != "A" {
		t.Errorf("key = %q, want A", tr.Key)
	}
	if tr.TransposeBy != 2 {
		t.Errorf("TransposeBy = %d", tr.TransposeBy)
	}
	segs := tr.Sections[0].Lines[0].Segments
	if segs[0].Chord != "A" || segs[1].Chord != "E" {
		t.Errorf("chords = %q, %q, want A, E", segs[0].Chord, segs[1].Chord)
	}
	// Original must be untouched.
	if song.Sections[0].Lines[0].Segments[0].Chord != "G" {
		t.Error("transpose mutated the original song")
	}
}

func TestAlternateTuningTitle(t *testing.T) {
	cases := []struct {
		title string
		n     int
		want  string
	}{
		{"Stolen Car", 1, "Stolen Car (Alternate Tuning: +1)"},
		{"Stolen Car", -4, "Stolen Car (Alternate Tuning: -4)"},
		{"", 2, "(Alternate Tuning: +2)"},
	}
	for _, c := range cases {
		if got := AlternateTuningTitle(c.title, c.n); got != c.want {
			t.Errorf("AlternateTuningTitle(%q, %d) = %q, want %q", c.title, c.n, got, c.want)
		}
	}
}

func TestTransposeSource(t *testing.T) {
	src := `{title: Stolen Car}
{key: Em}
{capo: 2}
# editor note
{comment: Intro}
{define: Em base-fret 1 frets 0 2 2 0 0 0}
[Em]Stolen [G]car [*let ring]
{start_of_tab}
e|--0--2--|
{end_of_tab}
`
	got := TransposeSource(src, 2, AlternateTuningTitle("Stolen Car", 2))

	mustContain := []string{
		"{title: Stolen Car (Alternate Tuning: +2)}", // title replaced
		"{key: F#m}",       // key transposed
		"{capo: 2}",        // capo untouched
		"# editor note",    // remark preserved
		"{comment: Intro}", // comment preserved
		"{define: Em base-fret 1 frets 0 2 2 0 0 0}", // define preserved
		"[F#m]Stolen [A]car [*let ring]",             // chords transposed, annotation kept
		"e|--0--2--|",                                // tab content verbatim (not transposed)
	}
	for _, want := range mustContain {
		if !strings.Contains(got, want) {
			t.Errorf("transposed source missing %q\n---\n%s", want, got)
		}
	}
}

func TestTransposeSourcePrependsTitle(t *testing.T) {
	got := TransposeSource("[C]hi\n", 2, "New (Alternate Tuning: +2)")
	if !strings.HasPrefix(got, "{title: New (Alternate Tuning: +2)}\n") {
		t.Errorf("title not prepended when source had none:\n%s", got)
	}
	if !strings.Contains(got, "[D]hi") {
		t.Errorf("chord not transposed: %q", got)
	}
}

func TestParseDuration(t *testing.T) {
	cases := map[string]time.Duration{
		"3:30":    3*time.Minute + 30*time.Second,
		"0:45":    45 * time.Second,
		"1:02:03": time.Hour + 2*time.Minute + 3*time.Second,
		"210":     210 * time.Second,
		"":        0,
		"bogus":   0,
	}
	for in, want := range cases {
		if got := parseDuration(in); got != want {
			t.Errorf("parseDuration(%q) = %v, want %v", in, got, want)
		}
	}
}
