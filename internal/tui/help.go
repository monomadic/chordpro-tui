package tui

import (
	"strings"

	"chordpro-tui/internal/render"

	"github.com/charmbracelet/lipgloss"
)

type helpRow struct{ key, desc string }

type helpGroup struct {
	title string
	rows  []helpRow
}

// helpGroups is the full keyboard reference shown by the `?` overlay.
var helpGroups = []helpGroup{
	{"Songs", []helpRow{
		{"o", "open song finder"},
		{"n / p", "next / previous song"},
		{"r", "random song"},
		{"e", "edit in $EDITOR"},
		{"w", "save transposed copy"},
	}},
	{"View", []helpRow{
		{"s", "cycle fit / scroll / sync"},
		{"space", "play / pause"},
		{"↑ ↓  j k", "scroll a line / seek"},
		{"f b", "page down / up"},
		{"g G", "jump top / bottom"},
		{"+ -", "scroll speed / sync length"},
	}},
	{"Music", []helpRow{
		{"c", "chord-shape sheet"},
		{"[ ]", "transpose down / up"},
		{"0", "reset transpose"},
	}},
	{"Appearance", []helpRow{
		{"t", "cycle theme"},
		{"B", "toggle background fill"},
		{"h", "toggle title header"},
	}},
	{"", []helpRow{
		{"?", "this help"},
		{"q", "quit"},
	}},
}

// helpView renders the keyboard-shortcut overlay, centered on the screen.
func helpView(w, h int, th *render.Theme) string {
	keyStyle := lipgloss.NewStyle().Foreground(th.P.Chord).Bold(true)

	keyW := 0
	for _, g := range helpGroups {
		for _, r := range g.rows {
			if x := lipgloss.Width(r.key); x > keyW {
				keyW = x
			}
		}
	}

	var lines []string
	lines = append(lines, th.Title.Render("Keyboard shortcuts"), "")
	for _, g := range helpGroups {
		if g.title != "" {
			lines = append(lines, th.Section.Render(g.title))
		}
		for _, r := range g.rows {
			pad := strings.Repeat(" ", keyW-lipgloss.Width(r.key))
			lines = append(lines, keyStyle.Render(r.key)+pad+"   "+th.Lyric.Render(r.desc))
		}
		lines = append(lines, "")
	}
	lines = append(lines, th.Muted.Render("press any key to close"))

	panel := strings.Join(lines, "\n")
	// Left-align the rows within a fixed-width block, then center that block on
	// screen, so the key column lines up instead of every row floating.
	block := lipgloss.NewStyle().Width(lipgloss.Width(panel)).Render(panel)
	return lipgloss.Place(w, h, lipgloss.Center, lipgloss.Center, block)
}
