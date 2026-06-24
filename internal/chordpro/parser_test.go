package chordpro

import "testing"

func TestParseMetadataAndSections(t *testing.T) {
	src := `{title: Test Song}
{artist: The Testers}
{key: C}
{tempo: 120}

{start_of_verse: Verse 1}
[C]Hello [G]world
{end_of_verse}

{soc}
[F]Chorus [C]line
{eoc}
`
	song, err := ParseString(src)
	if err != nil {
		t.Fatal(err)
	}
	if song.Title != "Test Song" {
		t.Errorf("title = %q", song.Title)
	}
	if song.Artist != "The Testers" {
		t.Errorf("artist = %q", song.Artist)
	}
	if song.Key != "C" || song.Tempo != "120" {
		t.Errorf("key=%q tempo=%q", song.Key, song.Tempo)
	}
	if len(song.Sections) != 2 {
		t.Fatalf("want 2 sections, got %d", len(song.Sections))
	}
	if song.Sections[0].Kind != KindVerse || song.Sections[0].Label != "Verse 1" {
		t.Errorf("verse section = %+v", song.Sections[0])
	}
	if song.Sections[1].Kind != KindChorus {
		t.Errorf("want chorus, got kind %d", song.Sections[1].Kind)
	}
}

func TestParseSegments(t *testing.T) {
	segs := parseSegments("[C]Hello [G]world")
	if len(segs) != 2 {
		t.Fatalf("want 2 segments, got %d: %+v", len(segs), segs)
	}
	if segs[0].Chord != "C" || segs[0].Text != "Hello " {
		t.Errorf("seg0 = %+v", segs[0])
	}
	if segs[1].Chord != "G" || segs[1].Text != "world" {
		t.Errorf("seg1 = %+v", segs[1])
	}
}

func TestParseSegmentsLeadingText(t *testing.T) {
	segs := parseSegments("Oh [C]baby")
	if len(segs) != 2 {
		t.Fatalf("want 2 segments, got %d", len(segs))
	}
	if segs[0].Chord != "" || segs[0].Text != "Oh " {
		t.Errorf("leading seg = %+v", segs[0])
	}
}

func TestCommentBecomesLine(t *testing.T) {
	song, err := ParseString("{c: play softly}\n[C]la")
	if err != nil {
		t.Fatal(err)
	}
	if len(song.Sections) != 1 || len(song.Sections[0].Lines) != 2 {
		t.Fatalf("unexpected sections: %+v", song.Sections)
	}
	if song.Sections[0].Lines[0].Comment != "play softly" {
		t.Errorf("comment = %q", song.Sections[0].Lines[0].Comment)
	}
}

func TestParseDefine(t *testing.T) {
	d, ok := parseDefine("Am base-fret 1 frets x 0 2 2 1 0")
	if !ok {
		t.Fatal("parseDefine failed")
	}
	if d.Name != "Am" || d.BaseFret != 1 {
		t.Errorf("got name=%q baseFret=%d", d.Name, d.BaseFret)
	}
	want := []int{-1, 0, 2, 2, 1, 0}
	if len(d.Frets) != len(want) {
		t.Fatalf("frets = %v", d.Frets)
	}
	for i := range want {
		if d.Frets[i] != want[i] {
			t.Errorf("fret %d = %d, want %d", i, d.Frets[i], want[i])
		}
	}
	// base_fret variant and a fingers section that must be ignored.
	d2, ok := parseDefine("F base_fret 1 frets 1 3 3 2 1 1 fingers 1 3 4 2 1 1")
	if !ok || len(d2.Frets) != 6 || d2.Frets[0] != 1 {
		t.Errorf("define with fingers: %+v", d2)
	}
	if _, ok := parseDefine("X"); ok {
		t.Error("a define with no frets should fail")
	}
}

func TestParseStoresDefines(t *testing.T) {
	s, err := ParseString("{define: Am base-fret 1 frets x 0 2 2 1 0}\n[Am]hi\n")
	if err != nil {
		t.Fatal(err)
	}
	if d, ok := s.Defines["Am"]; !ok || len(d.Frets) != 6 {
		t.Errorf("define not stored: %+v", s.Defines)
	}
}

func TestBPMTimeSignatureTuning(t *testing.T) {
	s, _ := ParseString("{bpm: 140}\n{tempo: Allegro}\n{time_signature: 6/8}\n{tuning: D A D G B E}\n")
	if s.BPM != "140" || s.Tempo != "Allegro" {
		t.Errorf("bpm=%q tempo=%q", s.BPM, s.Tempo)
	}
	if got := s.TempoDisplay(); got != "140" {
		t.Errorf("TempoDisplay = %q, want 140", got)
	}
	if s.Time != "6/8" {
		t.Errorf("time = %q", s.Time)
	}
	if s.Tuning != "D A D G B E" {
		t.Errorf("tuning = %q", s.Tuning)
	}
}

func TestIntroOutroEnvironments(t *testing.T) {
	s, _ := ParseString("{start_of_intro}\n[Am]x\n{end_of_intro}\n{soo}\n[C]y\n{eoo}\n")
	if len(s.Sections) != 2 {
		t.Fatalf("want 2 sections, got %d", len(s.Sections))
	}
	if s.Sections[0].Kind != KindIntro || s.Sections[0].Label != "Intro" {
		t.Errorf("intro = %+v", s.Sections[0])
	}
	if s.Sections[1].Kind != KindOutro || s.Sections[1].Label != "Outro" {
		t.Errorf("outro = %+v", s.Sections[1])
	}
}

func TestBlankLinesKeptInEnvironment(t *testing.T) {
	s, _ := ParseString("{start_of_bridge}\n[G]a\n\n[C]b\n{end_of_bridge}\n")
	if len(s.Sections) != 1 {
		t.Fatalf("environment fragmented into %d sections", len(s.Sections))
	}
	lines := s.Sections[0].Lines
	if len(lines) != 3 || !lines[1].IsBlank() {
		t.Errorf("blank line not preserved inside environment: %+v", lines)
	}
}

func TestLooseBlankStillSplits(t *testing.T) {
	s, _ := ParseString("[G]a\n\n[C]b\n")
	if len(s.Sections) != 2 {
		t.Errorf("loose paragraphs should split on blank lines, got %d sections", len(s.Sections))
	}
}

func TestSectionLabelAttribute(t *testing.T) {
	cases := []struct {
		src  string
		want string
	}{
		{`{start_of_verse: label="Verse 1"}`, "Verse 1"}, // colon + double quote
		{`{start_of_verse label="Verse 2"}`, "Verse 2"},  // whitespace separator
		{`{start_of_verse: label='Verse 3'}`, "Verse 3"}, // single quote
		{`{start_of_verse: Bridge Riff}`, "Bridge Riff"}, // bare argument
		{`{start_of_verse}`, "Verse"},                    // defaults to "Verse"
	}
	for _, c := range cases {
		s, err := ParseString(c.src + "\n[G]x\n{end_of_verse}\n")
		if err != nil {
			t.Fatalf("%s: %v", c.src, err)
		}
		if len(s.Sections) == 0 {
			t.Fatalf("%s: no sections", c.src)
		}
		if got := s.Sections[0].Label; got != c.want {
			t.Errorf("%s -> label %q, want %q", c.src, got, c.want)
		}
	}
}

func TestSectionDefaultLabels(t *testing.T) {
	cases := []struct {
		dir  string
		want string
		kind SectionKind
	}{
		{"start_of_verse", "Verse", KindVerse},
		{"start_of_chorus", "Chorus", KindChorus},
		{"start_of_bridge", "Bridge", KindBridge},
		{"start_of_tab", "Tab", KindTab},
		{"start_of_section", "Section", KindOther},
	}
	for _, c := range cases {
		s, err := ParseString("{" + c.dir + "}\n[G]x\n{end_of_section}\n")
		if err != nil {
			t.Fatalf("%s: %v", c.dir, err)
		}
		if len(s.Sections) == 0 {
			t.Fatalf("%s: no sections", c.dir)
		}
		if got := s.Sections[0].Label; got != c.want {
			t.Errorf("%s -> label %q, want %q", c.dir, got, c.want)
		}
		if got := s.Sections[0].Kind; got != c.kind {
			t.Errorf("%s -> kind %d, want %d", c.dir, got, c.kind)
		}
	}
}

// Loose lyric lines (no explicit section directive) must not gain a heading.
func TestLooseLyricsUnlabelled(t *testing.T) {
	s, _ := ParseString("[G]just loose lyrics\n")
	if len(s.Sections) == 0 {
		t.Fatal("no sections")
	}
	if s.Sections[0].Label != "" {
		t.Errorf("loose lyrics got a label %q, want none", s.Sections[0].Label)
	}
}

func TestStartOfSection(t *testing.T) {
	s, _ := ParseString("{start_of_section: Intro}\n[G]riff\n{end_of_section}\n")
	if len(s.Sections) != 1 {
		t.Fatalf("want 1 section, got %d", len(s.Sections))
	}
	if s.Sections[0].Label != "Intro" || s.Sections[0].Kind != KindOther {
		t.Errorf("section = %+v", s.Sections[0])
	}
}

func TestParseDirectiveDoesNotSplitOnEquals(t *testing.T) {
	// '=' introduces an attribute value, not the name/value boundary.
	name, val, ok := parseDirective(`{start_of_chorus: label="A B"}`)
	if !ok || name != "start_of_chorus" || val != `label="A B"` {
		t.Errorf("parseDirective = (%q, %q, %v)", name, val, ok)
	}
}

func TestParseAnnotation(t *testing.T) {
	segs := parseSegments("[*Riff x2] [C]word [*N.C.]more")
	if segs[0].Annotation != "Riff x2" || segs[0].Chord != "" {
		t.Errorf("seg0 = %+v, want annotation %q", segs[0], "Riff x2")
	}
	if segs[1].Chord != "C" || segs[1].Annotation != "" {
		t.Errorf("seg1 = %+v, want chord C", segs[1])
	}
	// The annotation segment must report a marker so a chord row is rendered.
	if !(Line{Segments: segs}).HasMarkers() {
		t.Error("line with an annotation should HasMarkers()")
	}
}

func TestAnnotationNotTransposed(t *testing.T) {
	// "[*Coda]" must not transpose like a C-rooted chord (-> "Doda").
	s, _ := ParseString("[*Coda] [C]word\n")
	out := s.Transposed(2)
	seg := out.Sections[0].Lines[0].Segments[0]
	if seg.Annotation != "Coda" {
		t.Errorf("annotation transposed/mangled: %q", seg.Annotation)
	}
}

func TestCommentVariants(t *testing.T) {
	for _, d := range []string{"highlight", "comment_box", "cb"} {
		s, _ := ParseString("{" + d + ": Note here}\n[C]x\n")
		if len(s.Sections) == 0 || s.Sections[0].Lines[0].Comment != "Note here" {
			t.Errorf("{%s} did not produce a visible comment: %+v", d, s.Sections)
		}
	}
}
