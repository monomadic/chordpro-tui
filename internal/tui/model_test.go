package tui

import (
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"chordpro-tui/internal/chordpro"

	tea "github.com/charmbracelet/bubbletea"
)

const song = `{title: T}
{artist: A}
{key: C}
{tempo: 90}
{duration: 3:00}

{sov: Verse}
[C]One [G]two
{eov}
`

var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string { return ansiRE.ReplaceAllString(s, "") }
func timeNow() time.Time        { return time.Now() }

func mustModel(t *testing.T) Model {
	t.Helper()
	s, err := chordpro.ParseString(song)
	if err != nil {
		t.Fatal(err)
	}
	return New(s, Options{})
}

// resize feeds a WindowSizeMsg and returns the updated model.
func resize(m Model, w, h int) Model {
	nm, _ := m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	return nm.(Model)
}

func TestResizeReflowsAndFits(t *testing.T) {
	m := mustModel(t)
	for _, dim := range [][2]int{{80, 24}, {120, 40}, {40, 60}, {200, 20}} {
		m = resize(m, dim[0], dim[1])
		out := m.View()
		if got := strings.Count(out, "\n") + 1; got > dim[1] {
			t.Errorf("at %dx%d: %d lines exceeds height", dim[0], dim[1], got)
		}
	}
}

func key(s string) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)} }

func TestModeCycleStaysWithinHeight(t *testing.T) {
	m := resize(mustModel(t), 80, 24)
	seen := map[mode]bool{}
	for i := 0; i < 4; i++ { // fit -> scroll -> sync -> fit ...
		seen[m.mode] = true
		out := m.View()
		if lines := strings.Count(out, "\n") + 1; lines > 24 {
			t.Errorf("mode %d: %d lines exceeds height 24", m.mode, lines)
		}
		nm, _ := m.handleKey(key("v"))
		m = nm.(Model)
	}
	for _, md := range []mode{modeFit, modeScroll, modeSync} {
		if !seen[md] {
			t.Errorf("mode %d never reached while cycling", md)
		}
	}
}

func TestSyncTimelineAdvances(t *testing.T) {
	m := resize(mustModel(t), 80, 24)
	// Cycle to sync mode (v twice from fit).
	for i := 0; i < 2; i++ {
		nm, _ := m.handleKey(key("v"))
		m = nm.(Model)
	}
	if m.mode != modeSync {
		t.Fatalf("expected sync mode, got %d", m.mode)
	}
	// Start the timeline and advance a few ticks.
	nm, _ := m.handleKey(key(" "))
	m = nm.(Model)
	if !m.running {
		t.Fatal("space did not start the sync timeline")
	}
	for i := 0; i < fps; i++ { // ~1 second
		nm, _ := m.Update(tickMsg(timeNow()))
		m = nm.(Model)
	}
	if m.elapsed <= 0 {
		t.Errorf("elapsed did not advance: %v", m.elapsed)
	}
	if v := m.View(); !strings.Contains(stripANSI(v), "/") {
		t.Error("sync view missing progress time readout")
	}
}

func TestTransposeKeyChangesView(t *testing.T) {
	m := resize(mustModel(t), 80, 24)
	nm, _ := m.handleKey(key("]"))
	m = nm.(Model)
	if m.transp != 1 {
		t.Errorf("transpose = %d, want 1", m.transp)
	}
	// Original key was C; +1 spells the fixed C# (not Db).
	if !strings.Contains(stripANSI(m.View()), "C#") {
		t.Error("transposed view does not show C#")
	}
}

func TestHelpOverlayOpensAndDismisses(t *testing.T) {
	m := resize(mustModel(t), 80, 30)
	nm, _ := m.handleKey(key("?"))
	m = nm.(Model)
	if !m.helping {
		t.Fatal("? did not open the help overlay")
	}
	if !strings.Contains(stripANSI(m.View()), "Keyboard shortcuts") {
		t.Error("help view missing its title")
	}
	// Any key dismisses it.
	nm, _ = m.handleKey(key("x"))
	m = nm.(Model)
	if m.helping {
		t.Error("help overlay did not close")
	}
}

func TestThemeCycleChanges(t *testing.T) {
	m := resize(mustModel(t), 80, 24)
	start := m.tIdx
	nm, _ := m.handleKey(key("t"))
	m = nm.(Model)
	if m.tIdx == start {
		t.Error("theme index did not change on 't'")
	}
}

const blowinPath = "../../testdata/blowin_in_the_wind.cho"

func TestNextPrevSong(t *testing.T) {
	s, _ := chordpro.ParseString("{title: A}\n")
	m := resize(New(s, Options{Path: blowinPath}), 100, 30)

	// Sorted order: blowin, house, scarborough, wagon. 'n' from blowin -> house.
	nm, _ := m.handleKey(key("n"))
	m = nm.(Model)
	if m.base.Title != "House of the Rising Sun" {
		t.Fatalf("next loaded %q, want House of the Rising Sun", m.base.Title)
	}
	// 'p' back to blowin.
	nm, _ = m.handleKey(key("p"))
	m = nm.(Model)
	if m.base.Title != "Blowin' in the Wind" {
		t.Errorf("prev loaded %q, want Blowin' in the Wind", m.base.Title)
	}
}

func TestRandomSongAvoidsCurrent(t *testing.T) {
	s, _ := chordpro.ParseString("{title: A}\n")
	m := resize(New(s, Options{Path: blowinPath}), 100, 30)
	for i := 0; i < 10; i++ {
		nm, _ := m.handleKey(key("r"))
		got := nm.(Model)
		if filepath.Base(got.path) == "blowin_in_the_wind.cho" {
			t.Fatal("random landed on the current song")
		}
		if got.base.Title == "" {
			t.Fatal("random song has no title")
		}
	}
}

func TestEditDoneReloadsFromDisk(t *testing.T) {
	s, _ := chordpro.ParseString("{title: A}\n")
	m := resize(New(s, Options{Path: blowinPath}), 80, 24)
	nm, _ := m.Update(editDoneMsg{})
	m = nm.(Model)
	if m.base.Title != "Blowin' in the Wind" {
		t.Errorf("editDone did not reload from disk, title = %q", m.base.Title)
	}
}

func TestReloadKeepsTranspose(t *testing.T) {
	s, _ := chordpro.ParseString("{title: A}\n")
	m := resize(New(s, Options{Path: blowinPath}), 80, 24)
	(&m).setTranspose(2)
	(&m).reloadKeepingState()
	if m.transp != 2 {
		t.Errorf("transpose lost across reload: %d", m.transp)
	}
	if m.base.Title != "Blowin' in the Wind" {
		t.Errorf("reload read %q, want the file's title", m.base.Title)
	}
}

func TestCtrlNPScrollsLine(t *testing.T) {
	// A long song at a short height, so there is room to scroll a line.
	s, _ := chordpro.ParseString("{title: A}\n")
	m := New(s, Options{Path: blowinPath})
	if err := (&m).loadSong(blowinPath); err != nil {
		t.Fatal(err)
	}
	m = resize(m, 80, 10)
	// Into scroll mode, where line movement is meaningful.
	nm, _ := m.handleKey(key("v"))
	m = nm.(Model)
	if m.mode != modeScroll {
		t.Fatalf("expected scroll mode, got %d", m.mode)
	}
	nm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlN})
	m = nm.(Model)
	if m.offset != 1 {
		t.Errorf("ctrl+n offset = %v, want 1", m.offset)
	}
	nm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlP})
	m = nm.(Model)
	if m.offset != 0 {
		t.Errorf("ctrl+p offset = %v, want 0", m.offset)
	}
}

func TestViewBadgeShownEveryMode(t *testing.T) {
	m := resize(mustModel(t), 100, 30)
	for _, want := range []string{"fit-to-screen", "auto-scroll", "player"} {
		if got := stripANSI(m.View()); !strings.Contains(got, want) {
			t.Errorf("view badge %q missing from bottom bar in mode %d", want, m.mode)
		}
		nm, _ := m.handleKey(key("v"))
		m = nm.(Model)
	}
}

const tabSong = `{title: T}
{artist: A}

{sov: Verse}
[C]Hello [G]world
{eov}

{sot}
TABMARKER e|--0--2--|
{eot}
`

func TestFoldTabsTogglesTabSection(t *testing.T) {
	s, err := chordpro.ParseString(tabSong)
	if err != nil {
		t.Fatal(err)
	}
	m := resize(New(s, Options{}), 100, 40)
	if !strings.Contains(stripANSI(m.View()), "TABMARKER") {
		t.Fatal("tab content missing before folding")
	}
	nm, _ := m.handleKey(key("T"))
	m = nm.(Model)
	if !m.hideTabs {
		t.Fatal("T did not set hideTabs")
	}
	if strings.Contains(stripANSI(m.View()), "TABMARKER") {
		t.Error("tab content still shown after folding")
	}
	// Lyrics from non-tab sections survive the fold.
	if !strings.Contains(stripANSI(m.View()), "Hello") {
		t.Error("folding tabs also hid lyric content")
	}
	nm, _ = m.handleKey(key("T"))
	m = nm.(Model)
	if !strings.Contains(stripANSI(m.View()), "TABMARKER") {
		t.Error("tab content not restored after unfolding")
	}
}

func TestFoldTabsNoTabsIsNoop(t *testing.T) {
	m := resize(mustModel(t), 80, 24) // the shared song has no tab section
	nm, _ := m.handleKey(key("T"))
	m = nm.(Model)
	if m.hideTabs {
		t.Error("hideTabs set on a song with no tab sections")
	}
}

func TestScrollModeToggleAndClamp(t *testing.T) {
	m := resize(mustModel(t), 80, 24)

	// Toggle into scroll mode.
	nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("v")})
	m = nm.(Model)
	if m.mode != modeScroll {
		t.Fatal("expected scroll mode after 'v'")
	}

	// Scrolling up past the top must clamp to 0, not go negative.
	nm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyUp})
	m = nm.(Model)
	if m.offset < 0 {
		t.Errorf("offset went negative: %v", m.offset)
	}
	if v := m.View(); v == "" {
		t.Error("scroll view empty")
	}
}
