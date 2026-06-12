package render

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Palette holds the raw colors a Theme is built from. Swapping a Palette is the
// easiest way to reskin the renderer.
type Palette struct {
	Name     string         // display name, e.g. "Mocha"
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
	Chord:    lipgloss.Color("#5cf0ff"), // neon blue
	ChordBg:  lipgloss.Color("#093247"), // deep teal-navy
	Lyric:    lipgloss.Color("#cdd6f4"),
	Title:    lipgloss.Color("#89b4fa"),
	Subtitle: lipgloss.Color("#f9e2af"),
	Section:  lipgloss.Color("#a6e3a1"),
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

// Palettes is the ordered set the UI cycles through.
var Palettes = []Palette{CatppuccinMocha, TokyoNight, Gruvbox, Dracula, Nord}

// Theme is the set of compiled lipgloss styles used while rendering.
type Theme struct {
	Name string
	P    Palette

	Chord    lipgloss.Style
	Lyric    lipgloss.Style
	Title    lipgloss.Style
	Subtitle lipgloss.Style
	Section  lipgloss.Style
	Comment  lipgloss.Style
	Tab      lipgloss.Style
	Frame    lipgloss.Style
	PillKey  lipgloss.Style
	PillVal  lipgloss.Style
	ChorusBar lipgloss.Style
	Muted    lipgloss.Style
}

// NewTheme compiles a Palette into ready-to-use styles.
func NewTheme(p Palette) *Theme {
	return &Theme{
		Name:      p.Name,
		P:         p,
		Chord:     lipgloss.NewStyle().Foreground(p.Chord).Background(p.ChordBg).Bold(true),
		Lyric:     lipgloss.NewStyle().Foreground(p.Lyric),
		Title:     lipgloss.NewStyle().Foreground(p.Title).Bold(true),
		Subtitle:  lipgloss.NewStyle().Foreground(p.Subtitle).Italic(true),
		Section:   lipgloss.NewStyle().Foreground(p.Section).Bold(true).Italic(true),
		Comment:   lipgloss.NewStyle().Foreground(p.Comment).Italic(true),
		Tab:       lipgloss.NewStyle().Foreground(p.Tab),
		Frame:     lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(p.Border).Padding(0, 2),
		PillKey:   lipgloss.NewStyle().Background(p.PillBg).Foreground(p.Section).Bold(true).Padding(0, 1),
		PillVal:   lipgloss.NewStyle().Background(p.PillBg).Foreground(p.PillFg).Padding(0, 1),
		ChorusBar: lipgloss.NewStyle().Foreground(p.Chorus),
		Muted:     lipgloss.NewStyle().Foreground(p.Muted),
	}
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
