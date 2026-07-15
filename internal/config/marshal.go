package config

import (
	"fmt"
	"strings"
)

// Marshal renders the configuration as a documented TOML file — the form
// printed by `--print-config`. It round-trips through Parse.
func (c Config) Marshal() string {
	var b strings.Builder
	b.WriteString("# chordpro-tui configuration\n")
	b.WriteString("#\n")
	b.WriteString("# Save as ./" + FileName + " (project-local) or in your config\n")
	b.WriteString("# directory, e.g. ~/.config/chordpro-tui/" + FileName + ".\n")
	b.WriteString("#\n")
	b.WriteString("# Tri-state options accept: true (always), false (never), auto (only when\n")
	b.WriteString("# the song would otherwise overflow the screen). auto applies in fit view;\n")
	b.WriteString("# scroll and player views always have room, so auto reads as false there.\n\n")

	tri := func(key, val string, comments ...string) {
		for _, c := range comments {
			b.WriteString("# " + c + "\n")
		}
		fmt.Fprintf(&b, "%s = %q\n\n", key, val)
	}
	boolean := func(key string, val bool, comments ...string) {
		for _, c := range comments {
			b.WriteString("# " + c + "\n")
		}
		fmt.Fprintf(&b, "%s = %t\n\n", key, val)
	}

	tri("collapse-tablature-sections", triString(c.CollapseTablatureSections),
		"Fold away tab (tablature) sections.")
	tri("autohide-song-title", triString(c.AutohideSongTitle),
		"Hide the title and artist line at the top of the page.")
	tri("autohide-song-info", triString(c.AutohideSongInfo),
		"Hide the metadata pills (KEY, CAPO, TEMPO, …).")
	boolean("autohide-section-titles", c.AutohideSectionTitles,
		"Hide all section labels (CHORUS, VERSE, …).")
	tri("collapse-page-title", triString(c.CollapsePageTitle),
		"Lay the title, artist, and metadata out on a single line.")
	tri("collapse-section-title", triString(c.CollapseSectionTitle),
		"Add a blank row above each section label (true), never (false),",
		"or drop it only when the layout is cramped (auto).")

	b.WriteString("# Song-queue order for next/previous navigation and the open-song list.\n")
	b.WriteString("# One of: none (directory order), name (by title), date (newest first).\n")
	fmt.Fprintf(&b, "sort-songs = %q\n", sortString(c.SortSongs))

	return b.String()
}
