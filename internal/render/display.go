package render

// Tri is a three-state display switch. The zero value is Off, so a bare
// RenderOpts{} renders with every option disabled (the plain, un-collapsed
// look). On forces the behaviour; Auto applies it only when the song would
// otherwise overflow the screen (fit mode only — scroll mode always has room,
// so Auto behaves as Off there).
type Tri int

const (
	Off Tri = iota
	On
	Auto
)

// display holds the concrete rendering choices for a single fit pass, after
// tri-state options have been resolved to plain booleans.
type display struct {
	hideHeader        bool // hide the whole title/metadata header (the 'h' key)
	hideTitle         bool // hide the title + artist line
	hideInfo          bool // hide the metadata pills (KEY, CAPO, …)
	collapsePageTitle bool // lay the title, artist, and metadata on one line
	hideTabs          bool // fold away tab (tablature) sections
	hideSectionTitles bool // drop section labels (CHORUS, VERSE, …)
	sectionTitleGap   bool // add a blank row above each labeled section
}

// resolveDisplay turns opts into a base display (auto options in their roomiest
// state) plus an ordered ladder of reduction steps for the options set to Auto.
// The fit renderer applies steps one at a time, least-destructive first, until
// the song fits.
func resolveDisplay(opts RenderOpts) (display, []func(*display)) {
	d := display{
		hideHeader:        opts.HideHeader,
		hideSectionTitles: opts.HideSectionTitles,
	}
	// On settings apply immediately; Auto options start roomy and are trimmed by
	// the ladder below.
	if opts.CollapseTabs == On {
		d.hideTabs = true
	}
	if opts.HideTabs {
		d.hideTabs = true // the 'T' key forces a fold regardless of config
	}
	if opts.HideInfo == On {
		d.hideInfo = true
	}
	if opts.HideTitle == On {
		d.hideTitle = true
	}
	if opts.CollapsePageTitle == On {
		d.collapsePageTitle = true
	}
	if opts.SectionTitleGap == On || opts.SectionTitleGap == Auto {
		d.sectionTitleGap = true
	}

	// Reduction ladder, ordered least- to most-destructive: first give back the
	// blank rows above sections, then pack the header onto one line, then fold
	// tabs, then drop the metadata pills, and only as a last resort hide the
	// title itself.
	var steps []func(*display)
	if opts.SectionTitleGap == Auto {
		steps = append(steps, func(x *display) { x.sectionTitleGap = false })
	}
	if opts.CollapsePageTitle == Auto {
		steps = append(steps, func(x *display) { x.collapsePageTitle = true })
	}
	if opts.CollapseTabs == Auto {
		steps = append(steps, func(x *display) { x.hideTabs = true })
	}
	if opts.HideInfo == Auto {
		steps = append(steps, func(x *display) { x.hideInfo = true })
	}
	if opts.HideTitle == Auto {
		steps = append(steps, func(x *display) { x.hideTitle = true })
	}
	return d, steps
}
