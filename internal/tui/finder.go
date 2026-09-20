package tui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/monomadic/chordpro-tui/internal/chords"
	"github.com/monomadic/chordpro-tui/internal/render"
)

type finderState struct {
	frets                  [6]int
	stringIndex, row, base int
}

func newFinder() finderState { return finderState{frets: [6]int{-1, -1, -1, -1, -1, -1}, base: 1} }

func (f *finderState) toggle() {
	if f.row == 0 {
		if f.frets[f.stringIndex] == 0 {
			f.frets[f.stringIndex] = -1
		} else {
			f.frets[f.stringIndex] = 0
		}
	} else {
		fret := f.base + f.row - 1
		if f.frets[f.stringIndex] == fret {
			f.frets[f.stringIndex] = -1
		} else {
			f.frets[f.stringIndex] = fret
		}
	}
}

// Geometry is shared by rendering and hit testing. The diagram itself is the
// existing 11-column, four-fret sheet diagram, with room for its fret label.
func finderOrigin(w, h int) (int, int) { return (w - 11) / 2, max(4, (h-17)/2) }
func finderFits(w, h int) bool         { return w >= 40 && h >= 23 }

func (m Model) handleFinderMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	if !finderFits(m.w, m.h) || msg.Button != tea.MouseButtonLeft || msg.Action != tea.MouseActionPress {
		return m, nil
	}
	x, y := finderOrigin(m.w, m.h)
	dx, dy := msg.X-x, msg.Y-y
	if dx < 0 || dx > 10 || dy < 1 || dy > 9 {
		return m, nil
	}
	m.finder.stringIndex = (dx + 1) / 2
	if dy <= 2 {
		m.finder.row = 0
	} else {
		m.finder.row = (dy - 1) / 2
	}
	m.finder.toggle()
	return m, nil
}

func (m Model) handleFinderKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	f := &m.finder
	switch msg.String() {
	case "q", "ctrl+c":
		return m, tea.Quit
	case "esc", "F":
		m.finding = false
		return m, tea.DisableMouse
	case "left", "h":
		f.stringIndex = max(0, f.stringIndex-1)
	case "right", "l":
		f.stringIndex = min(5, f.stringIndex+1)
	case "up", "k":
		f.row = max(0, f.row-1)
	case "down", "j":
		f.row = min(4, f.row+1)
	case " ", "enter":
		f.toggle()
	case "o":
		f.frets[f.stringIndex] = 0
	case "x":
		f.frets[f.stringIndex] = -1
	case "[", "pgup":
		f.base = max(1, f.base-1)
	case "]", "pgdown":
		f.base = min(21, f.base+1)
	case "0":
		*f = newFinder()
	case "t":
		m.tIdx = (m.tIdx + 1) % len(m.themes)
		m.theme = m.themes[m.tIdx]
		m.rebuild()
	case "B":
		m.bgFill = !m.bgFill
	}
	return m, nil
}

func (m Model) finderView() string {
	if !finderFits(m.w, m.h) {
		return ansi.Truncate("Chordfinder needs 40 columns × 23 rows. Esc back · q quit", m.w, "")
	}
	f := m.finder
	result := chords.Identify(f.frets)
	name := "—"
	if len(result.Names) > 0 {
		name = result.Names[0]
	}
	shape := chords.Shape{Name: name, BaseFret: f.base, Frets: make([]int, 6)}
	positions := make([]string, 6)
	for i, v := range f.frets {
		positions[i] = "×"
		if v >= 0 {
			positions[i] = fmt.Sprint(v)
		}
		shape.Frets[i] = v
		if v > 0 {
			shape.Frets[i] = v - f.base + 1
			if shape.Frets[i] <= 0 {
				shape.Frets[i] = 5
			} // fretted outside this window, never open/muted
		}
	}
	lines := make([]string, m.h)
	put := func(y int, s string) {
		if y >= 0 && y < len(lines) {
			lines[y] = lipgloss.PlaceHorizontal(m.w, lipgloss.Center, ansi.Truncate(s, m.w, "…"))
		}
	}
	put(0, m.theme.Title.Render("Chordfinder"))
	put(1, m.theme.Muted.Render("Standard tuning · E A D G B E"))
	x, y := finderOrigin(m.w, m.h)
	for i, line := range strings.Split(render.ChordDiagram(shape, m.theme, f.stringIndex, f.row), "\n") {
		lines[y+i] = strings.Repeat(" ", x) + line
	}
	put(y+12, m.theme.Section.Render("Frets: "+strings.Join(positions, " ")))
	note := "Select notes on the neck"
	if len(result.Notes) > 0 {
		note = "Notes: " + strings.Join(result.Notes, " · ")
	}
	put(y+13, note)
	if len(result.Names) > 1 {
		put(y+14, "Also: "+strings.Join(result.Names[1:], ", "))
	} else if len(result.Notes) > 0 && len(result.Names) == 0 {
		put(y+14, "No exact common chord match")
	}
	put(m.h-4, m.theme.Muted.Render("Click / Space toggle · arrows / hjkl move"))
	put(m.h-3, m.theme.Muted.Render("o open · x mute · [ ] frets · 0 clear"))
	put(m.h-2, m.theme.Muted.Render("t theme · B background · Esc back · q quit"))
	return strings.Join(lines, "\n")
}
