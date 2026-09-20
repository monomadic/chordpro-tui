package chords

import "testing"

func TestIdentify(t *testing.T) {
	for _, tc := range []struct {
		name  string
		frets [6]int
		want  string
	}{
		{"Am", [6]int{-1, 0, 2, 2, 1, 0}, "Am"},
		{"C", [6]int{-1, 3, 2, 0, 1, 0}, "C"},
		{"inversion", [6]int{0, 3, 2, 0, 1, 0}, "C/E"},
		{"G7", [6]int{3, 2, 0, 0, 0, 1}, "G7"},
		{"high Am", [6]int{-1, 12, 14, 14, 13, 12}, "Am"},
		{"actual lowest pitch", [6]int{12, 3, 2, 0, 1, 0}, "C"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got := Identify(tc.frets)
			if len(got.Names) == 0 || got.Names[0] != tc.want {
				t.Fatalf("got %v, want %s", got, tc.want)
			}
		})
	}
	for _, frets := range [][6]int{{-1, -1, -1, -1, -1, -1}, {0, -1, -1, -1, -1, -1}} {
		if got := Identify(frets); len(got.Names) != 0 {
			t.Fatal(got)
		}
	}
	// C6 and Am7 have identical pitch classes; preserve both readings.
	got := Identify([6]int{-1, 3, 2, 2, 1, 3})
	if len(got.Names) < 2 || got.Names[0] != "C6" {
		t.Fatal(got)
	}
}
