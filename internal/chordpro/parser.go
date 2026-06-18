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
				cur = &Section{Kind: KindChorus, Label: orDefault(val, "Chorus")}
				inEnv = true
			case "start_of_verse", "sov":
				flush()
				cur = &Section{Kind: KindVerse, Label: val}
				inEnv = true
			case "start_of_bridge", "sob":
				flush()
				cur = &Section{Kind: KindBridge, Label: orDefault(val, "Bridge")}
				inEnv = true
			case "start_of_tab", "sot":
				flush()
				cur = &Section{Kind: KindTab, Label: orDefault(val, "Tab")}
				inEnv = true
			case "start_of_intro", "soi":
				flush()
				cur = &Section{Kind: KindIntro, Label: orDefault(val, "Intro")}
				inEnv = true
			case "start_of_outro", "soo":
				flush()
				cur = &Section{Kind: KindOutro, Label: orDefault(val, "Outro")}
				inEnv = true
			case "end_of_chorus", "eoc",
				"end_of_verse", "eov",
				"end_of_bridge", "eob",
				"end_of_tab", "eot",
				"end_of_intro", "eoi",
				"end_of_outro", "eoo":
				flush()
				inEnv = false

			case "comment", "c", "comment_italic", "ci":
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
func parseDirective(line string) (name, value string, ok bool) {
	if !strings.HasPrefix(line, "{") || !strings.HasSuffix(line, "}") {
		return "", "", false
	}
	inner := strings.TrimSuffix(strings.TrimPrefix(line, "{"), "}")
	if strings.ContainsAny(inner, "{}") {
		// Nested braces: not a clean directive line.
		return "", "", false
	}
	if i := strings.IndexAny(inner, ":="); i >= 0 {
		name = strings.TrimSpace(inner[:i])
		value = strings.TrimSpace(inner[i+1:])
	} else {
		name = strings.TrimSpace(inner)
	}
	if name == "" {
		return "", "", false
	}
	return name, value, true
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
// first chord becomes a leading chord-less segment.
func parseSegments(line string) []Segment {
	var segs []Segment
	var text strings.Builder
	pendingChord := ""

	flushText := func() {
		// Emit a segment only when it carries a chord or some text; this avoids
		// a spurious empty segment when a line begins with a chord.
		if text.Len() > 0 || pendingChord != "" {
			segs = append(segs, Segment{Chord: pendingChord, Text: text.String()})
		}
		pendingChord = ""
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
			// Close out the text accumulated for the previous chord, then start
			// a new segment headed by this chord.
			flushText()
			pendingChord = strings.TrimSpace(line[i+1 : i+end])
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

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return v
}
