package chords

import (
	"reflect"
	"testing"
)

func TestLookupCommonChords(t *testing.T) {
	tests := []struct {
		name      string
		wantFrets []int
		wantBase  int
	}{
		{"C", []int{-1, 3, 2, 0, 1, 0}, 1},
		{"Am", []int{-1, 0, 2, 2, 1, 0}, 1},
		{"G", []int{3, 2, 0, 0, 0, 3}, 1},
	}
	for _, tt := range tests {
		s, ok := Lookup(tt.name)
		if !ok {
			t.Fatalf("Lookup(%q): not found", tt.name)
		}
		if !reflect.DeepEqual(s.Frets, tt.wantFrets) {
			t.Errorf("Lookup(%q).Frets = %v, want %v", tt.name, s.Frets, tt.wantFrets)
		}
		if s.BaseFret != tt.wantBase {
			t.Errorf("Lookup(%q).BaseFret = %d, want %d", tt.name, s.BaseFret, tt.wantBase)
		}
	}
}

// TestLookupResolves checks that the name parser and suffix aliasing find a
// shape for a spread of realistic ChordPro tokens, including enharmonics,
// minor/extended suffixes, and slash chords.
func TestLookupResolves(t *testing.T) {
	resolve := []string{
		"Cmaj7", "Dm7", "G7", "C#m", "Bb", "Ab", "G/B", "Csus4",
		"Aadd9", "F#m7b5", "Em7", "D/F#", "Bbm", "F#", "Gsus2",
	}
	for _, n := range resolve {
		if _, ok := Lookup(n); !ok {
			t.Errorf("Lookup(%q): want a shape, got none", n)
		}
	}
}

func TestLookupRejectsNonChords(t *testing.T) {
	for _, n := range []string{"", "N.C.", "%", "x"} {
		if s, ok := Lookup(n); ok {
			t.Errorf("Lookup(%q): want no shape, got %v", n, s.Frets)
		}
	}
}

// TestSlashFallsBackToBase confirms that a slash chord whose exact bass isn't
// catalogued still resolves to the base chord's shape rather than failing.
func TestSlashFallsBackToBase(t *testing.T) {
	s, ok := Lookup("Cmaj7/G")
	if !ok {
		t.Fatal("Cmaj7/G: want a fallback shape, got none")
	}
	base, _ := Lookup("Cmaj7")
	if !reflect.DeepEqual(s.Frets, base.Frets) {
		t.Errorf("Cmaj7/G fell back to %v, want base Cmaj7 %v", s.Frets, base.Frets)
	}
}
