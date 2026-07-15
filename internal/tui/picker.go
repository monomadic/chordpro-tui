package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"chordpro-tui/internal/chordpro"
	"chordpro-tui/internal/config"
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
	key    string // key signature
	capo   string // capo fret
	tempo  string // tempo (bpm)
	year   string // year
	mod    int64  // modification time (unix nanos), for date sorting
}

type pickMatch struct {
	idx       int          // index into picker.entries
	pos       map[int]bool // matched rune positions in the title, for highlighting
	artistPos map[int]bool // matched rune positions in the artist, for highlighting
}

// picker is the fuzzy "open song" overlay state.
type picker struct {
	dir     string
	entries []pickEntry
	query   string
	matches []pickMatch
	cursor  int
	top     int             // index of the first visible row
	sort    config.SortMode // order of the unfiltered listing
	err     string
}

// newPicker scans dir for ChordPro files and pre-selects the current song.
func newPicker(dir, currentPath string, mode config.SortMode) picker {
	p := picker{dir: dir, sort: mode}
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

// orderedChordPaths returns the ChordPro files in dir ordered by mode: filename
// order (none), song title (name), or newest-first (date). name/date read each
// file's metadata; none stays cheap and skips that.
func orderedChordPaths(dir string, mode config.SortMode) []string {
	paths, err := chordFilePaths(dir) // filename order
	if err != nil || mode == config.SortNone {
		return paths
	}
	entries := make([]pickEntry, len(paths))
	for i, p := range paths {
		base := filepath.Base(p)
		entries[i] = readMeta(p, strings.TrimSuffix(base, filepath.Ext(base)))
	}
	order := sortedIndices(entries, mode)
	out := make([]string, len(order))
	for i, idx := range order {
		out[i] = entries[idx].path
	}
	return out
}

// sortedIndices returns the indices of entries in the order implied by mode.
// SortNone keeps the incoming (filename) order.
func sortedIndices(entries []pickEntry, mode config.SortMode) []int {
	idx := make([]int, len(entries))
	for i := range idx {
		idx[i] = i
	}
	switch mode {
	case SortByTitle:
		sort.SliceStable(idx, func(a, b int) bool {
			return strings.ToLower(entries[idx[a]].title) < strings.ToLower(entries[idx[b]].title)
		})
	case SortByDate:
		sort.SliceStable(idx, func(a, b int) bool {
			return entries[idx[a]].mod > entries[idx[b]].mod // newest first
		})
	}
	return idx
}

// Aliases keep the sort-mode references readable within this file.
const (
	SortByTitle = config.SortName
	SortByDate  = config.SortDate
)

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
	if fi, err := os.Stat(path); err == nil {
		e.mod = fi.ModTime().UnixNano()
	}
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
	e.key = s.Key
	e.capo = s.Capo
	e.tempo = s.Tempo
	e.year = s.Year
	return e
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

// refilter recomputes and re-sorts matches for the current query. With no
// query the full list is shown in the configured sort order; otherwise entries
// are ranked by fuzzy relevance.
func (p *picker) refilter() {
	p.matches = p.matches[:0]
	if p.query == "" {
		for _, i := range sortedIndices(p.entries, p.sort) {
			p.matches = append(p.matches, pickMatch{idx: i})
		}
		return
	}
	type scored struct {
		m     pickMatch
		score int
		name  string
	}
	var hits []scored
	for i, e := range p.entries {
		// Match against both title and artist; an entry is shown if either hits.
		// The title drives the score (artist matches are biased slightly lower so
		// a title hit outranks an artist-only hit for the same query), but matched
		// characters are highlighted in whichever column they fall.
		const artistBias = 2
		var best int
		matched := false
		m := pickMatch{idx: i}
		if score, pos, ok := fuzzyMatch(p.query, e.title); ok {
			best, matched = score, true
			m.pos = runeSet(pos)
		}
		if score, pos, ok := fuzzyMatch(p.query, e.artist); ok && e.artist != "" {
			m.artistPos = runeSet(pos)
			if s := score - artistBias; !matched || s > best {
				best = s
			}
			matched = true
		}
		if !matched {
			continue
		}
		hits = append(hits, scored{m, best, strings.ToLower(e.title)})
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

// runeSet turns a slice of matched positions into a set for O(1) lookup.
func runeSet(pos []int) map[int]bool {
	if len(pos) == 0 {
		return nil
	}
	set := make(map[int]bool, len(pos))
	for _, x := range pos {
		set[x] = true
	}
	return set
}

func isBoundary(r rune) bool {
	switch r {
	case ' ', '-', '_', '.', '/':
		return true
	}
	return false
}

// pcols holds the rendered width of each picker column (0 = hidden).
type pcols struct {
	title, artist, key, capo, tempo, year int
}

// pickerColumns lays out the columns for width w. Title and artist share the
// flexible space; key/capo/tempo/year are fixed and dropped (year first, key
// last) when the screen is too narrow to fit them.
func pickerColumns(w int) pcols {
	c := pcols{key: 5, capo: 5, tempo: 6, year: 5}
	avail := w - 2 // minus the pointer column
	const minTitleArtist = 24

	metaTotal := func(c pcols) int {
		t := 0
		for _, x := range []int{c.key, c.capo, c.tempo, c.year} {
			if x > 0 {
				t += 1 + x // leading gap + column
			}
		}
		return t
	}
	for metaTotal(c) > avail-minTitleArtist-1 {
		switch {
		case c.year > 0:
			c.year = 0
		case c.tempo > 0:
			c.tempo = 0
		case c.capo > 0:
			c.capo = 0
		case c.key > 0:
			c.key = 0
		default:
		}
		if c.key == 0 && c.capo == 0 && c.tempo == 0 && c.year == 0 {
			break
		}
	}

	rest := avail - metaTotal(c) - 1 // 1 gap between title and artist
	if rest < 8 {
		rest = 8
	}
	c.title = rest * 55 / 100
	c.artist = rest - c.title
	return c
}

// view renders the full-screen picker overlay.
func (p *picker) view(w, h int, th *render.Theme) string {
	const headerRows, footerRows = 4, 1 // title, prompt, divider, column headers
	listH := h - headerRows - footerRows
	if listH < 1 {
		listH = 1
	}
	p.scrollIntoView(listH)

	cols := pickerColumns(w)
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
	columnHeader := p.renderHeaderRow(cols, th)

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
		rows = append(rows, p.renderRow(p.matches[i], i == p.cursor, cols, th))
	}
	for len(rows) < listH {
		rows = append(rows, "")
	}

	hint := dim.Render("↑/↓ move · type to filter · enter open · esc cancel")

	out := titleLine + "\n" + promptLine + "\n" + divider + "\n" + columnHeader + "\n" +
		strings.Join(rows[:listH], "\n") + "\n" + hint
	return out
}

// renderHeaderRow draws the column-title row, aligned with the data columns.
func (p *picker) renderHeaderRow(c pcols, th *render.Theme) string {
	hs := lipgloss.NewStyle().Foreground(th.P.Muted).Bold(true)
	var b strings.Builder
	b.WriteString("  ") // pointer column
	b.WriteString(styleCol([]rune("TITLE"), nil, c.title, hs, hs))
	b.WriteString(" ")
	b.WriteString(styleCol([]rune("ARTIST"), nil, c.artist, hs, hs))
	col := func(label string, width int) {
		if width <= 0 {
			return
		}
		b.WriteString(" ")
		b.WriteString(styleCol([]rune(label), nil, width, hs, hs))
	}
	col("KEY", c.key)
	col("CAPO", c.capo)
	col("TEMPO", c.tempo)
	col("YEAR", c.year)
	return b.String()
}

// renderRow styles one list entry across the columns, highlighting matched
// characters in the title; the selected row is filled with a subtle background.
func (p *picker) renderRow(m pickMatch, selected bool, c pcols, th *render.Theme) string {
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
	keySt := mk(th.P.Section)
	capoSt := mk(th.P.Comment)
	tempoSt := mk(th.P.Comment)
	yearSt := mk(th.P.Muted)
	sepSt := mk(th.P.Muted)
	matchSt := mk(th.P.Chord).Bold(true)
	pointerSt := mk(th.P.Chord).Bold(true)

	pointer := "  "
	if selected {
		pointer = "❯ "
	}

	var b strings.Builder
	b.WriteString(pointerSt.Render(pointer))
	b.WriteString(styleCol([]rune(e.title), m.pos, c.title, titleSt, matchSt))
	b.WriteString(sepSt.Render(" "))
	b.WriteString(styleCol([]rune(e.artist), m.artistPos, c.artist, artistSt, matchSt))
	col := func(val string, width int, st lipgloss.Style) {
		if width <= 0 {
			return
		}
		b.WriteString(sepSt.Render(" "))
		b.WriteString(styleCol([]rune(val), nil, width, st, st))
	}
	col(e.key, c.key, keySt)
	col(e.capo, c.capo, capoSt)
	col(e.tempo, c.tempo, tempoSt)
	col(e.year, c.year, yearSt)
	return b.String()
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
