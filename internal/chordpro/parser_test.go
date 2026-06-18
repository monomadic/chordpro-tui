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
