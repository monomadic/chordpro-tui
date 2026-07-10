package render

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette holds the raw colors a Theme is built from. Swapping a Palette is the
// easiest way to reskin the renderer.
type Palette struct {
	Name     string         // display name, e.g. "Mocha"
	Bg       lipgloss.Color // full-screen background (used when bg-fill is on)
	Chord    lipgloss.Color // inline chord names
	ChordBg  lipgloss.Color // background behind each chord pill
	Lyric    lipgloss.Color // sung text
	Title    lipgloss.Color // song title
	Subtitle lipgloss.Color // artist / subtitle
	Section  lipgloss.Color // verse/chorus labels
	Chorus   lipgloss.Color // chorus accent bar
	Comment  lipgloss.Color // inline {comment} annotations
	Tab      lipgloss.Color // monospace tab blocks
	Border   lipgloss.Color // header frame
	PillBg   lipgloss.Color // metadata pill background
	PillFg   lipgloss.Color // metadata pill text
	Muted    lipgloss.Color // footer / hints
}

// CatppuccinMocha is the default dark palette: neon-blue chord pills over soft
// lavender-white lyrics, with a mauve chorus accent.
var CatppuccinMocha = Palette{
	Name:     "Mocha",
	Bg:       lipgloss.Color("#1e1e2e"),
	Chord:    lipgloss.Color("#5cf0ff"), // neon blue
	ChordBg:  lipgloss.Color("#093247"), // deep teal-navy
	Lyric:    lipgloss.Color("#cdd6f4"),
	Title:    lipgloss.Color("#89b4fa"),
	Subtitle: lipgloss.Color("#f9e2af"),
	Section:  lipgloss.Color("#00FF88"),
	Chorus:   lipgloss.Color("#cba6f7"),
	Comment:  lipgloss.Color("#94e2d5"),
	Tab:      lipgloss.Color("#89dceb"),
	Border:   lipgloss.Color("#585b70"),
	PillBg:   lipgloss.Color("#313244"),
	PillFg:   lipgloss.Color("#f5e0dc"),
	Muted:    lipgloss.Color("#6c7086"),
}

// TokyoNight: cool blue night palette with a cyan chord glow.
var TokyoNight = Palette{
	Name:     "Tokyo Night",
	Bg:       lipgloss.Color("#1a1b26"),
	Chord:    lipgloss.Color("#7dcfff"),
	ChordBg:  lipgloss.Color("#1f2a44"),
	Lyric:    lipgloss.Color("#c0caf5"),
	Title:    lipgloss.Color("#bb9af7"),
	Subtitle: lipgloss.Color("#7aa2f7"),
	Section:  lipgloss.Color("#9ece6a"),
	Chorus:   lipgloss.Color("#ff9e64"),
	Comment:  lipgloss.Color("#73daca"),
	Tab:      lipgloss.Color("#2ac3de"),
	Border:   lipgloss.Color("#3b4261"),
	PillBg:   lipgloss.Color("#24283b"),
	PillFg:   lipgloss.Color("#c0caf5"),
	Muted:    lipgloss.Color("#565f89"),
}

// Gruvbox: warm retro palette, amber chords on a dark brown pill.
var Gruvbox = Palette{
	Name:     "Gruvbox",
	Bg:       lipgloss.Color("#282828"),
	Chord:    lipgloss.Color("#fe8019"),
	ChordBg:  lipgloss.Color("#3c2a21"),
	Lyric:    lipgloss.Color("#ebdbb2"),
	Title:    lipgloss.Color("#fabd2f"),
	Subtitle: lipgloss.Color("#83a598"),
	Section:  lipgloss.Color("#b8bb26"),
	Chorus:   lipgloss.Color("#d3869b"),
	Comment:  lipgloss.Color("#8ec07c"),
	Tab:      lipgloss.Color("#83a598"),
	Border:   lipgloss.Color("#504945"),
	PillBg:   lipgloss.Color("#3c3836"),
	PillFg:   lipgloss.Color("#ebdbb2"),
	Muted:    lipgloss.Color("#928374"),
}

// Dracula: vivid purple-and-pink dark palette.
var Dracula = Palette{
	Name:     "Dracula",
	Bg:       lipgloss.Color("#282a36"),
	Chord:    lipgloss.Color("#8be9fd"),
	ChordBg:  lipgloss.Color("#22243a"),
	Lyric:    lipgloss.Color("#f8f8f2"),
	Title:    lipgloss.Color("#bd93f9"),
	Subtitle: lipgloss.Color("#ffb86c"),
	Section:  lipgloss.Color("#50fa7b"),
	Chorus:   lipgloss.Color("#ff79c6"),
	Comment:  lipgloss.Color("#8be9fd"),
	Tab:      lipgloss.Color("#f1fa8c"),
	Border:   lipgloss.Color("#44475a"),
	PillBg:   lipgloss.Color("#343746"),
	PillFg:   lipgloss.Color("#f8f8f2"),
	Muted:    lipgloss.Color("#6272a4"),
}

// Nord: muted arctic blues with a frost chord pill.
var Nord = Palette{
	Name:     "Nord",
	Bg:       lipgloss.Color("#2e3440"),
	Chord:    lipgloss.Color("#88c0d0"),
	ChordBg:  lipgloss.Color("#2e3440"),
	Lyric:    lipgloss.Color("#eceff4"),
	Title:    lipgloss.Color("#81a1c1"),
	Subtitle: lipgloss.Color("#ebcb8b"),
	Section:  lipgloss.Color("#a3be8c"),
	Chorus:   lipgloss.Color("#b48ead"),
	Comment:  lipgloss.Color("#8fbcbb"),
	Tab:      lipgloss.Color("#81a1c1"),
	Border:   lipgloss.Color("#434c5e"),
	PillBg:   lipgloss.Color("#3b4252"),
	PillFg:   lipgloss.Color("#e5e9f0"),
	Muted:    lipgloss.Color("#616e88"),
}

// Synthwave: outrun neon — cyan chords and hot-magenta accents on deep indigo.
var Synthwave = Palette{
	Name:     "Synthwave",
	Bg:       lipgloss.Color("#190a2a"),
	Chord:    lipgloss.Color("#00eaff"),
	ChordBg:  lipgloss.Color("#2a0e5a"),
	Lyric:    lipgloss.Color("#ece3ff"),
	Title:    lipgloss.Color("#ff3caf"),
	Subtitle: lipgloss.Color("#00eaff"),
	Section:  lipgloss.Color("#ffd000"),
	Chorus:   lipgloss.Color("#b14bff"),
	Comment:  lipgloss.Color("#4dffd1"),
	Tab:      lipgloss.Color("#00eaff"),
	Border:   lipgloss.Color("#ff3caf"),
	PillBg:   lipgloss.Color("#2a0e5a"),
	PillFg:   lipgloss.Color("#ffffff"),
	Muted:    lipgloss.Color("#8a6bc4"),
}

// Cyberpunk: electric yellow chords with magenta/cyan accents on near-black.
var Cyberpunk = Palette{
	Name:     "Cyberpunk",
	Bg:       lipgloss.Color("#0a0a12"),
	Chord:    lipgloss.Color("#f6ff00"),
	ChordBg:  lipgloss.Color("#1b1b00"),
	Lyric:    lipgloss.Color("#e8faff"),
	Title:    lipgloss.Color("#ff007a"),
	Subtitle: lipgloss.Color("#00f0ff"),
	Section:  lipgloss.Color("#00ffa3"),
	Chorus:   lipgloss.Color("#ff007a"),
	Comment:  lipgloss.Color("#c77dff"),
	Tab:      lipgloss.Color("#00f0ff"),
	Border:   lipgloss.Color("#f6ff00"),
	PillBg:   lipgloss.Color("#14141f"),
	PillFg:   lipgloss.Color("#f6ff00"),
	Muted:    lipgloss.Color("#5a5a72"),
}

// Laser: acid-lime chords against magenta and teal on dark green-black.
var Laser = Palette{
	Name:     "Laser",
	Bg:       lipgloss.Color("#08160a"),
	Chord:    lipgloss.Color("#aaff00"),
	ChordBg:  lipgloss.Color("#0d2600"),
	Lyric:    lipgloss.Color("#eafff0"),
	Title:    lipgloss.Color("#ff00e6"),
	Subtitle: lipgloss.Color("#00ffcc"),
	Section:  lipgloss.Color("#ffe600"),
	Chorus:   lipgloss.Color("#ff00e6"),
	Comment:  lipgloss.Color("#00ffcc"),
	Tab:      lipgloss.Color("#aaff00"),
	Border:   lipgloss.Color("#ff00e6"),
	PillBg:   lipgloss.Color("#122100"),
	PillFg:   lipgloss.Color("#eafff0"),
	Muted:    lipgloss.Color("#6f8a5c"),
}

// Vapor: pastel-neon vaporwave — pink and aqua on twilight purple.
var Vapor = Palette{
	Name:     "Vapor",
	Bg:       lipgloss.Color("#1c1230"),
	Chord:    lipgloss.Color("#ff6ad5"),
	ChordBg:  lipgloss.Color("#241734"),
	Lyric:    lipgloss.Color("#f4e9ff"),
	Title:    lipgloss.Color("#8c9eff"),
	Subtitle: lipgloss.Color("#94d0ff"),
	Section:  lipgloss.Color("#a0ffe6"),
	Chorus:   lipgloss.Color("#ff6ad5"),
	Comment:  lipgloss.Color("#c8a2ff"),
	Tab:      lipgloss.Color("#94d0ff"),
	Border:   lipgloss.Color("#c47fd5"),
	PillBg:   lipgloss.Color("#241734"),
	PillFg:   lipgloss.Color("#f4e9ff"),
	Muted:    lipgloss.Color("#7c6f9c"),
}

// Palettes is the ordered set the UI cycles through.
var Palettes = []Palette{
	CatppuccinMocha, TokyoNight, Gruvbox, Dracula, Nord,
	Synthwave, Cyberpunk, Laser, Vapor,
}

// Theme is the set of compiled lipgloss styles used while rendering.
type Theme struct {
	Name string
	P    Palette

	Chord      lipgloss.Style
	Annotation lipgloss.Style
	Lyric      lipgloss.Style
	Title      lipgloss.Style
	Subtitle   lipgloss.Style
	Section    lipgloss.Style
	Comment    lipgloss.Style
	Tab        lipgloss.Style
	Frame      lipgloss.Style
	PillKey    lipgloss.Style
	PillVal    lipgloss.Style
	ChorusBar  lipgloss.Style
	Muted      lipgloss.Style
}

// NewTheme compiles a Palette into ready-to-use styles.
func NewTheme(p Palette) *Theme {
	return &Theme{
		Name:       p.Name,
		P:          p,
		Chord:      lipgloss.NewStyle().Foreground(p.Chord).Background(p.ChordBg).Bold(true),
		Annotation: lipgloss.NewStyle().Foreground(p.Comment).Italic(true),
		Lyric:      lipgloss.NewStyle().Foreground(p.Lyric),
		Title:      lipgloss.NewStyle().Foreground(p.Title).Bold(true),
		Subtitle:   lipgloss.NewStyle().Foreground(p.Subtitle).Italic(true),
		Section:    lipgloss.NewStyle().Foreground(p.Section).Bold(true).Italic(true),
		Comment:    lipgloss.NewStyle().Foreground(p.Comment).Italic(true),
		Tab:        lipgloss.NewStyle().Foreground(p.Tab),
		Frame:      lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.Border).Padding(0, 2),
		PillKey:    lipgloss.NewStyle().Background(p.PillBg).Foreground(p.Section).Bold(true).Padding(0, 1),
		PillVal:    lipgloss.NewStyle().Background(lighten(p.PillBg, 0.12)).Foreground(p.PillFg).Padding(0, 1),
		ChorusBar:  lipgloss.NewStyle().Foreground(p.Chorus),
		Muted:      lipgloss.NewStyle().Foreground(p.Muted),
	}
}

// lighten blends c a fraction amt toward white (0 = unchanged, 1 = white). It
// gives metadata pill values a slightly brighter fill than their labels. A
// non-hex color is returned unchanged.
func lighten(c lipgloss.Color, amt float64) lipgloss.Color {
	r, g, b, ok := hexRGB(string(c))
	if !ok {
		return c
	}
	blend := func(v int) int { return v + int(float64(255-v)*amt) }
	return lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", blend(r), blend(g), blend(b)))
}

// DefaultTheme returns the renderer's default theme.
func DefaultTheme() *Theme { return NewTheme(CatppuccinMocha) }

// Themes compiles every palette in Palettes into a ready-to-use Theme slice,
// in cycle order.
func Themes() []*Theme {
	out := make([]*Theme, len(Palettes))
	for i, p := range Palettes {
		out[i] = NewTheme(p)
	}
	return out
}

// ThemeIndexByName returns the index of the palette whose name matches (case-
// and space-insensitive), or -1 if none.
func ThemeIndexByName(name string) int {
	want := strings.ToLower(strings.TrimSpace(name))
	if want == "" {
		return -1
	}
	for i, p := range Palettes {
		if strings.ToLower(p.Name) == want {
			return i
		}
	}
	return -1
}

// ThemeByName returns the named theme, or the default if not found.
func ThemeByName(name string) *Theme {
	if i := ThemeIndexByName(name); i >= 0 {
		return NewTheme(Palettes[i])
	}
	return DefaultTheme()
}
