package tui

import (
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
		nm, _ := m.handleKey(key("s"))
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
	// Cycle to sync mode (s twice from fit).
	for i := 0; i < 2; i++ {
		nm, _ := m.handleKey(key("s"))
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
	// Original key was C; +1 should show as Db (flat spelling) somewhere.
	if !strings.Contains(stripANSI(m.View()), "Db") {
		t.Error("transposed view does not show Db")
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

func TestScrollModeToggleAndClamp(t *testing.T) {
	m := resize(mustModel(t), 80, 24)

	// Toggle into scroll mode.
	nm, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("s")})
	m = nm.(Model)
	if m.mode != modeScroll {
		t.Fatal("expected scroll mode after 's'")
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
