package tui

import (
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"chordpro-tui/internal/chordpro"
	"chordpro-tui/internal/render"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeFit    mode = iota // whole song fitted to one screen
	modeScroll             // tall single column, constant auto-scroll
	modeSync               // tall single column, scrolls over the song duration
)

const fps = 30

// tickMsg drives the animation loop.
type tickMsg time.Time

// editDoneMsg is delivered after the external $EDITOR exits.
type editDoneMsg struct{ err error }

// Model is the Bubbletea state for viewing a song.
type Model struct {
	base   *chordpro.Song  // untransposed source
	song   *chordpro.Song  // current (transposed) view
	themes []*render.Theme // cycle order
	tIdx   int             // active theme index
	theme  *render.Theme   // == themes[tIdx]
	transp int             // transpose in semitones

	w, h int
	mode mode

	long []string // RenderLong cache for scroll/sync modes

	// constant auto-scroll
	offset    float64
	auto      bool
	linesPerS float64

	// duration sync
	duration time.Duration
	elapsed  time.Duration
	running  bool

	// file picker
	path    string // path of the current song ("" if from stdin)
	picking bool
	pick    picker

	helping    bool // showing the keyboard-shortcut overlay
	bgFill     bool // fill the screen with the theme's background color
	hideHeader bool // hide the title/metadata header on song views

	chords bool // showing the chord-shapes sheet overlay
}

// Options configure the initial view state.
type Options struct {
	StartScroll bool
	Transpose   int
	ThemeName   string
	Path        string // path of the song being shown, for sibling browsing
	Background  bool   // start with the themed background fill on
}

// New builds the initial model.
func New(song *chordpro.Song, opts Options) Model {
	themes := render.Themes()
	tIdx := 0
	if i := render.ThemeIndexByName(opts.ThemeName); i >= 0 {
		tIdx = i
	}
	m := Model{
		base:      song,
		themes:    themes,
		tIdx:      tIdx,
		theme:     themes[tIdx],
		transp:    clampTranspose(opts.Transpose),
		linesPerS: speedFromTempo(song.SpeedHint()),
		duration:  songDuration(song),
		path:      opts.Path,
		bgFill:    opts.Background,
	}
	m.song = song.Transposed(m.transp)
	if opts.StartScroll {
		m.mode = modeScroll
		m.auto = true
	}
	return m
}

func clampTranspose(n int) int {
	if n > 11 {
		return 11
	}
	if n < -11 {
		return -11
	}
	return n
}

func (m Model) Init() tea.Cmd { return tick() }

func tick() tea.Cmd {
	return tea.Tick(time.Second/fps, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// speedFromTempo turns a BPM string into a constant scroll speed in lines/sec.
// We assume roughly one lyric line per two bars of 4/4: a gentle crawl.
func speedFromTempo(tempo string) float64 {
	bpm, err := strconv.ParseFloat(strings.TrimSpace(tempo), 64)
	if err != nil || bpm <= 0 {
		bpm = 100
	}
	const beatsPerLine = 8.0
	return bpm / 60.0 / beatsPerLine
}

// songDuration returns the song's stated duration, or a sensible default for
// sync mode when none is given.
func songDuration(song *chordpro.Song) time.Duration {
	if song.Duration > 0 {
		return song.Duration
	}
	return 210 * time.Second // 3:30, adjustable in-app with +/-
}

// rebuild refreshes the transposed song and the long-render cache after a
// change to transpose, theme, or width.
func (m *Model) rebuild() {
	m.song = m.base.Transposed(m.transp)
	if m.w > 0 {
		m.long = render.RenderLongWith(m.song, m.w, m.theme, render.RenderOpts{HideHeader: m.hideHeader})
	}
	m.clampOffset()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		m.rebuild()
		return m, nil

	case tickMsg:
		switch m.mode {
		case modeScroll:
			if m.auto {
				m.offset += m.linesPerS / fps
				m.clampOffset()
			}
		case modeSync:
			if m.running {
				m.elapsed += time.Second / fps
				if m.elapsed >= m.duration {
					m.elapsed = m.duration
					m.running = false
				}
			}
		}
		return m, tick()

	case editDoneMsg:
		// The file may have changed; reload it, keeping transpose & theme.
		m.reloadKeepingState()
		return m, nil

	case tea.KeyMsg:
		return m.handleKey(msg)
	}
	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.picking {
		return m.handlePickerKey(msg)
	}
	if m.chords {
		return m.handleChordsKey(msg)
	}
	if m.helping {
		// Any key dismisses the help overlay (ctrl+c still quits).
		if msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
		m.helping = false
		return m, nil
	}

	switch msg.String() {
	case "q", "ctrl+c", "esc":
		return m, tea.Quit

	case "?": // keyboard-shortcut help
		m.helping = true

	case "o": // open another song from this directory
		m.pick = newPicker(m.dir(), m.path)
		m.picking = true
		return m, nil

	case "e": // edit the current file in $EDITOR
		if m.path != "" {
			return m, editCmd(m.path)
		}
	case "n": // next song in this folder
		m.loadSibling(1)
	case "p": // previous song in this folder
		m.loadSibling(-1)
	case "r": // random song in this folder
		m.loadRandom()

	case "s": // cycle view mode
		m.mode = (m.mode + 1) % 3
		switch m.mode {
		case modeScroll:
			m.auto = true
		case modeSync:
			m.running = false
		}

	case "t": // cycle theme
		m.tIdx = (m.tIdx + 1) % len(m.themes)
		m.theme = m.themes[m.tIdx]
		m.rebuild()

	case "c": // show the chord-shapes sheet for the current song
		m.chords = true

	case "B": // toggle themed background fill
		m.bgFill = !m.bgFill

	case "h": // toggle the title/metadata header
		m.hideHeader = !m.hideHeader
		m.rebuild()

	case "]", "+", "=": // transpose up (also speed/duration, see below)
		if m.mode == modeFit {
			m.setTranspose(m.transp + 1)
		}
	case "[", "-", "_": // transpose down
		if m.mode == modeFit {
			m.setTranspose(m.transp - 1)
		}
	case "0": // reset transpose
		m.setTranspose(0)

	case " ":
		switch m.mode {
		case modeScroll:
			m.auto = !m.auto
		case modeSync:
			m.running = !m.running
			if m.elapsed >= m.duration {
				m.elapsed = 0 // restart from the top if finished
			}
		}

	case "down", "j":
		m.nudge(1)
	case "up", "k":
		m.nudge(-1)
	case "pgdown", "f":
		m.nudge(m.h)
	case "pgup", "b":
		m.nudge(-m.h)
	case "g", "home":
		m.offset, m.elapsed = 0, 0
	case "G", "end":
		m.offset = float64(len(m.long))
		m.elapsed = m.duration
		m.clampOffset()
	}

	// In scroll/sync modes the bracket/plus keys retune speed or duration
	// rather than transpose.
	switch msg.String() {
	case "]", "+", "=":
		if m.mode == modeScroll {
			m.linesPerS *= 1.25
		} else if m.mode == modeSync {
			m.duration += 5 * time.Second
		}
	case "[", "-", "_":
		if m.mode == modeScroll {
			m.linesPerS /= 1.25
		} else if m.mode == modeSync && m.duration > 10*time.Second {
			m.duration -= 5 * time.Second
		}
	}
	return m, nil
}

func (m *Model) setTranspose(n int) {
	m.transp = clampTranspose(n)
	m.rebuild()
}

// dir is the directory the picker browses: the current song's folder, or the
// working directory when the song came from stdin.
func (m Model) dir() string {
	if m.path == "" {
		return "."
	}
	return filepath.Dir(m.path)
}

// handlePickerKey drives the open-file overlay.
func (m Model) handlePickerKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+g":
		m.picking = false
	case "ctrl+c":
		return m, tea.Quit
	case "enter":
		if sel, ok := m.pick.selected(); ok {
			if err := m.loadSong(sel); err != nil {
				m.pick.err = err.Error()
			} else {
				m.picking = false
			}
		}
	case "up", "ctrl+p", "ctrl+k":
		m.pick.move(-1)
	case "down", "ctrl+n", "ctrl+j":
		m.pick.move(1)
	case "backspace":
		m.pick.backspace()
	case "ctrl+u":
		m.pick.setQuery("")
	default:
		if s := msg.String(); len([]rune(s)) == 1 {
			m.pick.appendQuery(s)
		} else if msg.Type == tea.KeySpace {
			m.pick.appendQuery(" ")
		}
	}
	return m, nil
}

// handleChordsKey drives the chord-shapes sheet: transpose and theme stay live
// so the diagrams track the song, and c/esc dismiss it.
func (m Model) handleChordsKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "c", "esc":
		m.chords = false
	case "q", "ctrl+c":
		return m, tea.Quit
	case "]", "+", "=":
		m.setTranspose(m.transp + 1)
	case "[", "-", "_":
		m.setTranspose(m.transp - 1)
	case "0":
		m.setTranspose(0)
	case "t":
		m.tIdx = (m.tIdx + 1) % len(m.themes)
		m.theme = m.themes[m.tIdx]
		m.rebuild()
	case "B":
		m.bgFill = !m.bgFill
	}
	return m, nil
}

// loadSong replaces the current song with the one at path, resetting transpose
// and playback state while keeping the active theme.
func (m *Model) loadSong(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	song, err := chordpro.Parse(f)
	if err != nil {
		return err
	}
	m.base = song
	m.path = path
	m.transp = 0
	m.duration = songDuration(song)
	m.linesPerS = speedFromTempo(song.SpeedHint())
	m.elapsed = 0
	m.offset = 0
	m.running = false
	m.rebuild()
	return nil
}

// reloadKeepingState re-reads the current file (e.g. after editing) while
// preserving the transpose offset and theme.
func (m *Model) reloadKeepingState() {
	if m.path == "" {
		return
	}
	saved := m.transp
	if err := m.loadSong(m.path); err == nil {
		m.transp = saved
		m.rebuild()
	}
}

// loadSibling loads the next (delta=1) or previous (delta=-1) song in the
// folder, wrapping around.
func (m *Model) loadSibling(delta int) {
	paths, _ := chordFilePaths(m.dir())
	if len(paths) == 0 {
		return
	}
	i := siblingIndex(paths, m.path)
	if i < 0 {
		i = 0
	} else {
		i = ((i+delta)%len(paths) + len(paths)) % len(paths)
	}
	_ = m.loadSong(paths[i])
}

// loadRandom loads a random song in the folder, avoiding the current one.
func (m *Model) loadRandom() {
	paths, _ := chordFilePaths(m.dir())
	n := len(paths)
	if n == 0 {
		return
	}
	cur := siblingIndex(paths, m.path)
	j := rand.Intn(n)
	if n > 1 && j == cur {
		j = (j + 1) % n
	}
	_ = m.loadSong(paths[j])
}

// siblingIndex finds path within paths by base name, or -1.
func siblingIndex(paths []string, path string) int {
	base := filepath.Base(path)
	for i, p := range paths {
		if filepath.Base(p) == base {
			return i
		}
	}
	return -1
}

// editCmd launches $EDITOR (or vi) on path, suspending the TUI until it exits.
func editCmd(path string) tea.Cmd {
	editor := os.Getenv("EDITOR")
	if strings.TrimSpace(editor) == "" {
		editor = "vi"
	}
	parts := strings.Fields(editor)
	args := append(parts[1:], path)
	return tea.ExecProcess(exec.Command(parts[0], args...), func(err error) tea.Msg {
		return editDoneMsg{err}
	})
}

// nudge scrolls in scroll mode, or seeks the timeline in sync mode.
func (m *Model) nudge(lines int) {
	switch m.mode {
	case modeScroll:
		m.offset += float64(lines)
		m.clampOffset()
	case modeSync:
		per := m.duration / time.Duration(max(1, len(m.long)))
		m.elapsed += time.Duration(lines) * per
		if m.elapsed < 0 {
			m.elapsed = 0
		}
		if m.elapsed > m.duration {
			m.elapsed = m.duration
		}
	}
}

func (m *Model) clampOffset() {
	hi := float64(len(m.long) - m.h)
	if hi < 0 {
		hi = 0
	}
	if m.offset > hi {
		m.offset = hi
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m Model) View() string {
	if m.w == 0 || m.h == 0 {
		return "loading…"
	}
	var out string
	switch {
	case m.picking:
		out = m.pick.view(m.w, m.h, m.theme)
	case m.helping:
		out = helpView(m.w, m.h, m.theme)
	case m.chords:
		out = render.RenderChordSheet(m.song, m.w, m.h, m.theme)
	case m.mode == modeScroll:
		out = m.windowView(m.offset, m.scrollStatus())
	case m.mode == modeSync:
		out = m.windowView(m.syncOffset(), m.progressBar())
	default:
		out = render.RenderWith(m.song, m.w, m.h, m.theme, render.RenderOpts{HideHeader: m.hideHeader})
	}
	if m.bgFill {
		out = render.ApplyBackground(out, m.w, m.theme.P.Bg)
	}
	return out
}

// syncOffset maps elapsed time to a scroll offset so the last line is reached
// exactly at the song's duration.
func (m Model) syncOffset() float64 {
	content := m.h - 1 // reserve a row for the progress bar
	hi := len(m.long) - content
	if hi <= 0 {
		return 0
	}
	if m.duration <= 0 {
		return 0
	}
	p := float64(m.elapsed) / float64(m.duration)
	if p > 1 {
		p = 1
	}
	return p * float64(hi)
}

// windowView shows a height-sized window into the long render with a status
// line pinned to the bottom row.
func (m Model) windowView(offset float64, status string) string {
	content := m.h - 1
	start := int(offset)
	if start < 0 {
		start = 0
	}
	end := start + content
	if end > len(m.long) {
		end = len(m.long)
	}
	window := make([]string, 0, content)
	if start < len(m.long) {
		window = append(window, m.long[start:end]...)
	}
	for len(window) < content {
		window = append(window, "")
	}
	body := lipgloss.NewStyle().Width(m.w).Height(content).MaxHeight(content).
		Render(strings.Join(window, "\n"))
	return body + "\n" + status
}

// scrollStatus is the bottom line shown in constant auto-scroll mode.
func (m Model) scrollStatus() string {
	state := "▶ auto"
	if !m.auto {
		state = "⏸ paused"
	}
	left := fmt.Sprintf("%s  %.1f ln/s", state, m.linesPerS)
	hint := "space pause · +/- speed · s mode · ? help · q"
	return m.statusBar(left, hint)
}

// progressBar is the bottom line shown in duration-sync mode: a play state,
// elapsed/total time, and a filled bar.
func (m Model) progressBar() string {
	icon := "⏸"
	if m.running {
		icon = "▶"
	}
	if m.elapsed >= m.duration {
		icon = "■"
	}
	times := fmt.Sprintf("%s %s / %s", icon, mmss(m.elapsed), mmss(m.duration))

	hint := "space play · g restart · +/- length · ? help"
	// Lay out: [times] [bar....] [hint]
	reserved := lipgloss.Width(times) + lipgloss.Width(hint) + 4
	barW := m.w - reserved
	if barW < 6 {
		// Too narrow for the hint; drop it.
		hint = ""
		barW = m.w - lipgloss.Width(times) - 2
	}
	if barW < 1 {
		barW = 1
	}

	p := 0.0
	if m.duration > 0 {
		p = float64(m.elapsed) / float64(m.duration)
	}
	if p > 1 {
		p = 1
	}
	filled := int(p * float64(barW))
	bar := m.theme.Chord.Render(strings.Repeat("━", filled))
	if filled < barW {
		bar += m.theme.ChorusBar.Render("╸")
		bar += m.theme.Muted.Render(strings.Repeat("─", barW-filled-1))
	}

	line := m.theme.Section.Render(times) + " " + bar
	if hint != "" {
		gap := m.w - lipgloss.Width(times) - 1 - barW - lipgloss.Width(hint) - 1
		if gap < 1 {
			gap = 1
		}
		line += strings.Repeat(" ", gap) + m.theme.Muted.Render(hint)
	}
	return line
}

// statusBar justifies a left label and a right hint across the screen width.
func (m Model) statusBar(left, right string) string {
	gap := m.w - lipgloss.Width(left) - lipgloss.Width(right) - 2
	if gap < 1 {
		return m.theme.Muted.Render(right)
	}
	return m.theme.Section.Render(left) + strings.Repeat(" ", gap) + m.theme.Muted.Render(right)
}

func mmss(d time.Duration) string {
	if d < 0 {
		d = 0
	}
	s := int(d.Seconds())
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}
