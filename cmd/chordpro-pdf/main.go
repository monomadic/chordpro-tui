// Command chordpro-pdf exports a ChordPro song as a single-page PDF sized for
// a target screen (iPad, iPhone, Mac full-screen) or paper. The whole song is
// scaled to exactly fill one page — big type for short songs, columns and
// smaller type for long ones — in monochrome sheet-music style.
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"chordpro-tui/internal/chordpro"
	"chordpro-tui/internal/pdf"

	"golang.org/x/term"
)

func main() {
	var (
		out       string
		preset    = flag.String("preset", "ipad-mini", "page preset (see --list-presets)")
		landscape = flag.Bool("landscape", false, "turn the page to landscape")
		portrait  = flag.Bool("portrait", false, "turn the page to portrait")
		page      = flag.String("page", "", "custom page size in points, e.g. 800x600 (overrides --preset)")
		transpose = flag.Int("transpose", 0, "transpose chords by N semitones")
		columns   = flag.Int("columns", 0, "force column count (default: auto)")
		maxFont   = flag.Float64("max-font", 20, "largest body font size in points")
		margin    = flag.Float64("margin", 0, "page margin in points (default: 4.5% of the short edge)")
		serif     = flag.Bool("serif", false, "serif lyrics (Times) instead of sans (Helvetica)")
		inverted  = flag.Bool("inverted", false, "white text on a black page (night mode)")
		noDiags   = flag.Bool("no-diagrams", false, "omit the chord fingering diagrams in the header")
		list      = flag.Bool("list-presets", false, "list page presets and exit")
	)
	flag.StringVar(&out, "o", "", "output file (default: <input>.<preset>.pdf; \"-\" = stdout)")
	flag.StringVar(&out, "output", "", "alias for -o")
	flag.Usage = usage
	flag.Parse()

	if *list {
		listPresets()
		return
	}

	pageW, pageH, sizeName, err := pageSize(*preset, *page, *landscape, *portrait)
	if err != nil {
		fail(err)
	}
	input := flag.Arg(0)
	song, err := readSong(input)
	if err != nil {
		fail(err)
	}
	song = song.Transposed(*transpose)

	outPath := out
	if outPath == "" {
		outPath = defaultOut(input, song, sizeName)
	}
	var dst io.Writer = os.Stdout
	if outPath != "-" {
		f, err := os.Create(outPath)
		if err != nil {
			fail(err)
		}
		defer f.Close()
		dst = f
	}

	res, err := pdf.Export(song, pdf.Options{
		PageW:      pageW,
		PageH:      pageH,
		Margin:     *margin,
		Columns:    *columns,
		MaxFont:    *maxFont,
		Serif:      *serif,
		NoDiagrams: *noDiags,
		Inverted:   *inverted,
	}, dst)
	if err != nil {
		fail(err)
	}
	report(song, res, sizeName, outPath, *transpose)
}

// report prints a short styled summary of what was written to stderr, so it
// stays out of the way when the PDF itself goes to stdout.
func report(song *chordpro.Song, res pdf.Result, sizeName, outPath string, transpose int) {
	st := newStyler(os.Stderr)
	w := os.Stderr

	title := song.Title
	if title == "" {
		title = "Untitled"
	}
	head := st.bold(title)
	if song.Artist != "" {
		head += st.dim(" — ") + song.Artist
	}
	if transpose != 0 {
		head += st.dim(fmt.Sprintf("  (%+d st)", transpose))
	}

	layout := fmt.Sprintf("%d %s · %.1f pt body", res.Columns, plural(res.Columns, "column"), res.BodyPt)
	dest := st.cyan(outPath)
	if outPath == "-" {
		dest = st.dim("(stdout)")
	}

	fmt.Fprintf(w, "\n  %s\n\n", head)
	fmt.Fprintf(w, "  %s  %s %s\n", st.dim("page  "), sizeName, st.dim(fmt.Sprintf("· %g × %g pt", res.PageW, res.PageH)))
	fmt.Fprintf(w, "  %s  %s\n", st.dim("layout"), layout)
	fmt.Fprintf(w, "  %s  %s\n\n", st.dim("output"), dest)
}

func plural(n int, word string) string {
	if n == 1 {
		return word
	}
	return word + "s"
}

func fail(err error) {
	st := newStyler(os.Stderr)
	fmt.Fprintln(os.Stderr, st.red("chordpro-pdf:"), err)
	os.Exit(1)
}

// styler applies basic ANSI styling when the target is a terminal and the
// user hasn't opted out via NO_COLOR.
type styler struct{ on bool }

func newStyler(f *os.File) styler {
	return styler{on: term.IsTerminal(int(f.Fd())) && os.Getenv("NO_COLOR") == ""}
}

func (s styler) wrap(code, t string) string {
	if !s.on || t == "" {
		return t
	}
	return "\x1b[" + code + "m" + t + "\x1b[0m"
}

func (s styler) bold(t string) string { return s.wrap("1", t) }
func (s styler) dim(t string) string  { return s.wrap("2", t) }
func (s styler) cyan(t string) string { return s.wrap("36", t) }
func (s styler) red(t string) string  { return s.wrap("31", t) }

// pageSize resolves the page geometry from a preset or a custom WxH string,
// then applies an orientation override.
func pageSize(preset, custom string, landscape, portrait bool) (w, h float64, name string, err error) {
	if custom != "" {
		if _, err := fmt.Sscanf(strings.ToLower(custom), "%gx%g", &w, &h); err != nil || w <= 0 || h <= 0 {
			return 0, 0, "", fmt.Errorf("bad --page %q: want WIDTHxHEIGHT in points, e.g. 800x600", custom)
		}
		name = "Custom"
	} else {
		p, ok := pdf.PresetByName(preset)
		if !ok {
			return 0, 0, "", fmt.Errorf("unknown preset %q (try --list-presets)", preset)
		}
		w, h, name = p.W, p.H, p.Display
	}
	if landscape && portrait {
		return 0, 0, "", fmt.Errorf("--landscape and --portrait are mutually exclusive")
	}
	if landscape && h > w {
		w, h = h, w
	}
	if portrait && w > h {
		w, h = h, w
	}
	return w, h, name, nil
}

// defaultOut derives the output path: the input file with a " (Preset).pdf"
// suffix (e.g. "Wagon Wheel (iPad Mini).pdf"), so exports for different
// devices coexist side by side. Stdin input names the file after the song
// title.
func defaultOut(input string, song *chordpro.Song, sizeName string) string {
	base := strings.TrimSuffix(input, filepath.Ext(input))
	if input == "" {
		base = "song"
		if song.Title != "" {
			base = song.Title
		}
	}
	return base + " (" + sizeName + ").pdf"
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

func listPresets() {
	st := newStyler(os.Stdout)
	fmt.Printf("\n  Page presets %s\n\n", st.dim("(points · 72 pt = 1 inch)"))
	for _, p := range pdf.Presets {
		fmt.Printf("  %s  %s  %s\n",
			st.cyan(fmt.Sprintf("%-10s", p.Name)),
			fmt.Sprintf("%4.0f × %-4.0f", p.W, p.H),
			st.dim(p.Note))
	}
	fmt.Println()
}

func usage() { fmt.Fprint(os.Stderr, usageText()) }

func usageText() string {
	return `chordpro-pdf — export a ChordPro song as a one-page PDF

Usage:
  chordpro-pdf [flags] <song.cho>
  chordpro-pdf < song.cho

Flags:
  -o, --output FILE  output path (default: "<input> (<Preset>).pdf"; "-" = stdout)
  --preset NAME      page preset: ipad-mini, ipad, iphone, mac, a4, letter
  --landscape        turn the page to landscape
  --portrait         turn the page to portrait
  --page WxH         custom page size in points (overrides --preset)
  --transpose N      transpose chords by N semitones
  --columns N        force column count (default: auto)
  --max-font F       largest body font size in points (default 20)
  --margin F         page margin in points (default: 4.5% of the short edge)
  --serif            serif lyrics (Times) instead of sans (Helvetica)
  --inverted         white text on a black page (night mode)
  --no-diagrams      omit the chord fingering diagrams in the header
  --list-presets     list page presets and exit

The song is always laid out on exactly one page: short songs get large,
centered type; long songs flow into balanced columns at smaller sizes.
The header shows fingering diagrams for every chord the song uses.
Device presets match the screen's aspect ratio, so a full-screen PDF
viewer shows the song with no letterboxing.
`
}
