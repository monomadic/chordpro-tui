package tui

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"chordpro-tui/internal/config"
)

func TestOrderedChordPaths(t *testing.T) {
	dir := t.TempDir()
	// Filenames sort a,b,c; titles sort Apple,Mango,Zebra; write a first but make
	// it the oldest so date order differs from both.
	files := []struct {
		name, title string
		age         time.Duration
	}{
		{"c.cho", "Apple", 3 * time.Hour},
		{"a.cho", "Zebra", 1 * time.Hour},
		{"b.cho", "Mango", 2 * time.Hour},
	}
	now := time.Now()
	for _, f := range files {
		p := filepath.Join(dir, f.name)
		if err := os.WriteFile(p, []byte("{title: "+f.title+"}\n[C]x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		mt := now.Add(-f.age)
		if err := os.Chtimes(p, mt, mt); err != nil {
			t.Fatal(err)
		}
	}

	base := func(paths []string) []string {
		out := make([]string, len(paths))
		for i, p := range paths {
			out[i] = filepath.Base(p)
		}
		return out
	}

	cases := []struct {
		mode config.SortMode
		want []string
	}{
		{config.SortNone, []string{"a.cho", "b.cho", "c.cho"}}, // filename order
		{config.SortName, []string{"c.cho", "b.cho", "a.cho"}}, // Apple, Mango, Zebra
		{config.SortDate, []string{"a.cho", "b.cho", "c.cho"}}, // newest (1h) → oldest (3h)
	}
	for _, tc := range cases {
		got := base(orderedChordPaths(dir, tc.mode))
		if len(got) != len(tc.want) {
			t.Fatalf("mode %d: got %v, want %v", tc.mode, got, tc.want)
		}
		for i := range tc.want {
			if got[i] != tc.want[i] {
				t.Errorf("mode %d: order = %v, want %v", tc.mode, got, tc.want)
				break
			}
		}
	}
}

func TestPickerUnfilteredHonorsSort(t *testing.T) {
	dir := t.TempDir()
	write := func(name, title string) {
		if err := os.WriteFile(filepath.Join(dir, name), []byte("{title: "+title+"}\n[C]x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("a.cho", "Zebra")
	write("b.cho", "Apple")

	p := newPicker(dir, "", config.SortName)
	sel, ok := p.selected()
	if !ok {
		t.Fatal("no selection")
	}
	// With title sort and no query, "Apple" (b.cho) leads the list.
	if filepath.Base(sel) != "b.cho" {
		t.Errorf("title-sorted picker leads with %q, want b.cho", filepath.Base(sel))
	}
}
