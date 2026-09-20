package tui

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/monomadic/chordpro-tui/internal/chordpro"
	"github.com/monomadic/chordpro-tui/internal/chords"
	"strings"
	"testing"
)

func TestFinderMouseShape(t *testing.T) {
	m := New(&chordpro.Song{}, Options{StartFinder: true})
	m.w, m.h = 80, 30
	x, y := finderOrigin(m.w, m.h)
	click := func(s, row int) {
		dy := 1
		if row > 0 {
			dy = 2*row + 1
		}
		next, _ := m.Update(tea.MouseMsg{X: x + 2*s, Y: y + dy, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
		m = next.(Model)
	}
	click(1, 0)
	click(2, 2)
	click(3, 2)
	click(4, 1)
	click(5, 0)
	if got := chords.Identify(m.finder.frets); len(got.Names) == 0 || got.Names[0] != "Am" {
		t.Fatal(got)
	}
	click(2, 2)
	if m.finder.frets[2] != -1 {
		t.Fatal("second click must mute")
	}
	click(1, 0)
	if m.finder.frets[1] != -1 {
		t.Fatal("open must toggle to mute")
	}
	before := m.finder
	next, _ := m.Update(tea.MouseMsg{X: x, Y: y + 3, Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease})
	if next.(Model).finder != before {
		t.Fatal("release changed shape")
	}
	next, _ = m.Update(tea.MouseMsg{X: 0, Y: 0, Button: tea.MouseButtonLeft, Action: tea.MouseActionPress})
	if next.(Model).finder != before {
		t.Fatal("outside click changed shape")
	}
}

func TestFinderKeyboardAndSizing(t *testing.T) {
	m := New(&chordpro.Song{}, Options{StartFinder: true})
	for _, key := range []string{"right", "down", " ", "]", "down", "enter"} {
		next, _ := m.handleFinderKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(key)})
		m = next.(Model)
	}
	if m.finder.frets[1] != 3 {
		t.Fatal(m.finder)
	}
	for _, size := range [][2]int{{40, 23}, {80, 30}, {100, 40}, {20, 10}} {
		m.w, m.h = size[0], size[1]
		view := m.View()
		if lipgloss.Width(view) > m.w || lipgloss.Height(view) > m.h {
			t.Fatalf("view overflows %v", size)
		}
		if !strings.Contains(view, "Chordfinder") {
			t.Fatal("missing title")
		}
	}
	next, cmd := m.handleFinderKey(tea.KeyMsg{Type: tea.KeyEsc})
	if next.(Model).finding || cmd == nil {
		t.Fatal("exit must disable mouse")
	}
}
