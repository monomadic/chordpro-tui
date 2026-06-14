package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"chordpro-tui/internal/chordpro"
	"chordpro-tui/internal/render"

	"github.com/charmbracelet/lipgloss"
)

// chordExts are the file extensions treated as ChordPro songs.
var chordExts = map[string]bool{
	".cho": true, ".chopro": true, ".chordpro": true,
	".crd": true, ".pro": true, ".cp": true,
}

// NewestSong returns the most recently modified ChordPro file in dir.
func NewestSong(dir string) (string, error) {
	paths, err := chordFilePaths(dir)
	if err != nil {
		return "", err
	}
	if len(paths) == 0 {
		return "", fmt.Errorf("no ChordPro files in %s", dir)
	}
	newest := paths[0]
	var newestMod int64 = -1
	for _, p := range paths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		if m := fi.ModTime().UnixNano(); m > newestMod {
			newestMod = m
			newest = p
		}
	}
	return newest, nil
}

type pickEntry struct {
	path   string // full path passed to loadSong
	title  string // song title (falls back to the filename without extension)
	artist string // song artist, if any
	meta   string // compact extra metadata (key · capo · tempo · year)
}

type pickMatch struct {
	idx int          // index into picker.entries
	pos map[int]bool // matched rune positions, for highlighting
}

// picker is the fuzzy "open song" overlay state.
type picker struct {
	dir     string
	entries []pickEntry
	query   string
	matches []pickMatch
	cursor  int
	top     int // index of the first visible row
	err     string
}

// newPicker scans dir for ChordPro files and pre-selects the current song.
func newPicker(dir, currentPath string) picker {
	p := picker{dir: dir}
	entries, err := scanChordFiles(dir)
	if err != nil {
		p.err = err.Error()
		return p
	}
	p.entries = entries
	p.refilter()

	// Start the cursor on the current song if it's in the list. The view's own
	// scrollIntoView positions the window on first render.
	if currentPath != "" {
		want := filepath.Base(currentPath)
		for i, m := range p.matches {
			if filepath.Base(p.entries[m.idx].path) == want {
				p.cursor = i
				break
			}
		}
	}
	return p
}

// chordFilePaths returns the sorted full paths of ChordPro files in dir.
func chordFilePaths(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		if !chordExts[strings.ToLower(filepath.Ext(e.Name()))] {
			continue
		}
		paths = append(paths, filepath.Join(dir, e.Name()))
	}
	sort.Slice(paths, func(i, j int) bool {
		return strings.ToLower(filepath.Base(paths[i])) < strings.ToLower(filepath.Base(paths[j]))
	})
	return paths, nil
}

// scanChordFiles lists ChordPro files in dir with their displayable metadata.
func scanChordFiles(dir string) ([]pickEntry, error) {
	paths, err := chordFilePaths(dir)
	if err != nil {
		return nil, err
	}
	entries := make([]pickEntry, 0, len(paths))
	for _, p := range paths {
		base := filepath.Base(p)
		fallback := strings.TrimSuffix(base, filepath.Ext(base))
		entries = append(entries, readMeta(p, fallback))
	}
	return entries, nil
}

// readMeta reads a song's displayable metadata, falling back to the filename
// for the title when the file has none or can't be read.
func readMeta(path, fallback string) pickEntry {
	e := pickEntry{path: path, title: fallback}
	f, err := os.Open(path)
	if err != nil {
		return e
	}
	defer f.Close()
	s, err := chordpro.Parse(f)
	if err != nil {
		return e
	}
	if s.Title != "" {
		e.title = s.Title
	}
	e.artist = s.Artist
	e.meta = songMeta(s)
	return e
}

// songMeta builds a compact "key · capo · tempo · year" string, skipping
// anything the song doesn't specify.
func songMeta(s *chordpro.Song) string {
	var parts []string
	if s.Key != "" {
		parts = append(parts, s.Key)
	}
	if s.Capo != "" {
		parts = append(parts, "capo "+s.Capo)
	}
	if s.Tempo != "" {
		parts = append(parts, s.Tempo+" bpm")
	}
	if s.Year != "" {
		parts = append(parts, s.Year)
	}
	return strings.Join(parts, " · ")
}

func (p *picker) setQuery(q string) {
	p.query = q
	p.refilter()
	p.cursor, p.top = 0, 0
}

func (p *picker) appendQuery(s string) { p.setQuery(p.query + s) }

func (p *picker) backspace() {
	if p.query == "" {
		return
	}
	r := []rune(p.query)
	p.setQuery(string(r[:len(r)-1]))
}

// refilter recomputes and re-sorts matches for the current query.
func (p *picker) refilter() {
	p.matches = p.matches[:0]
	type scored struct {
		m     pickMatch
		score int
		name  string
	}
	var hits []scored
	for i, e := range p.entries {
		score, pos, ok := fuzzyMatch(p.query, e.title)
		if !ok {
			continue
		}
		set := make(map[int]bool, len(pos))
		for _, x := range pos {
			set[x] = true
		}
		hits = append(hits, scored{pickMatch{idx: i, pos: set}, score, strings.ToLower(e.title)})
	}
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].score != hits[b].score {
			return hits[a].score > hits[b].score
		}
		return hits[a].name < hits[b].name
	})
	for _, h := range hits {
		p.matches = append(p.matches, h.m)
	}
}

func (p *picker) move(delta int) {
	if len(p.matches) == 0 {
		return
	}
	p.cursor += delta
	if p.cursor < 0 {
		p.cursor = 0
	}
	if p.cursor >= len(p.matches) {
		p.cursor = len(p.matches) - 1
	}
}

func (p picker) selected() (string, bool) {
	if len(p.matches) == 0 {
		return "", false
	}
	return p.entries[p.matches[p.cursor].idx].path, true
}

// scrollIntoView keeps the cursor within the visible window of listH rows.
func (p *picker) scrollIntoView(listH int) {
	if listH < 1 {
		listH = 1
	}
	if p.cursor < p.top {
		p.top = p.cursor
	}
	if p.cursor >= p.top+listH {
		p.top = p.cursor - listH + 1
	}
	if p.top < 0 {
		p.top = 0
	}
}

// fuzzyMatch reports whether query is a subsequence of target (case-insensitive)
// and, if so, returns a relevance score and the matched rune positions. Higher
// scores rank better: contiguous runs and word-boundary hits are rewarded,
// later positions and longer names are penalised.
func fuzzyMatch(query, target string) (int, []int, bool) {
	if query == "" {
		return 0, nil, true
	}
	q := []rune(strings.ToLower(query))
	t := []rune(strings.ToLower(target))
	qi, prev, score := 0, -2, 0
	var pos []int
	for ti := 0; ti < len(t) && qi < len(q); ti++ {
		if t[ti] != q[qi] {
			continue
		}
		switch {
		case ti == 0 || isBoundary(t[ti-1]):
			score += 8 // start of a word
		case ti == prev+1:
			score += 5 // contiguous run
		}
		score -= ti / 4 // mild preference for earlier matches
		pos = append(pos, ti)
		prev = ti
		qi++
	}
	if qi < len(q) {
		return 0, nil, false
	}
	score += 20 - len(t) // prefer shorter names
	return score, pos, true
}

func isBoundary(r rune) bool {
	switch r {
	case ' ', '-', '_', '.', '/':
		return true
	}
	return false
}

// view renders the full-screen picker overlay.
func (p *picker) view(w, h int, th *render.Theme) string {
	const headerRows, footerRows = 3, 1
	listH := h - headerRows - footerRows
	if listH < 1 {
		listH = 1
	}
	p.scrollIntoView(listH)

	caret := lipgloss.NewStyle().Foreground(th.P.Chord).Bold(true)
	dim := th.Muted

	// Title + count line.
	count := th.Muted.Render(strconv.Itoa(len(p.matches)) + "/" + strconv.Itoa(len(p.entries)))
	title := th.Section.Render("Open song") + "  " + dim.Render("· "+p.dir)
	titleLine := justify(title, count, w)

	// Prompt line with the live query.
	prompt := caret.Render("❯ ") + th.Lyric.Render(p.query) + caret.Render("▌")
	promptLine := lipgloss.NewStyle().Width(w).MaxWidth(w).Render(prompt)

	divider := dim.Render(strings.Repeat("─", w))

	// List rows.
	var rows []string
	if p.err != "" {
		rows = append(rows, th.Comment.Render("  "+p.err))
	} else if len(p.matches) == 0 {
		rows = append(rows, dim.Render("  no matching songs"))
	}
	end := p.top + listH
	if end > len(p.matches) {
		end = len(p.matches)
	}
	for i := p.top; i < end; i++ {
		rows = append(rows, p.renderRow(p.matches[i], i == p.cursor, w, th))
	}
	for len(rows) < listH {
		rows = append(rows, "")
	}

	hint := dim.Render("↑/↓ move · type to filter · enter open · esc cancel")

	out := titleLine + "\n" + promptLine + "\n" + divider + "\n" +
		strings.Join(rows[:listH], "\n") + "\n" + hint
	return out
}

// renderRow styles one list entry as three columns — title, artist, and a
// compact metadata column (key · capo · tempo · year), each in its own color.
// Matched characters in the title are highlighted; the selected row is filled
// with a subtle background.
func (p *picker) renderRow(m pickMatch, selected bool, w int, th *render.Theme) string {
	e := p.entries[m.idx]

	// Style factory: every segment shares the selection background so it reads
	// as one continuous bar.
	mk := func(fg lipgloss.Color) lipgloss.Style {
		s := lipgloss.NewStyle().Foreground(fg)
		if selected {
			s = s.Background(th.P.PillBg)
		}
		return s
	}
	titleSt := mk(th.P.Title)
	artistSt := mk(th.P.Subtitle)
	metaSt := mk(th.P.Comment)
	sepSt := mk(th.P.Muted)
	matchSt := mk(th.P.Chord).Bold(true)
	pointerSt := mk(th.P.Chord).Bold(true)

	// Column geometry: pointer(2) + title + gap(1) + artist + gap(1) + meta == w.
	avail := w - 4
	if avail < 6 {
		avail = 6
	}
	titleW := avail * 42 / 100
	artistW := avail * 30 / 100
	metaW := avail - titleW - artistW
	if titleW < 6 { // too narrow to split: give it all to the title
		titleW, artistW, metaW = avail, 0, 0
	}

	pointer := "  "
	if selected {
		pointer = "❯ "
	}

	var b strings.Builder
	b.WriteString(pointerSt.Render(pointer))
	b.WriteString(styleCol([]rune(e.title), m.pos, titleW, titleSt, matchSt))
	b.WriteString(sepSt.Render(" "))
	b.WriteString(styleCol([]rune(e.artist), nil, artistW, artistSt, artistSt))
	b.WriteString(sepSt.Render(" "))
	b.WriteString(styleCol([]rune(fitTokens(e.meta, metaW)), nil, metaW, metaSt, metaSt))
	return b.String()
}

// fitTokens trims a " · "-joined string to the most leading tokens that fit in
// width w, so the metadata column never cuts a value mid-token.
func fitTokens(s string, w int) string {
	if lipgloss.Width(s) <= w {
		return s
	}
	out := ""
	for _, tok := range strings.Split(s, " · ") {
		cand := tok
		if out != "" {
			cand = out + " · " + tok
		}
		if lipgloss.Width(cand) > w {
			break
		}
		out = cand
	}
	return out
}

// styleCol renders runes into a fixed-width column, highlighting positions in
// pos with the match style and padding (with base) to width w.
func styleCol(runes []rune, pos map[int]bool, w int, base, match lipgloss.Style) string {
	var b strings.Builder
	n := 0
	for i, r := range runes {
		if n >= w {
			break
		}
		if pos[i] {
			b.WriteString(match.Render(string(r)))
		} else {
			b.WriteString(base.Render(string(r)))
		}
		n++
	}
	if n < w {
		b.WriteString(base.Render(strings.Repeat(" ", w-n)))
	}
	return b.String()
}

// justify places left and right on the same line, justified to width w.
func justify(left, right string, w int) string {
	gap := w - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}
