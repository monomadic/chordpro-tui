package chordpro

import "time"

// Song is a parsed ChordPro song: metadata plus an ordered list of sections.
type Song struct {
	Title    string
	Subtitle string
	Artist   string
	Composer string
	Album    string
	Key      string
	Capo     string
	Tempo    string
	Time     string // time signature, e.g. 4/4
	Year     string
	Duration time.Duration // total song length, for scroll-sync mode (0 if unknown)
	Sections []Section

	// TransposeBy records the semitone shift applied by Transposed (0 = none).
	TransposeBy int
}

// Meta returns the header "pills" worth displaying, in a stable order.
// Each entry is a (label, value) pair; empty values are skipped.
func (s Song) Meta() [][2]string {
	var out [][2]string
	add := func(label, val string) {
		if val != "" {
			out = append(out, [2]string{label, val})
		}
	}
	add("KEY", s.Key)
	add("CAPO", s.Capo)
	add("TEMPO", s.Tempo)
	add("TIME", s.Time)
	add("YEAR", s.Year)
	return out
}

// SectionKind classifies a block of the song for styling purposes.
type SectionKind int

const (
	KindVerse SectionKind = iota
	KindChorus
	KindBridge
	KindTab
	KindComment // a standalone {comment} not attached to lyrics
	KindOther
)

// Section is a contiguous block of lines (a verse, chorus, bridge, ...).
type Section struct {
	Kind  SectionKind
	Label string // optional display label, e.g. "Chorus", "Verse 1"
	Lines []Line
}

// Line is a single lyric line decomposed into chord/text segments, or a
// directive-driven line such as a comment.
type Line struct {
	// Comment, when non-empty, means this line is a {comment} annotation and
	// Segments is ignored.
	Comment string
	// Segments make up a lyric line. Each carries an optional chord that sits
	// at the start of its text.
	Segments []Segment
}

// IsBlank reports whether the line has no chords, text, or comment.
func (l Line) IsBlank() bool {
	if l.Comment != "" {
		return false
	}
	for _, s := range l.Segments {
		if s.Chord != "" || s.Text != "" {
			return false
		}
	}
	return true
}

// HasChords reports whether any segment carries a chord.
func (l Line) HasChords() bool {
	for _, s := range l.Segments {
		if s.Chord != "" {
			return true
		}
	}
	return false
}

// PlainText returns the line's lyric text with chords stripped.
func (l Line) PlainText() string {
	if l.Comment != "" {
		return l.Comment
	}
	var b []byte
	for _, s := range l.Segments {
		b = append(b, s.Text...)
	}
	return string(b)
}

// Segment is a chord followed by the text it applies to. Either may be empty.
type Segment struct {
	Chord string
	Text  string
}
