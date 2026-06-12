package render

import (
	"strings"
	"testing"

	"chordpro-tui/internal/chordpro"
)

func TestAlignChords(t *testing.T) {
	segs := []chordpro.Segment{
		{Chord: "C", Text: "Hello "},
		{Chord: "G", Text: "world"},
	}
	chord, lyric := alignChords(segs)
	chord = strings.TrimRight(chord, " ")
	// "C" sits above the H of Hello; "G" sits above the w of world (index 6).
	if lyric != "Hello world" {
		t.Errorf("lyric = %q", lyric)
	}
	if chord != "C     G" {
		t.Errorf("chord = %q (len %d)", chord, len(chord))
	}
}

func TestAlignChordOverhang(t *testing.T) {
	// A long chord over a short syllable must push the next syllable right.
	segs := []chordpro.Segment{
		{Chord: "Cmaj7", Text: "I "},
		{Chord: "G", Text: "go"},
	}
	chord, lyric := alignChords(segs)
	if !strings.HasPrefix(lyric, "I ") {
		t.Errorf("lyric = %q", lyric)
	}
	// The G chord should not collide with Cmaj7.
	if strings.Contains(chord, "Cmaj7G") {
		t.Errorf("chords collided: %q", chord)
	}
}

func TestRenderFitsHeight(t *testing.T) {
	song, err := chordpro.ParseString(sampleSong)
	if err != nil {
		t.Fatal(err)
	}
	const w, h = 120, 30
	out := Render(song, w, h, DefaultTheme())
	if got := strings.Count(out, "\n") + 1; got > h {
		t.Errorf("render produced %d lines, exceeds height %d", got, h)
	}
}

const sampleSong = `{title: Sample}
{artist: Tester}
{key: G}
{tempo: 90}

{sov: Verse}
[G]One little [C]line here
[D]Two little [G]lines here
{eov}

{soc}
[C]Chorus goes [G]here now
{eoc}
`

func TestRenderLongNoPanic(t *testing.T) {
	song, err := chordpro.ParseString(sampleSong)
	if err != nil {
		t.Fatal(err)
	}
	lines := RenderLong(song, 80, DefaultTheme())
	if len(lines) == 0 {
		t.Fatal("RenderLong returned no lines")
	}
}
