package pdf

import "strings"

// Preset is a named page geometry in PDF points (1/72 inch), in its natural
// orientation. Device presets use the device's logical point resolution, so
// the page aspect ratio matches the screen exactly: a full-screen PDF viewer
// shows the song edge-to-edge with no letterboxing. Display is the
// human-readable name used in output filenames, e.g. "Song (iPad Mini).pdf".
type Preset struct {
	Name    string
	Display string
	W, H    float64
	Note    string
}

// Presets are the built-in page geometries, in display order.
var Presets = []Preset{
	{"ipad-mini", "iPad Mini", 744, 1133, "iPad mini 6/7 screen, portrait"},
	{"ipad", "iPad", 834, 1194, "iPad Air / 11\" Pro screen, portrait"},
	{"iphone", "iPhone", 393, 852, "iPhone 15/16 Pro screen, portrait"},
	{"mac", "Mac", 1440, 900, "16:10 Mac display, full-screen landscape"},
	{"a4", "A4", 595.28, 841.89, "A4 paper, portrait"},
	{"letter", "Letter", 612, 792, "US Letter paper, portrait"},
}

// PresetByName looks up a preset case-insensitively.
func PresetByName(name string) (Preset, bool) {
	for _, p := range Presets {
		if strings.EqualFold(p.Name, name) {
			return p, true
		}
	}
	return Preset{}, false
}
