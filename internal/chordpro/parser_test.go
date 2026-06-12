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
