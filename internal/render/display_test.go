package render

import (
	"strings"
	"testing"

	"github.com/monomadic/chordpro-tui/internal/chordpro"

	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

func mustParse(t *testing.T, src string) *chordpro.Song {
	t.Helper()
	s, err := chordpro.ParseString(src)
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestHideSectionTitles(t *testing.T) {
	song := mustParse(t, sampleSong)
	th := DefaultTheme()
	shown := RenderWith(song, 100, 30, th, RenderOpts{})
	if !strings.Contains(shown, "VERSE") {
		t.Fatalf("baseline should contain the VERSE label:\n%s", shown)
	}
	hidden := RenderWith(song, 100, 30, th, RenderOpts{HideSectionTitles: true})
	if strings.Contains(hidden, "VERSE") {
		t.Errorf("HideSectionTitles should drop the VERSE label:\n%s", hidden)
	}
	// The lyrics themselves must still be present.
	if !strings.Contains(hidden, "One little") {
		t.Errorf("lyrics were dropped along with the labels:\n%s", hidden)
	}
}

func TestCollapsePageTitleOnePlace(t *testing.T) {
	song := mustParse(t, sampleSong)
	th := DefaultTheme()
	stacked := RenderWith(song, 100, 30, th, RenderOpts{})
	// In the stacked header the title and the pills are on different rows.
	collapsed := RenderWith(song, 100, 30, th, RenderOpts{CollapsePageTitle: On})

	titleRow := func(out string) int {
		for i, ln := range strings.Split(out, "\n") {
			if strings.Contains(ln, "Sample") {
				return i
			}
		}
		return -1
	}
	pillRow := func(out string) int {
		for i, ln := range strings.Split(out, "\n") {
			if strings.Contains(ln, "KEY") {
				return i
			}
		}
		return -1
	}
	if titleRow(stacked) == pillRow(stacked) {
		t.Fatalf("expected title and pills on separate rows without collapse")
	}
	if r := titleRow(collapsed); r < 0 || r != pillRow(collapsed) {
		t.Errorf("collapse-page-title should put title and pills on one row (title=%d pill=%d)",
			titleRow(collapsed), pillRow(collapsed))
	}
}

func TestHideTitleAndInfoRemovesHeader(t *testing.T) {
	song := mustParse(t, sampleSong)
	th := DefaultTheme()
	out := RenderWith(song, 100, 30, th, RenderOpts{HideTitle: On, HideInfo: On})
	// The footer always echoes the song title/artist, so inspect only the body
	// above it: the header (title line + KEY/… pills) must be gone.
	lines := strings.Split(out, "\n")
	body := strings.Join(lines[:len(lines)-1], "\n")
	if strings.Contains(body, "Sample") || strings.Contains(body, "KEY") {
		t.Errorf("hiding both title and info should remove the whole header:\n%s", out)
	}
}

func TestSectionTitleGapAddsBlankAboveLabel(t *testing.T) {
	// Two verses so the second label has a block above it to be set off from.
	song := mustParse(t, "{title: T}\n{sov: One}\n[G]a\n{eov}\n{sov: Two}\n[C]b\n{eov}\n")
	th := DefaultTheme()
	withGap := buildBlocks(song, th, display{sectionTitleGap: true})
	without := buildBlocks(song, th, display{})
	// The gap adds a leading blank row to each labeled block, so blocks are taller.
	if len(withGap) != len(without) || len(withGap) < 1 {
		t.Fatalf("unexpected block counts: %d vs %d", len(withGap), len(without))
	}
	if withGap[0].height != without[0].height+1 {
		t.Errorf("section-title gap should add one row (got %d, base %d)",
			withGap[0].height, without[0].height)
	}
	if withGap[0].lines[0] != "" {
		t.Errorf("first line of a gapped block should be blank, got %q", withGap[0].lines[0])
	}
}

func TestAutoReducesToAvoidTruncation(t *testing.T) {
	// A song long enough to overflow a short screen with the roomy header, but
	// that fits once the header collapses to one line.
	song := mustParse(t, reclaimSong)
	th := DefaultTheme()
	const w = 60

	// Find a height where the full (roomy) render is truncated.
	h := 6
	for ; h < 40; h++ {
		if strings.Contains(RenderWith(song, w, h, th, RenderOpts{}), "▾") {
			continue
		}
		break
	}
	// Just below that fit height, the roomy layout truncates...
	h--
	if h < 6 {
		t.Skip("song too small to exercise the auto ladder")
	}
	roomy := RenderWith(song, w, h, th, RenderOpts{})
	if !strings.Contains(roomy, "▾") {
		t.Skipf("no truncation at h=%d to reduce", h)
	}
	// ...but with the header set to collapse on demand, auto reclaims the row.
	auto := RenderWith(song, w, h, th, RenderOpts{CollapsePageTitle: Auto, SectionTitleGap: Auto})
	if strings.Contains(auto, "▾") {
		t.Errorf("auto options should have reclaimed space to avoid truncation at h=%d:\n%s", h, auto)
	}
}

const tabSectionSong = `{title: Tabs}
{artist: A}

{sov: Verse}
[C]Hello there
{eov}

{sot: Riff}
e|--0--2--3--|
B|--1--------|
{eot}
`

func TestTabFoldOverridesCollapseTabs(t *testing.T) {
	song := mustParse(t, tabSectionSong)
	th := DefaultTheme()

	folded := RenderWith(song, 100, 30, th, RenderOpts{CollapseTabs: On})
	if strings.Contains(folded, "e|--0--2--3--|") {
		t.Errorf("CollapseTabs: On should fold the tab section:\n%s", folded)
	}
	// The 'T' key must be able to unfold what the config folded, not just fold.
	show, hide := false, true
	shown := RenderWith(song, 100, 30, th, RenderOpts{CollapseTabs: On, TabFold: &show})
	if !strings.Contains(shown, "e|--0--2--3--|") {
		t.Errorf("TabFold=false should unfold a config-folded tab section:\n%s", shown)
	}
	hidden := RenderWith(song, 100, 30, th, RenderOpts{TabFold: &hide})
	if strings.Contains(hidden, "e|--0--2--3--|") {
		t.Errorf("TabFold=true should fold the tab section:\n%s", hidden)
	}
}

func TestTabPanelIsAPaddedRectangle(t *testing.T) {
	song := mustParse(t, tabSectionSong)
	lines := tabPanel(song.Sections[1], DefaultTheme(), true)
	if len(lines) != 5 { // blank fill, RIFF title bar, two tab rows, blank fill
		t.Fatalf("want 5 panel rows, got %d: %q", len(lines), lines)
	}
	w := lipgloss.Width(lines[0])
	for i, l := range lines {
		if got := lipgloss.Width(l); got != w {
			t.Errorf("row %d width %d, want %d (panel must be a rectangle)", i, got, w)
		}
	}
	if !strings.Contains(stripANSI(lines[1]), "RIFF") {
		t.Errorf("section label should head the panel, got %q", lines[1])
	}
}

func TestSideSectionTitlesSaveARow(t *testing.T) {
	song := mustParse(t, "{title: T}\n{sov: Verse}\n[G]a\nb\n{eov}\n{soc}\n[C]c\n{eoc}\n")
	th := DefaultTheme()
	above := buildBlocks(song, th, display{})
	side := buildBlocks(song, th, display{sideLabels: true})
	if above[0].height != side[0].height+1 {
		t.Errorf("side label should save a row (above %d, side %d)", above[0].height, side[0].height)
	}
	// Label sits on the first lyric row (below its chords), right-aligned
	// against the body: " VERSE " for a margin of len("CHORUS")+1, so every
	// block's body starts together.
	if first := stripANSI(side[0].lines[0]); !strings.HasPrefix(first, strings.Repeat(" ", 9)+"G") {
		t.Errorf("chord row = %q, want it indented with no label", first)
	}
	if text := stripANSI(side[0].lines[1]); !strings.HasPrefix(text, " VERSE   a") {
		t.Errorf("lyric row = %q, want right-aligned VERSE label", text)
	}
	// The chorus bar also starts on the lyric row, leaving the chords above it.
	if chords := stripANSI(side[1].lines[0]); strings.Contains(chords, chorusBar) {
		t.Errorf("chorus chord row = %q, want no bar above the first lyric", chords)
	}
	if text := stripANSI(side[1].lines[1]); !strings.HasPrefix(text, "CHORUS "+chorusBar+"c") {
		t.Errorf("chorus lyric row = %q, want label then chorus bar", text)
	}
}

func TestSideSectionTitlesAutoFallsBack(t *testing.T) {
	song := mustParse(t, "{title: T}\n{sov: Longer label}\n[G]abcdefgh\n{eov}\n")
	th := DefaultTheme()
	// The label fits above the body but not beside it: auto falls back to above.
	out := RenderWith(song, 22, 12, th, RenderOpts{SideSectionTitles: Auto})
	if !strings.Contains(out, "LONGER LABEL") {
		t.Fatalf("label missing:\n%s", out)
	}
	for _, ln := range strings.Split(stripANSI(out), "\n") {
		if strings.Contains(ln, "LABEL") && strings.Contains(ln, "abc") {
			t.Errorf("label should not share a row with the body when it doesn't fit:\n%s", out)
		}
	}
}

func TestSuperQuality(t *testing.T) {
	cases := map[string]string{
		"A":      "A",
		"Am":     "Aᵐ",
		"F#m7b5": "F#ᵐ⁷ᵇ⁵",
		"Bbmaj7": "Bbᵐᵃʲ⁷",
		"Csus4":  "Cˢᵘˢ⁴",
		"D/F#":   "D/F#",
		"Am7/G":  "Aᵐ⁷/G",
		"G7#9":   "G7#9", // '#' has no superscript: leave the quality alone
		"N.C.":   "N.C.",
	}
	for in, want := range cases {
		if got := superQuality(in); got != want {
			t.Errorf("superQuality(%q) = %q, want %q", in, got, want)
		}
		if runeLen(superQuality(in)) != runeLen(in) {
			t.Errorf("superQuality(%q) changed the width", in)
		}
	}
}

func TestInlineChordsOneRowPerLine(t *testing.T) {
	song := mustParse(t, "{title: T}\n{sov: V}\n[G]Headin' down [D]south\n{eov}\n")
	th := DefaultTheme()
	stacked := buildBlocks(song, th, display{})
	inline := buildBlocks(song, th, display{inlineChords: true})
	if inline[0].height != stacked[0].height-1 {
		t.Fatalf("inline chords should drop the chord row (stacked %d, inline %d)",
			stacked[0].height, inline[0].height)
	}
	if got := stripANSI(inline[0].lines[len(inline[0].lines)-1]); !strings.Contains(got, "GHeadin' down Dsouth") {
		t.Errorf("inline row = %q", got)
	}
}

func TestInlineChordsAutoIsLastResort(t *testing.T) {
	_, steps := resolveDisplay(RenderOpts{HideTitle: Auto, SectionTitleGap: Auto, InlineChords: Auto})
	var d display
	for _, s := range steps[:len(steps)-1] {
		s(&d)
	}
	if d.inlineChords {
		t.Fatal("inline chords applied before the last step")
	}
	steps[len(steps)-1](&d)
	if !d.inlineChords {
		t.Fatal("last auto step should inline the chords")
	}
}

func TestPlainChordsDropBackground(t *testing.T) {
	old := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	defer lipgloss.SetColorProfile(old)
	song := mustParse(t, "{title: T}\n[G]a\n")
	th := DefaultTheme()
	chordRow := func(d display) string { return buildBlocks(song, th, d)[0].lines[0] }
	// The pill background is a 48;2 (truecolor background) SGR sequence.
	if pill := chordRow(display{}); !strings.Contains(pill, "48;") {
		t.Fatalf("chords should have a pill background by default: %q", pill)
	}
	if plain := chordRow(display{plainChords: true}); strings.Contains(plain, "48;") {
		t.Errorf("plain chords should have no background: %q", plain)
	}
}
