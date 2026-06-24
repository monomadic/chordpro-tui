package chordpro

import "strings"

// noteNames is the fixed enharmonic spelling used for every pitch class
// (0=C..11=B), chosen for the form most commonly seen on lead sheets rather
// than the strictly key-correct one. Only E♭ and B♭ are flats; C♯, F♯ and G♯
// are sharps. Spelling therefore does not depend on the key.
var noteNames = [12]string{
	"C", "C#", "D", "Eb", "E", "F", "F#", "G", "G#", "A", "Bb", "B",
}

// naturals is the semitone offset of each natural note from C.
var naturals = map[byte]int{'C': 0, 'D': 2, 'E': 4, 'F': 5, 'G': 7, 'A': 9, 'B': 11}

// Transposed returns a deep copy of the song with every chord (and its key)
// shifted by n semitones. n may be any integer. The copy records n in
// TransposeBy.
func (s *Song) Transposed(n int) *Song {
	out := *s // shallow copy of scalar metadata
	out.TransposeBy = n
	out.Key = TransposeKey(s.Key, n)

	out.Sections = make([]Section, len(s.Sections))
	for i, sec := range s.Sections {
		nsec := sec
		nsec.Lines = make([]Line, len(sec.Lines))
		for j, ln := range sec.Lines {
			nln := ln
			if len(ln.Segments) > 0 {
				nln.Segments = make([]Segment, len(ln.Segments))
				for k, seg := range ln.Segments {
					seg.Chord = transposeChord(seg.Chord, n)
					nln.Segments[k] = seg
				}
			}
			nsec.Lines[j] = nln
		}
		out.Sections[i] = nsec
	}
	return &out
}

// SignedSemitones formats a transpose offset with an explicit sign, e.g.
// +3 or -2.
func SignedSemitones(n int) string { return signedSemitones(n) }

// AlternateTuningTitle appends the transpose offset to a song title, e.g.
// "Stolen Car" with n=1 → "Stolen Car (Alternate Tuning: +1)".
func AlternateTuningTitle(title string, n int) string {
	return strings.TrimSpace(title + " (Alternate Tuning: " + signedSemitones(n) + ")")
}

// TransposeSource rewrites raw ChordPro text: every inline [chord] and the
// {key} directive are shifted by n semitones, and the song title is replaced
// with newTitle. Everything else — comments, {define}s, unknown directives,
// tab/grid blocks, [*annotations], blank lines and formatting — is preserved
// verbatim, so a saved copy stays faithful to the original. If the source has
// no {title} directive, one is prepended.
func TransposeSource(src string, n int, newTitle string) string {
	lines := strings.Split(strings.ReplaceAll(src, "\r\n", "\n"), "\n")
	out := make([]string, 0, len(lines)+1)
	inVerbatim := false // inside a tab/grid block: no chord parsing
	titleSeen := false
	for _, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if name, val, ok := parseDirective(trimmed); ok {
			switch normDirective(name) {
			case "title", "t":
				titleSeen = true
				out = append(out, "{title: "+newTitle+"}")
				continue
			case "key":
				out = append(out, "{key: "+TransposeKey(val, n)+"}")
				continue
			case "start_of_tab", "sot", "start_of_grid", "sog":
				inVerbatim = true
			case "end_of_tab", "eot", "end_of_grid", "eog":
				inVerbatim = false
			}
			out = append(out, raw)
			continue
		}
		if inVerbatim || strings.HasPrefix(trimmed, "#") {
			out = append(out, raw) // verbatim block or source remark
			continue
		}
		out = append(out, transposeLine(raw, n))
	}
	if !titleSeen && strings.TrimSpace(newTitle) != "" {
		out = append([]string{"{title: " + newTitle + "}"}, out...)
	}
	return strings.Join(out, "\n")
}

// transposeLine shifts every [chord] in a lyric line by n semitones, leaving
// the surrounding text, [*annotations] and empty brackets untouched.
func transposeLine(line string, n int) string {
	if !strings.ContainsRune(line, '[') {
		return line
	}
	var b strings.Builder
	i := 0
	for i < len(line) {
		if line[i] == '[' {
			end := strings.IndexByte(line[i:], ']')
			if end < 0 {
				b.WriteString(line[i:]) // unterminated: leave the rest as-is
				break
			}
			inner := strings.TrimSpace(line[i+1 : i+end])
			if inner == "" || strings.HasPrefix(inner, "*") {
				b.WriteString(line[i : i+end+1]) // empty or annotation: verbatim
			} else {
				b.WriteByte('[')
				b.WriteString(transposeChord(inner, n))
				b.WriteByte(']')
			}
			i += end + 1
			continue
		}
		b.WriteByte(line[i])
		i++
	}
	return b.String()
}

// TransposeKey shifts a key name (e.g. "G", "Am", "Bb") by n semitones,
// preserving a trailing minor "m".
func TransposeKey(key string, n int) string {
	root, minor, ok := splitKey(key)
	if !ok {
		return key
	}
	name, ok := transposeRoot(root, n)
	if !ok {
		return key
	}
	return name + minorSuffix(minor)
}

// transposeChord shifts a single chord token, including any slash-bass note.
func transposeChord(chord string, n int) string {
	if chord == "" {
		return chord
	}
	// Slash chords: transpose both sides.
	if i := strings.IndexByte(chord, '/'); i >= 0 {
		return transposeChord(chord[:i], n) + "/" + transposeChord(chord[i+1:], n)
	}

	rootLen := rootLength(chord)
	if rootLen == 0 {
		return chord // not a recognisable chord (e.g. "N.C.")
	}
	name, ok := transposeRoot(chord[:rootLen], n)
	if !ok {
		return chord
	}
	return name + chord[rootLen:]
}

// transposeRoot maps a root spelling to its shifted spelling via noteNames.
func transposeRoot(root string, n int) (string, bool) {
	pc, ok := rootPC(root)
	if !ok {
		return "", false
	}
	return noteNames[mod12(pc+n)], true
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
