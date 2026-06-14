package render

import (
	"strings"
	"testing"

	"chordpro-tui/internal/chordpro"

	"github.com/charmbracelet/lipgloss"
)

func TestApplyBackground(t *testing.T) {
	out := ApplyBackground("ab\ncd", 4, lipgloss.Color("#102030")) // 16,32,48
	for _, ln := range strings.Split(out, "\n") {
		if !strings.HasPrefix(ln, "\x1b[48;2;16;32;48m") {
			t.Errorf("line missing bg prefix: %q", ln)
		}
		if !strings.HasSuffix(ln, "\x1b[0m") {
			t.Errorf("line missing reset suffix: %q", ln)
		}
		if lipgloss.Width(ln) != 4 {
			t.Errorf("line width = %d, want 4 (padded)", lipgloss.Width(ln))
		}
	}
	// A reset inside the content re-asserts the background after it.
	tinted := ApplyBackground("\x1b[0mx", 2, lipgloss.Color("#000000"))
	if strings.Count(tinted, "\x1b[48;2;0;0;0m") < 2 {
		t.Errorf("background not re-asserted after inner reset: %q", tinted)
	}
	// An unparseable color is a no-op.
	if got := ApplyBackground("ab", 4, lipgloss.Color("nope")); got != "ab" {
		t.Errorf("bad color should be a no-op, got %q", got)
	}
}

func TestTidyBlanks(t *testing.T) {
	in := []string{"", "a", "", "", "b", "", ""}
	got := tidyBlanks(in)
	want := []string{"a", "", "b"}
	if len(got) != len(want) {
		t.Fatalf("tidyBlanks = %q, want %q", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("line %d = %q, want %q", i, got[i], want[i])
		}
	}
}

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
