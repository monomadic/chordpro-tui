package config

import (
	"reflect"
	"strings"
	"testing"

	"chordpro-tui/internal/render"
)

func TestDefaultIsZeroBehaviour(t *testing.T) {
	if got := Default().RenderOpts(); got != (render.RenderOpts{}) {
		t.Errorf("default RenderOpts = %+v, want zero value", got)
	}
}

func TestMarshalRoundTrips(t *testing.T) {
	want := Config{
		CollapseTablatureSections: render.Auto,
		AutohideSongTitle:         render.On,
		AutohideSongInfo:          render.Off,
		AutohideSectionTitles:     true,
		CollapsePageTitle:         render.Auto,
		CollapseSectionTitle:      render.On,
		SortSongs:                 SortDate,
	}
	got, err := Parse(want.Marshal())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("round-trip mismatch:\n got %+v\nwant %+v", got, want)
	}
}

func TestDefaultMarshalRoundTrips(t *testing.T) {
	got, err := Parse(Default().Marshal())
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got != Default() {
		t.Errorf("default round-trip = %+v, want %+v", got, Default())
	}
}

func TestParseAcceptsBareAndQuotedValues(t *testing.T) {
	src := `
# a comment
collapse-page-title = auto      # inline comment
autohide-song-title = "true"
sort-songs = name
autohide-section-titles = true
`
	c, err := Parse(src)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if c.CollapsePageTitle != render.Auto {
		t.Errorf("collapse-page-title = %v, want Auto", c.CollapsePageTitle)
	}
	if c.AutohideSongTitle != render.On {
		t.Errorf("autohide-song-title = %v, want On", c.AutohideSongTitle)
	}
	if c.SortSongs != SortName {
		t.Errorf("sort-songs = %v, want SortName", c.SortSongs)
	}
	if !c.AutohideSectionTitles {
		t.Error("autohide-section-titles should be true")
	}
}

func TestParseRejectsUnknownKeyAndValue(t *testing.T) {
	cases := []string{
		"nonsense = true",
		"sort-songs = sideways",
		"collapse-page-title = maybe",
		"autohide-section-titles = auto", // auto not valid for a plain bool
	}
	for _, src := range cases {
		if _, err := Parse(src); err == nil {
			t.Errorf("Parse(%q) = nil error, want a parse error", src)
		}
	}
}

func TestParseErrorReportsLine(t *testing.T) {
	_, err := Parse("collapse-page-title = auto\nbogus = 1\n")
	if err == nil || !strings.Contains(err.Error(), "line 2") {
		t.Errorf("error = %v, want it to mention line 2", err)
	}
}
