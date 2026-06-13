package render

import (
	"strconv"
	"strings"

	"chordpro-tui/internal/chordpro"

	"github.com/charmbracelet/lipgloss"
)

// buildHeader composes the framed title card: title, subtitle/artist line, and
// a row of metadata "pills" (key, capo, tempo, ...).
func buildHeader(song *chordpro.Song, width int, th *Theme) string {
	var rows []string

	title := song.Title
	if title == "" {
		title = "Untitled"
	}
	rows = append(rows, th.Title.Render(title))

	if sub := subtitleLine(song); sub != "" {
		rows = append(rows, th.Subtitle.Render(sub))
	}

	if pills := buildPills(song, th); pills != "" {
		rows = append(rows, pills)
	}

	inner := lipgloss.JoinVertical(lipgloss.Center, rows...)

	// Keep the frame within the screen width (account for border + padding).
	maxInner := width - 6
	if maxInner > 8 && lipgloss.Width(inner) > maxInner {
		inner = lipgloss.NewStyle().Width(maxInner).Render(inner)
	}
	return th.Frame.Render(inner)
}

// subtitleLine merges artist, subtitle, and album into one bullet-joined line.
func subtitleLine(song *chordpro.Song) string {
	var parts []string
	for _, p := range []string{song.Artist, song.Subtitle, song.Album} {
		if p != "" {
			parts = append(parts, p)
		}
	}
	return strings.Join(parts, "  •  ")
}

// buildPills renders the metadata as label/value chips joined horizontally.
func buildPills(song *chordpro.Song, th *Theme) string {
	meta := song.Meta()
	if len(meta) == 0 {
		return ""
	}
	var pills []string
	for i, kv := range meta {
		if i > 0 {
			pills = append(pills, " ")
		}
		pill := th.PillKey.Render(kv[0]) + th.PillVal.Render(kv[1])
		pills = append(pills, pill)
	}
	return lipgloss.JoinHorizontal(lipgloss.Center, pills...)
}

// buildFooter renders the muted status line at the bottom of the screen: song
// info plus the active theme and transpose state on the left, key hints on the
// right. When truncated is set, the hint nudges scroll mode.
func buildFooter(song *chordpro.Song, width int, th *Theme, truncated bool) string {
	left := song.Title
	if song.Artist != "" {
		left += " — " + song.Artist
	}
	if status := statusBadge(song, th); status != "" {
		left += "   " + status
	}

	hint := "o open · e edit · n/p/r songs · s view · t theme · B bg · q"
	if truncated {
		hint = "▾ more — s scroll · o open · q quit"
	}

	gap := width - lipgloss.Width(left) - lipgloss.Width(hint) - 2
	if gap < 2 {
		// Too narrow for both: just show the hint.
		return th.Muted.Render(hint)
	}
	return th.Muted.Render(left) + strings.Repeat(" ", gap) + th.Muted.Render(hint)
}

// statusBadge describes the active theme (with its position in the cycle) and
// any transpose offset.
func statusBadge(song *chordpro.Song, th *Theme) string {
	badge := th.Name
	if i := ThemeIndexByName(th.Name); i >= 0 {
		badge += " " + strconv.Itoa(i+1) + "/" + strconv.Itoa(len(Palettes))
	}
	if n := song.TransposeBy; n != 0 {
		sign := "+"
		if n < 0 {
			sign = "−" // U+2212 minus, lines up nicer than hyphen
		}
		abs := n
		if abs < 0 {
			abs = -abs
		}
		badge += " · " + sign + strconv.Itoa(abs) + "st"
	}
	return badge
}
