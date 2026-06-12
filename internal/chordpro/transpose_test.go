package chordpro

import (
	"testing"
	"time"
)

func TestTransposeChordSharps(t *testing.T) {
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
		{"C", 12, "C"},  // octave
		{"C", -1, "B"},  // wrap down
		{"D", -2, "C"},
	}
	for _, c := range cases {
		if got := transposeChord(c.chord, c.n, false); got != c.want {
			t.Errorf("transpose(%q, %d) = %q, want %q", c.chord, c.n, got, c.want)
		}
	}
}

func TestTransposeChordFlats(t *testing.T) {
	if got := transposeChord("C", 1, true); got != "Db" {
		t.Errorf("got %q, want Db", got)
	}
	if got := transposeChord("G", 1, true); got != "Ab" {
		t.Errorf("got %q, want Ab", got)
	}
}

func TestTransposeSlashChord(t *testing.T) {
	if got := transposeChord("G/B", 2, false); got != "A/C#" {
		t.Errorf("got %q, want A/C#", got)
	}
	if got := transposeChord("D/F#", 1, false); got != "D#/G" {
		t.Errorf("got %q, want D#/G", got)
	}
}

func TestTransposeNonChordUnchanged(t *testing.T) {
	for _, c := range []string{"", "N.C.", "%", "x4"} {
		if got := transposeChord(c, 3, false); got != c {
			t.Errorf("transpose(%q) = %q, want unchanged", c, got)
		}
	}
}

func TestTransposeKeyPicksFlats(t *testing.T) {
	// C up one semitone in a flat-leaning target spells with flats.
	if got := TransposeKey("C", 1); got != "Db" {
		t.Errorf("TransposeKey(C, 1) = %q, want Db", got)
	}
	// G up two -> A (sharp side).
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
