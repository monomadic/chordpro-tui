package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"chordpro-tui/internal/chordpro"
)

func TestSaveTransposed(t *testing.T) {
	dir := t.TempDir()
	src := "{title: Stolen Car}\n{key: C}\n[C]hello [G]world\n"
	path := filepath.Join(dir, "Stolen Car - Beth Orton.cho")
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	s, _ := chordpro.ParseString(src)
	m := New(s, Options{Transpose: 2, Path: path})

	dst, err := m.saveTransposed()
	if err != nil {
		t.Fatalf("saveTransposed: %v", err)
	}
	want := filepath.Join(dir, "Stolen Car - Beth Orton (Alternate Tuning +2).cho")
	if dst != want {
		t.Errorf("path = %q, want %q", dst, want)
	}

	b, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("written file unreadable: %v", err)
	}
	got := string(b)
	for _, w := range []string{
		"{title: Stolen Car (Alternate Tuning: +2)}",
		"{key: D}",
		"[D]hello [A]world",
	} {
		if !strings.Contains(got, w) {
			t.Errorf("saved file missing %q\n---\n%s", w, got)
		}
	}
}

func TestSaveTransposedGuards(t *testing.T) {
	s, _ := chordpro.ParseString("{title: T}\n{key: C}\n[C]x\n")

	// No source path → can't save.
	if _, err := New(s, Options{Transpose: 2}).saveTransposed(); err == nil {
		t.Error("expected an error when there is no source file")
	}

	// Not transposed → nothing to save.
	dir := t.TempDir()
	p := filepath.Join(dir, "a.cho")
	if err := os.WriteFile(p, []byte("{title: T}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := New(s, Options{Path: p}).saveTransposed(); err == nil {
		t.Error("expected an error when the song is not transposed")
	}
}

func TestAlternatePath(t *testing.T) {
	got := alternatePath("/songs/Stolen Car.cho", 1)
	want := "/songs/Stolen Car (Alternate Tuning +1).cho"
	if got != want {
		t.Errorf("alternatePath = %q, want %q", got, want)
	}
	if got := alternatePath("/songs/x.cho", -3); !strings.HasSuffix(got, "x (Alternate Tuning -3).cho") {
		t.Errorf("alternatePath negative = %q", got)
	}
}
