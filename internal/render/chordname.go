package render

import "strings"

// superscripts maps the characters that appear in chord qualities (m, maj7,
// sus4, add9, dim, aug, 7b5, …) to their superscript forms. Only glyphs with
// broad terminal-font support are listed; the superscript capitals for C and F
// (U+A7F2/3) are too new to rely on, which is why roots stay full size.
var superscripts = map[rune]rune{
	'a': 'ᵃ', 'b': 'ᵇ', 'c': 'ᶜ', 'd': 'ᵈ', 'e': 'ᵉ', 'f': 'ᶠ', 'g': 'ᵍ',
	'h': 'ʰ', 'i': 'ⁱ', 'j': 'ʲ', 'k': 'ᵏ', 'l': 'ˡ', 'm': 'ᵐ', 'n': 'ⁿ',
	'o': 'ᵒ', 'p': 'ᵖ', 'r': 'ʳ', 's': 'ˢ', 't': 'ᵗ', 'u': 'ᵘ', 'v': 'ᵛ',
	'w': 'ʷ', 'x': 'ˣ', 'y': 'ʸ', 'z': 'ᶻ',
	'M': 'ᴹ',
	'0': '⁰', '1': '¹', '2': '²', '3': '³', '4': '⁴',
	'5': '⁵', '6': '⁶', '7': '⁷', '8': '⁸', '9': '⁹',
	'+': '⁺', '-': '⁻', '(': '⁽', ')': '⁾',
}

// superQuality renders a chord name with its root (and any slash bass) at full
// size and the quality raised, e.g. "F#m7b5" → "F#ᵐ⁷ᵇ⁵", "D/F#" unchanged.
// Each character maps one-to-one, so the name keeps its width and chord
// alignment is unaffected. A quality with a character that has no superscript
// form (e.g. the '#' in "7#9") is left as-is rather than half-raised.
func superQuality(name string) string {
	rs := []rune(name)
	if len(rs) == 0 || rs[0] < 'A' || rs[0] > 'G' {
		return name // not a chord we can parse (N.C., %, …)
	}
	i := 1
	if i < len(rs) && (rs[i] == '#' || rs[i] == 'b') {
		i++
	}
	end := len(rs)
	if s := strings.IndexRune(string(rs[i:]), '/'); s >= 0 {
		end = i + len([]rune(string(rs[i:])[:s]))
	}
	quality := make([]rune, 0, end-i)
	for _, r := range rs[i:end] {
		sr, ok := superscripts[r]
		if !ok {
			return name
		}
		quality = append(quality, sr)
	}
	return string(rs[:i]) + string(quality) + string(rs[end:])
}
