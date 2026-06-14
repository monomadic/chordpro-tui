// Package chords resolves ChordPro chord tokens (e.g. "Am7", "C#", "G/B") into
// concrete guitar fingerings, drawn from an embedded copy of the open
// tombatossals/chords-db dataset (MIT). It is pure data — no rendering, no
// terminal concerns — so it can be unit-tested in isolation.
package chords

import (
	_ "embed"
	"encoding/json"
	"strings"
	"sync"
)

//go:embed guitar.json
var guitarJSON []byte

// Shape is a single playable fingering for a chord: one entry per string from
// the low E (index 0) to the high e (index 5). A fret value of -1 means the
// string is muted, 0 means played open, and any positive n means the n-th fret
// counting down from BaseFret (so n is absolute when BaseFret == 1).
type Shape struct {
	Name     string // the chord name as queried, e.g. "Am7"
	Frets    []int  // length 6, low→high string
	Fingers  []int  // length 6, fretting finger per string (0 = none)
	BaseFret int    // fret number of the topmost displayed row (1 = at the nut)
	Barres   []int  // fret rows that are barred, in the same coordinates as Frets
}

// position / chordEntry / database mirror the on-disk JSON shape just closely
// enough to decode it.
type position struct {
	Frets    []int `json:"frets"`
	Fingers  []int `json:"fingers"`
	BaseFret int   `json:"baseFret"`
	Barres   []int `json:"barres"`
}

type chordEntry struct {
	Key       string     `json:"key"`
	Suffix    string     `json:"suffix"`
	Positions []position `json:"positions"`
}

type database struct {
	Suffixes []string                `json:"suffixes"`
	Chords   map[string][]chordEntry `json:"chords"`
}

var (
	dbOnce   sync.Once
	db       database
	suffixOK map[string]bool
)

func load() {
	dbOnce.Do(func() {
		_ = json.Unmarshal(guitarJSON, &db)
		suffixOK = make(map[string]bool, len(db.Suffixes))
		for _, s := range db.Suffixes {
			suffixOK[s] = true
		}
	})
}

// dbKeyForPC maps a chromatic pitch class (0=C..11=B) to the key used by the
// chords object. Note the dataset spells the black keys as Csharp/Fsharp but
// Eb/Ab/Bb, and uses "sharp" rather than the "#" symbol for its map keys.
var dbKeyForPC = [12]string{
	"C", "Csharp", "D", "Eb", "E", "F", "Fsharp", "G", "Ab", "A", "Bb", "B",
}

// naturals is the semitone offset of each natural note from C.
var naturals = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// aliasSuffix maps common ChordPro suffix spellings that the dataset does not
// list verbatim onto ones it does. Suffixes already present in the dataset
// (m7, maj7, sus2, add9, slash forms like /G and m/B, …) are used as-is and
// never reach this table.
var aliasSuffix = map[string]string{
	"":     "major",
	"maj":  "major",
	"M":    "major",
	"m":    "minor",
	"min":  "minor",
	"-":    "minor",
	"+":    "aug",
	"aug5": "aug",
	"sus4": "sus4",
	"7sus": "7sus4",
	"maj7": "maj7",
}

// Lookup resolves a chord token to its easiest fingering (the dataset lists
// positions roughly from open shapes upward, so the first is the friendliest).
// It returns ok=false for tokens that aren't recognisable chords, such as
// "N.C." or a bare rest.
func Lookup(name string) (Shape, bool) {
	load()
	name = strings.TrimSpace(name)
	pc, rest, ok := splitRoot(name)
	if !ok {
		return Shape{}, false
	}
	entries, ok := db.Chords[dbKeyForPC[pc]]
	if !ok {
		return Shape{}, false
	}
	for _, suffix := range candidateSuffixes(rest) {
		for _, e := range entries {
			if e.Suffix == suffix && len(e.Positions) > 0 {
				p := e.Positions[0]
				return Shape{
					Name:     name,
					Frets:    p.Frets,
					Fingers:  p.Fingers,
					BaseFret: p.BaseFret,
					Barres:   p.Barres,
				}, true
			}
		}
	}
	return Shape{}, false
}

// candidateSuffixes returns the dataset suffixes to try for a token's suffix
// part, in priority order: the suffix as written, a known alias, and — for a
// slash chord whose exact bass isn't catalogued — the same chord without its
// bass note, so we still show a usable shape.
func candidateSuffixes(rest string) []string {
	var out []string
	add := func(s string) {
		if suffixOK[s] {
			out = append(out, s)
		}
	}
	add(rest)
	if a, ok := aliasSuffix[rest]; ok {
		add(a)
	}
	if i := strings.IndexByte(rest, '/'); i >= 0 {
		base := rest[:i]
		add(base)
		if a, ok := aliasSuffix[base]; ok {
			add(a)
		}
	}
	return out
}

// splitRoot peels the leading root note off a chord token, returning its pitch
// class (0..11) and the remaining suffix text. It reports ok=false when the
// token does not begin with a note letter.
func splitRoot(s string) (pc int, rest string, ok bool) {
	if s == "" {
		return 0, "", false
	}
	base, ok := naturals[s[0]]
	if !ok {
		return 0, "", false
	}
	n := 1
	if len(s) > 1 {
		switch s[1] {
		case '#':
			base++
			n = 2
		case 'b':
			base--
			n = 2
		}
	}
	return ((base % 12) + 12) % 12, s[n:], true
}
