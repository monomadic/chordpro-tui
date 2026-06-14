package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"chordpro-tui/internal/chordpro"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewestSong(t *testing.T) {
	dir := t.TempDir()
	a := filepath.Join(dir, "a.cho")
	b := filepath.Join(dir, "b.cho")
	if err := os.WriteFile(a, []byte("{title: A}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(b, []byte("{title: B}"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(a, old, old); err != nil {
		t.Fatal(err)
	}
	got, err := NewestSong(dir)
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "b.cho" {
		t.Errorf("NewestSong = %q, want b.cho", filepath.Base(got))
	}
	if _, err := NewestSong(t.TempDir()); err == nil {
		t.Error("expected an error for a directory with no songs")
	}
}

func TestFuzzyMatch(t *testing.T) {
	if _, _, ok := fuzzyMatch("wgn", "wagon_wheel.cho"); !ok {
		t.Error("wgn should match wagon_wheel")
	}
	if _, pos, ok := fuzzyMatch("wh", "wagon_wheel.cho"); !ok || len(pos) != 2 {
		t.Errorf("wh match pos = %v ok=%v", pos, ok)
	}
	if _, _, ok := fuzzyMatch("xyz", "wagon_wheel.cho"); ok {
		t.Error("xyz should not match")
	}
	// Empty query matches anything.
	if _, _, ok := fuzzyMatch("", "anything"); !ok {
		t.Error("empty query should match")
	}
}

func TestFuzzyRanksBoundaryHigher(t *testing.T) {
	// "rs" should score higher on "rising_sun" (two word starts) than as a
	// mid-word subsequence.
	hi, _, _ := fuzzyMatch("rs", "rising_sun.cho")
	lo, _, _ := fuzzyMatch("rs", "characters.cho")
	if hi <= lo {
		t.Errorf("expected boundary match to rank higher: %d vs %d", hi, lo)
	}
}

func TestNewPickerScansChordFiles(t *testing.T) {
	p := newPicker("../../testdata", "../../testdata/wagon_wheel.cho")
	if len(p.entries) < 4 {
		t.Fatalf("expected >=4 chord files, got %d", len(p.entries))
	}
	for _, e := range p.entries {
		if e.title == "" {
			t.Errorf("entry %q has no title", e.path)
		}
		if strings.Contains(e.title, ".cho") {
			t.Errorf("title still contains a file extension: %q", e.title)
		}
	}
	// Cursor should start on the current song.
	if sel, ok := p.selected(); !ok || !strings.HasSuffix(sel, "wagon_wheel.cho") {
		t.Errorf("cursor not on current song: %q", sel)
	}
}

func TestSongMeta(t *testing.T) {
	s, _ := chordpro.ParseString("{key: G}\n{capo: 2}\n{tempo: 76}\n{year: 2004}\n")
	if got := songMeta(s); got != "G · capo 2 · 76 bpm · 2004" {
		t.Errorf("songMeta = %q", got)
	}
	s2, _ := chordpro.ParseString("{tempo: 120}\n")
	if got := songMeta(s2); got != "120 bpm" {
		t.Errorf("songMeta(tempo only) = %q", got)
	}
	s3, _ := chordpro.ParseString("{title: X}\n")
	if got := songMeta(s3); got != "" {
		t.Errorf("songMeta(no meta) = %q, want empty", got)
	}
}

func TestFitTokens(t *testing.T) {
	full := "G · capo 2 · 76 bpm · 2004"
	if got := fitTokens(full, 100); got != full {
		t.Errorf("wide fit dropped tokens: %q", got)
	}
	if got := fitTokens(full, 10); got != "G · capo 2" {
		t.Errorf("fitTokens(10) = %q, want 'G · capo 2'", got)
	}
	if got := fitTokens(full, 1); got != "G" {
		t.Errorf("fitTokens(1) = %q, want 'G'", got)
	}
}

func TestPickerEntriesCarryMeta(t *testing.T) {
	p := newPicker("../../testdata", "")
	found := false
	for _, e := range p.entries {
		if strings.Contains(e.meta, "bpm") {
			found = true
		}
	}
	if !found {
		t.Error("expected at least one entry with tempo metadata")
	}
}

func TestPickerFilterAndSelect(t *testing.T) {
	p := newPicker("../../testdata", "")
	p.setQuery("blow")
	sel, ok := p.selected()
	if !ok || !strings.Contains(sel, "blowin") {
		t.Errorf("filter 'blow' selected %q (ok=%v)", sel, ok)
	}
	if len(p.matches) != 1 {
		t.Errorf("expected 1 match for 'blow', got %d", len(p.matches))
	}
}

func TestOpenPickerAndLoadSwitchesSong(t *testing.T) {
	s, err := chordpro.ParseString("{title: Start}\n{key: C}\n[C]hi\n")
	if err != nil {
		t.Fatal(err)
	}
	m := resize(New(s, Options{Path: "../../testdata/wagon_wheel.cho"}), 100, 30)

	// Open the picker.
	nm, _ := m.handleKey(key("o"))
	m = nm.(Model)
	if !m.picking {
		t.Fatal("'o' did not open the picker")
	}

	// Filter to a different song and open it.
	for _, r := range "house" {
		nm, _ = m.handleKey(key(string(r)))
		m = nm.(Model)
	}
	nm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEnter})
	m = nm.(Model)

	if m.picking {
		t.Error("picker still open after enter")
	}
	if m.base.Title != "House of the Rising Sun" {
		t.Errorf("song did not switch, title = %q", m.base.Title)
	}
	if m.transp != 0 {
		t.Errorf("transpose not reset after load: %d", m.transp)
	}
}

func TestPickerEscCancels(t *testing.T) {
	s, _ := chordpro.ParseString("{title: X}\n")
	m := resize(New(s, Options{Path: "../../testdata/wagon_wheel.cho"}), 80, 24)
	nm, _ := m.handleKey(key("o"))
	m = nm.(Model)
	nm, _ = m.handleKey(tea.KeyMsg{Type: tea.KeyEsc})
	m = nm.(Model)
	if m.picking {
		t.Error("esc did not close the picker")
	}
	if m.base.Title != "X" {
		t.Error("esc should not change the song")
	}
}
