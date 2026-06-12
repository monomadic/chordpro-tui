package chordpro

import "strings"

var (
	sharpNames = []string{"C", "C#", "D", "D#", "E", "F", "F#", "G", "G#", "A", "A#", "B"}
	flatNames  = []string{"C", "Db", "D", "Eb", "E", "F", "Gb", "G", "Ab", "A", "Bb", "B"}
	// semitone offset of each natural note from C
	naturals = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}
)

// pcToFlat encodes, per pitch class (0=C..11=B), whether the conventional major
// key at that pitch is spelled with flats. This drives enharmonic choices so a
// transpose into a black key lands on the usual spelling (Db not C#, Ab not G#),
// while F# (6 sharps) wins over Gb.
var pcToFlat = [12]bool{
	false, // C
	true,  // Db
	false, // D
	true,  // Eb
	false, // E
	true,  // F  (1 flat)
	false, // F#
	false, // G
	true,  // Ab
	false, // A
	true,  // Bb
	false, // B
}

// Transposed returns a deep copy of the song with every chord (and its key)
// shifted by n semitones. n may be any integer; the spelling (sharps vs flats)
// follows the resulting key signature. The copy records n in TransposeBy.
func (s *Song) Transposed(n int) *Song {
	out := *s // shallow copy of scalar metadata
	out.TransposeBy = n
	if n%12 == 0 && n != 0 {
		// A whole number of octaves: chords are unchanged, but still record it.
	}
	out.Key = TransposeKey(s.Key, n)
	flats := keyPrefersFlats(out.Key)

	out.Sections = make([]Section, len(s.Sections))
	for i, sec := range s.Sections {
		nsec := sec
		nsec.Lines = make([]Line, len(sec.Lines))
		for j, ln := range sec.Lines {
			nln := ln
			if len(ln.Segments) > 0 {
				nln.Segments = make([]Segment, len(ln.Segments))
				for k, seg := range ln.Segments {
					seg.Chord = transposeChord(seg.Chord, n, flats)
					nln.Segments[k] = seg
				}
			}
			nsec.Lines[j] = nln
		}
		out.Sections[i] = nsec
	}
	return &out
}

// TransposeKey shifts a key name (e.g. "G", "Am", "Bb") by n semitones,
// preserving a trailing minor "m" and choosing the conventional spelling.
func TransposeKey(key string, n int) string {
	root, minor, ok := splitKey(key)
	if !ok {
		return key
	}
	pc, ok := rootPC(root)
	if !ok {
		return key
	}
	np := mod12(pc + n)
	flats := keyFlats(np, minor)
	if flats {
		return flatNames[np] + minorSuffix(minor)
	}
	return sharpNames[np] + minorSuffix(minor)
}

// transposeChord shifts a single chord token, including any slash-bass note.
func transposeChord(chord string, n int, flats bool) string {
	if chord == "" {
		return chord
	}
	// Slash chords: transpose both sides.
	if i := strings.IndexByte(chord, '/'); i >= 0 {
		left := transposeChord(chord[:i], n, flats)
		right := transposeChord(chord[i+1:], n, flats)
		return left + "/" + right
	}

	rootLen := rootLength(chord)
	if rootLen == 0 {
		return chord // not a recognisable chord (e.g. "N.C.")
	}
	root, ok := transposeRoot(chord[:rootLen], n, flats)
	if !ok {
		return chord
	}
	return root + chord[rootLen:]
}

// rootLength returns the length of the leading root note (1 or 2 bytes), or 0
// if the token does not start with a note letter.
func rootLength(s string) int {
	if s == "" {
		return 0
	}
	if _, ok := naturals[s[0]]; !ok {
		return 0
	}
	if len(s) > 1 && (s[1] == '#' || s[1] == 'b') {
		return 2
	}
	return 1
}

// transposeRoot maps a root spelling to its shifted spelling.
func transposeRoot(root string, n int, flats bool) (string, bool) {
	if root == "" {
		return "", false
	}
	base, ok := naturals[root[0]]
	if !ok {
		return "", false
	}
	if len(root) > 1 {
		switch root[1] {
		case '#':
			base++
		case 'b':
			base--
		}
	}
	idx := ((base+n)%12 + 12) % 12
	if flats {
		return flatNames[idx], true
	}
	return sharpNames[idx], true
}

// keyPrefersFlats reports whether a key signature is conventionally written
// with flats, used to pick chord spellings within that key.
func keyPrefersFlats(key string) bool {
	root, minor, ok := splitKey(key)
	if !ok {
		return false
	}
	pc, ok := rootPC(root)
	if !ok {
		return false
	}
	return keyFlats(pc, minor)
}

// keyFlats reports whether the key at pitch class pc (relative major for minor
// keys) is spelled with flats.
func keyFlats(pc int, minor bool) bool {
	if minor {
		pc = mod12(pc + 3) // relative major
	}
	return pcToFlat[pc]
}

// rootPC returns the chromatic pitch class (0..11) of a root spelling.
func rootPC(root string) (int, bool) {
	if root == "" {
		return 0, false
	}
	base, ok := naturals[root[0]]
	if !ok {
		return 0, false
	}
	if len(root) > 1 {
		switch root[1] {
		case '#':
			base++
		case 'b':
			base--
		}
	}
	return mod12(base), true
}

// splitKey separates a key into its root and minor flag.
func splitKey(key string) (root string, minor, ok bool) {
	key = strings.TrimSpace(key)
	if key == "" {
		return "", false, false
	}
	if strings.HasSuffix(key, "m") && !strings.HasSuffix(key, "dim") {
		return strings.TrimSuffix(key, "m"), true, true
	}
	return key, false, true
}

func minorSuffix(minor bool) string {
	if minor {
		return "m"
	}
	return ""
}

func mod12(n int) int { return ((n % 12) + 12) % 12 }
