package chords

import "sort"

var standardTuning = [6]int{40, 45, 50, 55, 59, 64}
var noteNames = [12]string{"C", "C#", "D", "Eb", "E", "F", "F#", "G", "Ab", "A", "Bb", "B"}

// Identification describes exact pitch-class matches, with the actual bass
// indicated for inversions. Frets are absolute, -1 muted, 0 open.
type Identification struct {
	Names []string
	Notes []string
}

// Identify recognizes complete common chords without assuming omitted notes.
// Multiple names are retained because a pitch set need not have a unique root.
func Identify(frets [6]int) Identification {
	var out Identification
	mask, bass := 0, 1000
	for i, fret := range frets {
		if fret < 0 {
			continue
		}
		pitch := standardTuning[i] + fret
		mask |= 1 << (pitch % 12)
		if pitch < bass {
			bass = pitch
		}
		out.Notes = append(out.Notes, noteNames[pitch%12])
	}
	if mask == 0 {
		return out
	}
	patterns := []struct {
		suffix    string
		intervals []int
	}{
		{"", []int{0, 4, 7}}, {"m", []int{0, 3, 7}}, {"5", []int{0, 7}},
		{"dim", []int{0, 3, 6}}, {"aug", []int{0, 4, 8}},
		{"sus2", []int{0, 2, 7}}, {"sus4", []int{0, 5, 7}},
		{"7", []int{0, 4, 7, 10}}, {"maj7", []int{0, 4, 7, 11}}, {"m7", []int{0, 3, 7, 10}},
		{"m(maj7)", []int{0, 3, 7, 11}}, {"m7b5", []int{0, 3, 6, 10}}, {"dim7", []int{0, 3, 6, 9}},
		{"6", []int{0, 4, 7, 9}}, {"m6", []int{0, 3, 7, 9}},
		{"add9", []int{0, 2, 4, 7}}, {"madd9", []int{0, 2, 3, 7}},
		{"9", []int{0, 2, 4, 7, 10}}, {"maj9", []int{0, 2, 4, 7, 11}}, {"m9", []int{0, 2, 3, 7, 10}},
		{"7sus4", []int{0, 5, 7, 10}},
	}
	type match struct {
		name      string
		inversion bool
	}
	var matches []match
	for _, p := range patterns {
		for root := 0; root < 12; root++ {
			want := 0
			for _, interval := range p.intervals {
				want |= 1 << ((root + interval) % 12)
			}
			if mask != want {
				continue
			}
			name := noteNames[root] + p.suffix
			inversion := bass%12 != root
			if inversion {
				name += "/" + noteNames[bass%12]
			}
			matches = append(matches, match{name, inversion})
		}
	}
	sort.SliceStable(matches, func(i, j int) bool { return !matches[i].inversion && matches[j].inversion })
	for _, m := range matches {
		out.Names = append(out.Names, m.name)
	}
	return out
}
