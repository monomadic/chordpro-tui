// Command chordpro-tui renders a ChordPro song to the terminal: a colorful
// title card with chords stacked over lyrics, laid out to fill one screen.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"

	"chordpro-tui/internal/chordpro"
	"chordpro-tui/internal/render"
	"chordpro-tui/internal/tui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
	"golang.org/x/term"
)

func main() {
	var (
		printMode = flag.Bool("print", false, "render once to stdout and exit (no interactive TUI)")
		scroll    = flag.Bool("scroll", false, "start in auto-scrolling teleprompter mode")
		transpose = flag.Int("transpose", 0, "transpose chords by N semitones")
		themeName = flag.String("theme", "", "color theme: Mocha, Tokyo Night, Gruvbox, Dracula, Nord, Synthwave, Cyberpunk, Laser, Vapor")
		bg        = flag.Bool("bg", false, "fill the screen with the theme's background color")
		cols      = flag.Int("width", 0, "override terminal width (print mode)")
		rows      = flag.Int("height", 0, "override terminal height (print mode)")
	)
	flag.Usage = usage
	flag.Parse()

	inputPath, err := resolveInput(flag.Arg(0))
	if err != nil {
		fmt.Fprintln(os.Stderr, "chordpro-tui:", err)
		os.Exit(1)
	}
	song, err := readSong(inputPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "chordpro-tui:", err)
		os.Exit(1)
	}

	theme := render.ThemeByName(*themeName)
	interactive := term.IsTerminal(int(os.Stdout.Fd())) && !*printMode

	if !interactive {
		if os.Getenv("CHORDPRO_TUI_FORCE_COLOR") != "" {
			lipgloss.SetColorProfile(termenv.TrueColor)
			lipgloss.SetHasDarkBackground(true)
		}
		song = song.Transposed(*transpose)
		w, h := *cols, *rows
		if w == 0 || h == 0 {
			tw, th, err := term.GetSize(int(os.Stdout.Fd()))
			if err == nil {
				if w == 0 {
					w = tw
				}
				if h == 0 {
					h = th
				}
			}
		}
		if w == 0 {
			w = 100
		}
		if h == 0 {
			h = 40
		}
		out := render.Render(song, w, h, theme)
		if *bg {
			out = render.ApplyBackground(out, w, theme.P.Bg)
		}
		fmt.Println(out)
		return
	}

	p := tea.NewProgram(
		tui.New(song, tui.Options{
			StartScroll: *scroll,
			Transpose:   *transpose,
			ThemeName:   *themeName,
			Path:        inputPath,
			Background:  *bg,
		}),
		tea.WithAltScreen(),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "chordpro-tui:", err)
		os.Exit(1)
	}
}

// resolveInput turns a CLI argument into a concrete song path. A directory
// resolves to the most recently modified ChordPro file inside it; an empty
// argument is left as-is (stdin).
func resolveInput(path string) (string, error) {
	if path == "" {
		return "", nil
	}
	fi, err := os.Stat(path)
	if err != nil {
		return path, nil // let readSong surface the open error
	}
	if fi.IsDir() {
		return tui.NewestSong(path)
	}
	return path, nil
}

// readSong loads a song from the given path, or from stdin when path is empty.
func readSong(path string) (*chordpro.Song, error) {
	if path == "" {
		fi, _ := os.Stdin.Stat()
		if fi.Mode()&os.ModeCharDevice != 0 {
			return nil, fmt.Errorf("no input: pass a .cho file or pipe one in\n\n%s", usageText())
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return nil, err
		}
		return chordpro.ParseString(string(data))
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return chordpro.Parse(f)
}

func usage() { fmt.Fprint(os.Stderr, usageText()) }

func usageText() string {
	return `chordpro-tui — colorful ChordPro renderer for the terminal

Usage:
  chordpro-tui [flags] <song.cho>
  chordpro-tui [flags] <folder>     # opens the newest song in the folder
  chordpro-tui < song.cho

Flags:
  -print           render once to stdout and exit
  -scroll          start in auto-scrolling teleprompter mode
  -transpose N     transpose chords by N semitones
  -theme NAME      Mocha, Tokyo Night, Gruvbox, Dracula, Nord,
                   Synthwave, Cyberpunk, Laser, Vapor
  -bg              fill the screen with the theme's background color
  -width  N        override width (print mode)
  -height N        override height (print mode)

Keys (interactive):
  ?            show all keyboard shortcuts
  o            open another song from this folder (fuzzy finder)
  e            edit the current file in $EDITOR
  n / p        load next / previous song in the folder
  r            load a random song in the folder
  s            cycle view: fit → scroll → sync
  c            chord-shape sheet for the current song
  t            cycle color theme
  B            toggle themed background fill
  h            toggle the title header
  [ / ]        transpose down / up        (fit mode)
  0            reset transpose
  w            save a transposed copy alongside the original
  space        pause/resume scroll · play/pause sync
  g            restart sync timeline (jump to top)
  +/-          scroll speed / sync length (scroll & sync modes)
  ↑/↓ j/k      scroll a line / seek
  g/G          jump to top / bottom
  q            quit
`
}
