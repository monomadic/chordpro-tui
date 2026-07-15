// Package config loads chordpro-tui's optional TOML settings file: display
// options fed to the renderer, plus the song-queue sort order. The file is a
// flat set of `key = value` lines (a small TOML subset — no tables or arrays),
// so parsing is hand-rolled and dependency-free.
package config

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"chordpro-tui/internal/render"
)

// FileName is the base name looked for in the working and config directories.
const FileName = "chordpro-tui.toml"

// SortMode is the ordering applied to the song queue (next/prev navigation and
// the open-song picker's unfiltered listing).
type SortMode int

const (
	SortNone SortMode = iota // directory order (filename)
	SortName                 // by song title
	SortDate                 // by modification time, newest first
)

// Config is the resolved settings. Zero value == every option off / SortNone,
// which reproduces the default (un-collapsed, un-hidden) rendering.
type Config struct {
	CollapseTablatureSections render.Tri // fold tab sections
	AutohideSongTitle         render.Tri // hide the title + artist line
	AutohideSongInfo          render.Tri // hide the metadata pills (KEY, CAPO, …)
	AutohideSectionTitles     bool       // drop section labels (CHORUS, VERSE, …)
	CollapsePageTitle         render.Tri // lay title, artist, and metadata on one line
	CollapseSectionTitle      render.Tri // blank row above each section label
	SortSongs                 SortMode   // song-queue ordering
}

// Default returns the built-in configuration (behaviour identical to running
// with no config file).
func Default() Config { return Config{} }

// RenderOpts translates the display-related settings into renderer options.
// The caller merges the live runtime toggles (HideHeader/HideTabs/ViewMode) on
// top of the returned value.
func (c Config) RenderOpts() render.RenderOpts {
	return render.RenderOpts{
		CollapseTabs:      c.CollapseTablatureSections,
		HideTitle:         c.AutohideSongTitle,
		HideInfo:          c.AutohideSongInfo,
		HideSectionTitles: c.AutohideSectionTitles,
		CollapsePageTitle: c.CollapsePageTitle,
		SectionTitleGap:   c.CollapseSectionTitle,
	}
}

// Load resolves the configuration. When explicit is non-empty it is loaded
// directly (a missing file is an error). Otherwise the working directory and
// then the user config directory are searched for FileName; if neither exists
// the default config is returned with an empty path. The returned path is the
// file that was read (empty when defaults were used).
func Load(explicit string) (Config, string, error) {
	if explicit != "" {
		c, err := loadFile(explicit)
		return c, explicit, err
	}
	for _, p := range searchPaths() {
		if _, err := os.Stat(p); err == nil {
			c, err := loadFile(p)
			return c, p, err
		}
	}
	return Default(), "", nil
}

// searchPaths lists candidate config locations in precedence order: the working
// directory first (project-local overrides), then the XDG config directory
// (~/.config on Unix and, deliberately, on macOS too — where terminal tools are
// conventionally configured), then the OS-native config dir as a fallback (on
// macOS ~/Library/Application Support, which is what os.UserConfigDir returns).
func searchPaths() []string {
	paths := []string{FileName}
	if p := UserConfigPath(); p != "" {
		paths = append(paths, p)
	}
	if dir, err := os.UserConfigDir(); err == nil {
		p := filepath.Join(dir, "chordpro-tui", FileName)
		if !contains(paths, p) {
			paths = append(paths, p)
		}
	}
	return paths
}

// ConfigDir is the directory --init-config writes to and Load reads from first:
// $XDG_CONFIG_HOME/chordpro-tui when set, else ~/.config/chordpro-tui. Returns
// "" only if the home directory can't be determined.
func ConfigDir() string {
	if x := os.Getenv("XDG_CONFIG_HOME"); x != "" {
		return filepath.Join(x, "chordpro-tui")
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "chordpro-tui")
	}
	return ""
}

// UserConfigPath is the full path to the user-wide config file (ConfigDir plus
// FileName), or "" if the home directory can't be determined.
func UserConfigPath() string {
	if d := ConfigDir(); d != "" {
		return filepath.Join(d, FileName)
	}
	return ""
}

// WriteDefault writes the documented default config to UserConfigPath, creating
// the directory. Without force it refuses to overwrite an existing file. It
// returns the path (even on the "already exists" error, so callers can report
// it).
func WriteDefault(force bool) (string, error) {
	path := UserConfigPath()
	if path == "" {
		return "", fmt.Errorf("cannot determine config directory")
	}
	if !force {
		if _, err := os.Stat(path); err == nil {
			return path, fmt.Errorf("%s already exists", path)
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return path, err
	}
	if err := os.WriteFile(path, []byte(Default().Marshal()), 0o644); err != nil {
		return path, err
	}
	return path, nil
}

func contains(xs []string, x string) bool {
	for _, s := range xs {
		if s == x {
			return true
		}
	}
	return false
}

func loadFile(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Default(), err
	}
	return Parse(string(data))
}

// Parse reads the flat key/value settings from src. Unknown keys are an error
// so typos surface rather than being silently ignored.
func Parse(src string) (Config, error) {
	c := Default()
	sc := bufio.NewScanner(strings.NewReader(src))
	line := 0
	for sc.Scan() {
		line++
		raw := strings.TrimSpace(stripComment(sc.Text()))
		if raw == "" {
			continue
		}
		k, v, ok := strings.Cut(raw, "=")
		if !ok {
			return c, fmt.Errorf("line %d: expected key = value, got %q", line, raw)
		}
		key := strings.TrimSpace(k)
		val := unquote(strings.TrimSpace(v))
		if err := c.set(key, val); err != nil {
			return c, fmt.Errorf("line %d: %w", line, err)
		}
	}
	return c, sc.Err()
}

func (c *Config) set(key, val string) error {
	switch key {
	case "collapse-tablature-sections":
		return setTri(&c.CollapseTablatureSections, key, val)
	case "autohide-song-title":
		return setTri(&c.AutohideSongTitle, key, val)
	case "autohide-song-info":
		return setTri(&c.AutohideSongInfo, key, val)
	case "autohide-section-titles":
		return setBool(&c.AutohideSectionTitles, key, val)
	case "collapse-page-title":
		return setTri(&c.CollapsePageTitle, key, val)
	case "collapse-section-title":
		return setTri(&c.CollapseSectionTitle, key, val)
	case "sort-songs":
		return c.setSort(val)
	default:
		return fmt.Errorf("unknown key %q", key)
	}
}

func setTri(dst *render.Tri, key, val string) error {
	switch strings.ToLower(val) {
	case "false", "off":
		*dst = render.Off
	case "true", "on":
		*dst = render.On
	case "auto":
		*dst = render.Auto
	default:
		return fmt.Errorf("%s: want true, false, or auto, got %q", key, val)
	}
	return nil
}

func setBool(dst *bool, key, val string) error {
	switch strings.ToLower(val) {
	case "true", "on":
		*dst = true
	case "false", "off":
		*dst = false
	default:
		return fmt.Errorf("%s: want true or false, got %q", key, val)
	}
	return nil
}

func (c *Config) setSort(val string) error {
	switch strings.ToLower(val) {
	case "none":
		c.SortSongs = SortNone
	case "name":
		c.SortSongs = SortName
	case "date":
		c.SortSongs = SortDate
	default:
		return fmt.Errorf("sort-songs: want none, name, or date, got %q", val)
	}
	return nil
}

// stripComment removes a trailing `#` comment, ignoring `#` inside quotes.
func stripComment(s string) string {
	inQuote := false
	for i, r := range s {
		switch r {
		case '"':
			inQuote = !inQuote
		case '#':
			if !inQuote {
				return s[:i]
			}
		}
	}
	return s
}

// unquote strips a matched pair of surrounding double quotes, if present.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '"' && s[len(s)-1] == '"' {
		return s[1 : len(s)-1]
	}
	return s
}

func triString(t render.Tri) string {
	switch t {
	case render.On:
		return "true"
	case render.Auto:
		return "auto"
	default:
		return "false"
	}
}

func sortString(m SortMode) string {
	switch m {
	case SortName:
		return "name"
	case SortDate:
		return "date"
	default:
		return "none"
	}
}
