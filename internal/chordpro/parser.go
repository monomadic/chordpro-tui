package chordpro

import (
	"bufio"
	"io"
	"strconv"
	"strings"
	"time"
)

// Parse reads a ChordPro document and returns a structured Song.
//
// It understands the common directive set ({title}, {artist}, {key}, {capo},
// {tempo}, {comment}, chorus/verse/bridge/tab environments and their {soc}/{eoc}
// style abbreviations) plus inline [chord] markup. Unknown directives are
// ignored; lines outside any environment collect into verse sections.
func Parse(r io.Reader) (*Song, error) {
	song := &Song{}
	sc := bufio.NewScanner(r)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	// cur is the section currently being filled. A nil cur means we are between
	// sections and the next lyric line should open a fresh verse.
	var cur *Section
	flush := func() {
		if cur != nil && len(cur.Lines) > 0 {
			song.Sections = append(song.Sections, *cur)
		}
		cur = nil
	}
	ensure := func(kind SectionKind, label string) {
		if cur == nil {
			cur = &Section{Kind: kind, Label: label}
		}
	}
	// inEnv is true between an explicit {start_of_*} and its {end_of_*}; inside
	// such a block blank lines are kept rather than splitting the section.
	inEnv := false

	for sc.Scan() {
		raw := sc.Text()
		line := strings.TrimRight(raw, "\r\n")
		trimmed := strings.TrimSpace(line)

		// Whole-line comment in source (not a chord-sheet annotation).
		if strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Directive line: {name} or {name: value}. We only treat it as a
		// directive when the line is essentially just the brace expression.
		if name, val, ok := parseDirective(trimmed); ok {
			switch normDirective(name) {
			case "title", "t":
				song.Title = val
			case "subtitle", "st":
				song.Subtitle = val
			case "artist":
				song.Artist = val
			case "composer":
				song.Composer = val
			case "album":
				song.Album = val
			case "key":
				song.Key = val
			case "capo":
				song.Capo = val
			case "tempo":
				song.Tempo = val
			case "bpm":
				song.BPM = val
			case "time", "time_signature":
				song.Time = val
			case "year":
				song.Year = val
			case "tuning":
				song.Tuning = val
			case "duration", "length":
				song.Duration = parseDuration(val)
			case "define":
				if d, ok := parseDefine(val); ok {
					if song.Defines == nil {
						song.Defines = make(map[string]ChordDefinition)
					}
					song.Defines[d.Name] = d
				}

			case "start_of_chorus", "soc":
				flush()
				cur = &Section{Kind: KindChorus, Label: sectionLabel(val, "Chorus")}
				inEnv = true
			case "start_of_verse", "sov":
				flush()
				cur = &Section{Kind: KindVerse, Label: sectionLabel(val, "Verse")}
				inEnv = true
			case "start_of_bridge", "sob":
				flush()
				cur = &Section{Kind: KindBridge, Label: sectionLabel(val, "Bridge")}
				inEnv = true
			case "start_of_tab", "sot":
				flush()
				cur = &Section{Kind: KindTab, Label: sectionLabel(val, "Tab")}
				inEnv = true
			case "start_of_intro", "soi":
				flush()
				cur = &Section{Kind: KindIntro, Label: sectionLabel(val, "Intro")}
				inEnv = true
			case "start_of_outro", "soo":
				flush()
				cur = &Section{Kind: KindOutro, Label: sectionLabel(val, "Outro")}
				inEnv = true
			case "start_of_section", "sos":
				flush()
				cur = &Section{Kind: KindOther, Label: sectionLabel(val, "Section")}
				inEnv = true
			case "end_of_chorus", "eoc",
				"end_of_verse", "eov",
				"end_of_bridge", "eob",
				"end_of_tab", "eot",
				"end_of_intro", "eoi",
				"end_of_outro", "eoo",
				"end_of_section", "eos":
				flush()
				inEnv = false

			case "chorus":
				// Recall: re-insert the most recent preceding chorus's content.
				// {chorus: Label} (or label="…") re-labels the recalled copy;
				// a bare {chorus} keeps the original chorus's label.
				flush()
				if src := lastChorus(song.Sections); src != nil {
					rec := *src
					rec.Lines = append([]Line(nil), src.Lines...)
					rec.Label = sectionLabel(val, src.Label)
					song.Sections = append(song.Sections, rec)
				}

			case "comment", "c", "comment_italic", "ci",
				"comment_box", "cb", "highlight":
				ensure(KindVerse, "")
				cur.Lines = append(cur.Lines, Line{Comment: val})

			default:
				// Unknown / unsupported directive: ignore.
			}
			continue
		}

		// Blank line: inside an explicit environment keep it as spacing; loose
		// (un-bracketed) paragraphs are split into separate sections instead.
		if trimmed == "" {
			if inEnv && cur != nil {
				cur.Lines = append(cur.Lines, Line{})
			} else {
				flush()
			}
			continue
		}

		// Tab environment: keep raw text verbatim (no chord parsing).
		if cur != nil && cur.Kind == KindTab {
			cur.Lines = append(cur.Lines, Line{Segments: []Segment{{Text: line}}})
			continue
		}

		// Ordinary lyric line with inline chords.
		ensure(KindVerse, "")
		cur.Lines = append(cur.Lines, Line{Segments: parseSegments(line)})
	}
	flush()

	if err := sc.Err(); err != nil {
		return nil, err
	}
	return song, nil
}

// ParseString is a convenience wrapper around Parse.
func ParseString(s string) (*Song, error) {
	return Parse(strings.NewReader(s))
}

// parseDirective extracts a {name} or {name: value} directive from a line that
// consists solely of that brace expression. Returns ok=false otherwise.
//
// The name is separated from its argument by a colon and/or whitespace
// (ChordPro §5). '=' is NOT a separator: it introduces attribute values such as
// label="Verse 1", so splitting the name on it would mangle attribute-form
// directives — the value is returned whole for the caller to parse (see
// parseAttrs).
func parseDirective(line string) (name, value string, ok bool) {
	if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
		return "", "", false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, "{"), "}")
	if strings.ContainsAny(inner, "{}") {
		// Nested braces: not a clean directive line.
		return "", "", false
	}
	if i := strings.IndexAny(inner, ": \t"); i >= 0 {
		name = strings.TrimSpace(inner[:i])
		rest := strings.TrimLeft(inner[i:], " \t")
		value = strings.TrimSpace(strings.TrimPrefix(rest, ":"))
	} else {
		name = strings.TrimSpace(inner)
	}
	if name == "" {
		return "", "", false
	}
	return name, value, true
}

// lastChorus returns a pointer to the most recently parsed chorus section, or
// nil if none has been seen yet. Used by {chorus} recall.
func lastChorus(secs []Section) *Section {
	for i := len(secs) - 1; i >= 0; i-- {
		if secs[i].Kind == KindChorus {
			return &secs[i]
		}
	}
	return nil
}

// sectionLabel resolves the display label for a section start directive. It
// accepts the attribute form ({start_of_verse: label="Verse 1"}) and the
// bare-argument form ({start_of_verse: Verse 1}); an empty argument falls back
// to def.
func sectionLabel(val, def string) string {
	if attrs := parseAttrs(val); attrs != nil {
		if l := strings.TrimSpace(attrs["label"]); l != "" {
			return l
		}
		return def
	}
	if v := strings.TrimSpace(val); v != "" {
		return v
	}
	return def
}

// parseAttrs parses HTML-style attributes — key="value", key='value', or bare
// key=value — from a directive argument (ChordPro §5). It returns nil when the
// argument is not in attribute form (e.g. a bare "Verse 1"), so callers can
// treat the whole argument as a single positional value instead.
func parseAttrs(s string) map[string]string {
	s = strings.TrimSpace(s)
	if !strings.Contains(s, "=") {
		return nil
	}
	attrs := map[string]string{}
	i, n := 0, len(s)
	for i < n {
		for i < n && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= n {
			break
		}
		start := i
		for i < n && s[i] != '=' && s[i] != ' ' && s[i] != '\t' {
			i++
		}
		key := s[start:i]
		if key == "" || i >= n || s[i] != '=' {
			return nil // not a clean attribute list; treat as a bare argument
		}
		i++ // consume '='
		var val string
		if i < n && (s[i] == '"' || s[i] == '\'') {
			q := s[i]
			i++
			vstart := i
			for i < n && s[i] != q {
				i++
			}
			val = s[vstart:i]
			if i < n {
				i++ // consume the closing quote
			}
		} else {
			vstart := i
			for i < n && s[i] != ' ' && s[i] != '\t' {
				i++
			}
			val = s[vstart:i]
		}
		attrs[key] = val
	}
	if len(attrs) == 0 {
		return nil
	}
	return attrs
}

// parseDefine parses a {define} body such as
// "Am base-fret 1 frets x 0 2 2 1 0" into a ChordDefinition. It understands the
// "base-fret"/"base_fret" and "frets" keywords; a "fingers" section (and any
// other trailing tokens) is ignored. Returns ok=false if no chord name or fret
// list is found.
func parseDefine(val string) (ChordDefinition, bool) {
	fields := strings.Fields(val)
	if len(fields) == 0 {
		return ChordDefinition{}, false
	}
	d := ChordDefinition{Name: fields[0], BaseFret: 1}
	i := 1
	for i < len(fields) {
		switch strings.ToLower(fields[i]) {
		case "base-fret", "base_fret", "basefret":
			if i+1 < len(fields) {
				if n, err := strconv.Atoi(fields[i+1]); err == nil {
					d.BaseFret = n
				}
				i += 2
				continue
			}
		case "frets":
			i++
			for i < len(fields) {
				tok := fields[i]
				if strings.EqualFold(tok, "fingers") {
					break
				}
				switch tok {
				case "x", "X", "N", "n":
					d.Frets = append(d.Frets, -1)
				default:
					if n, err := strconv.Atoi(tok); err == nil {
						d.Frets = append(d.Frets, n)
					} else {
						d.Frets = append(d.Frets, -1)
					}
				}
				i++
			}
			continue
		}
		i++
	}
	if len(d.Frets) == 0 {
		return ChordDefinition{}, false
	}
	return d, true
}

// parseSegments splits a lyric line into chord/text segments. Text before the
// first chord becomes a leading chord-less segment. A bracket whose first
// character is '*' is an annotation ([*Riff x2]): the '*' is dropped and the
// rest is carried verbatim in the chord position, never parsed or transposed.
func parseSegments(line string) []Segment {
	var segs []Segment
	var text strings.Builder
	pendingChord := ""
	pendingAnnot := ""

	flushText := func() {
		// Emit a segment only when it carries a marker or some text; this avoids
		// a spurious empty segment when a line begins with a chord.
		if text.Len() > 0 || pendingChord != "" || pendingAnnot != "" {
			segs = append(segs, Segment{Chord: pendingChord, Annotation: pendingAnnot, Text: text.String()})
		}
		pendingChord = ""
		pendingAnnot = ""
		text.Reset()
	}

	i := 0
	for i < len(line) {
		c := line[i]
		switch c {
		case '[':
			end := strings.IndexByte(line[i:], ']')
			if end < 0 {
				// Unterminated bracket: treat literally.
				text.WriteByte(c)
				i++
				continue
			}
			// Close out the text accumulated for the previous marker, then start
			// a new segment headed by this chord or annotation.
			flushText()
			inner := strings.TrimSpace(line[i+1 : i+end])
			if strings.HasPrefix(inner, "*") {
				pendingAnnot = strings.TrimSpace(inner[1:])
			} else {
				pendingChord = inner
			}
			i += end + 1
		case ']':
			// Stray closing bracket: literal.
			text.WriteByte(c)
			i++
		default:
			text.WriteByte(c)
			i++
		}
	}
	flushText()
	return segs
}

// parseDuration accepts "mm:ss", "h:mm:ss", or a bare seconds count.
func parseDuration(v string) time.Duration {
	v = strings.TrimSpace(v)
	if v == "" {
		return 0
	}
	if !strings.Contains(v, ":") {
		if secs, err := strconv.Atoi(v); err == nil {
			return time.Duration(secs) * time.Second
		}
		return 0
	}
	parts := strings.Split(v, ":")
	total := 0
	for _, p := range parts {
		n, err := strconv.Atoi(strings.TrimSpace(p))
		if err != nil {
			return 0
		}
		total = total*60 + n
	}
	return time.Duration(total) * time.Second
}

func normDirective(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
