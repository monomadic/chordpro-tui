# ChordPro Format Specification (v6) — Agent Working Spec

> **Purpose.** A structured, machine-actionable reference to the **current** ChordPro
> format, for agents that (A) convert plain-text chord sheets / ASCII tab into ChordPro,
> and (B) repair sloppy or non-standard ChordPro into compliant files.
>
> **Authoritative source.** The official ChordPro documentation:
> <https://www.chordpro.org/chordpro/chordpro-introduction/> and the per-directive pages
> under <https://www.chordpro.org/chordpro/>. Markup, annotations, and the full metadata
> set described here became available in **ChordPro 6**; this spec targets that version
> (the current reference implementation).
>
> **Terminology — read this first.** The request said "guitarpro." This spec describes
> **ChordPro** (`.cho`), a **plain-text** lead-sheet markup, matching the official source.
> It is *not* **Guitar Pro** (`.gp`/`.gpx`/`.gp5`), a proprietary **binary** tablature
> format by Arobas Music. They are unrelated. Everything below is ChordPro. If binary
> Guitar Pro output is genuinely needed, that is a different target and out of scope.
>
> **Fidelity markers** (every line is tagged so agents know how much to trust it):
> - **[DOC]** — stated directly by the official chordpro.org documentation.
> - **[CONV]** — community convention or sensible default *not* nailed down by the docs.
>   Use it, but don't treat it as guaranteed across every renderer.
>
> **Golden rule for agents: never invent directives, chord spellings, or attributes.**
> If a construct is not in this document, look it up at chordpro.org before emitting it.
> When unsure, prefer plain lyrics + `{comment:}` over a guessed directive.

---

## 1. Overview [DOC]

- ChordPro is a **plain-text, line-based** format for lead sheets — lyrics with chords.
  Created 1992 by Martin Leclerc and Mario Dorion.
- **File extensions:** `.cho`, `.crd`, `.chopro`, `.chord`, `.pro`. **Use `.cho` for new files.**
- **File naming convention [CONV]:** `Track Name - Artist.cho` where both fields are
  **Title Case**. Example: `Stolen Car - Beth Orton.cho`
- The format defines the **source**, not the appearance. Rendering (PDF by default in the
  reference implementation, also HTML, etc.) varies by tool. Do not encode visual intent
  that belongs to the renderer unless the user explicitly asks.

---

## 2. Line Types & Lexical Rules

A ChordPro file is a sequence of lines. Each line is exactly one of:

| Line kind      | Recognized by                          | Meaning                                              |
|----------------|----------------------------------------|------------------------------------------------------|
| **Comment**    | First non-blank char is `#`            | Source-only annotation; **never rendered**. [DOC]    |
| **Directive**  | Wrapped in `{ … }`                     | A metadata/formatting/structural command. [DOC]      |
| **Song line**  | Anything else                          | Lyrics with optional inline `[chord]` markers. [DOC] |
| **Blank line** | Empty / whitespace only                | Separates stanzas. [CONV]                            |

Lexical details:
- A directive is `{name}` or `{name: argument}`. The separator after the name is a
  **colon**. Emit the colon form for every directive that takes an argument. [DOC]
- Whitespace immediately inside the braces is tolerated but discouraged — emit
  directives flush-left with no inner padding: `{title: Song}`, not `{ title : Song }`. [CONV]
- Encoding: **UTF-8**. [CONV]
- `#` comments vs. `{comment:}` are different: `#` is **invisible** source annotation;
  `{comment:}` produces **visible** printed text. Don't conflate them. [DOC]

---

## 3. Inline Chords [DOC]

Chords are written **between square brackets** immediately before the syllable they sound on:

```
Swing [D]low, sweet [G]chari[D]ot
```

- The bracket attaches to the character that **follows** it; place it mid-word for a
  mid-word chord change (`chari[D]ot`).
- One chord per bracket. **No spaces inside the brackets.** [CONV]
- A chord with no following lyric (instrumental, line ends on a chord) is a trailing or
  stand-alone `[chord]`. [CONV]

### 3.1 Annotations (bracketed text that is NOT a chord) [DOC]

Text in brackets that is not a real chord — performance marks, section cues — **must begin
with an asterisk**:

```
[*Coda]   [*Rit.]   [*N.C.]   [*x4]
```

- This is the modern, correct way to put non-chord text inline. Do **not** emit bare
  `[Coda]` — in **strict** parsing that is invalid; in **relaxed** parsing it is wrongly
  read as the chord root `C` + extension `oda`. Always use the `*` prefix. [DOC]

---

## 4. Chord Notation [DOC]

A chord = **root** + optional **qualifier** + optional **extension** + optional **/bass**.

```
C        Am7        C/B        D/F#        Em7b5        Gmaj7
```

- **Root notes / notation systems supported:**
  - European/Dutch: `A B C D E F G`
  - German: uses `H` for B
  - Roman numerals: `I II III IV V VI VII`
  - Nashville numbers: `1 2 3 4 5 6 7`
- **Accidentals:** sharp `#` (e.g. `F#`), flat `b` (e.g. `Bb`). (Per docs, `B♭`, `Bb`, and
  `Bes` may all designate B-flat depending on configuration.)
- **Qualifier / extension** (e.g. `m`, `aug`, `7`, `alt`) and **bass** via slash (`/B`).
- **Strict vs. relaxed parsing:** in strict mode an unrecognized token is an error; relaxed
  mode invents extensions (reads `[Coda]` as `C` + `oda`). For portability, only emit real
  chords inside `[ ]`; everything else is an annotation (`[*…]`, §3.1). [DOC]
- Pass through standard forms verbatim: `Dsus4`, `Cadd9`, `A7sus4`, `Gmaj7`, `Em7b5`. [CONV]

---

## 5. Directives Reference

Long form is canonical; short form (in parentheses) is an accepted alias. All names below
are **[DOC]** from the official directives index unless marked otherwise. Emit the long
form for clarity unless space matters.

### 5.1 Preamble
| Directive | Short | Argument | Effect |
|-----------|-------|----------|--------|
| `{new_song}` | `{ns}` | — | Begin a new song (for multi-song files). Implicit at file start. |

### 5.2 Metadata (data directives)
Each also has a generic form `{meta: name value}` (§5.9). [DOC]

| Directive | Short | Argument | Notes |
|-----------|-------|----------|-------|
| `{title: …}`      | `{t: …}`  | text | Song title. |
| `{sorttitle: …}`  | —         | text | Alternate title used for sorting. |
| `{subtitle: …}`   | `{st: …}` | text | Secondary title; repeatable. |
| `{artist: …}`     | —         | text | Performing artist; repeatable. |
| `{sortartist: …}` | —         | text | Alternate artist for sorting. |
| `{composer: …}`   | —         | text | Composer; repeatable. |
| `{lyricist: …}`   | —         | text | Lyricist; repeatable. |
| `{arranger: …}`   | —         | text | Arranger. |
| `{copyright: …}`  | —         | text | Copyright notice. |
| `{album: …}`      | —         | text | Album; repeatable. |
| `{year: …}`       | —         | text | Release year. |
| `{key: …}`        | —         | key  | Musical key, e.g. `{key: C}`, `{key: Am}`. See §5.2.1. |
| `{time: …}`       | —         | sig  | Time signature, e.g. `{time: 4/4}`. [DOC name; format CONV] |
| `{tempo: …}`      | —         | bpm  | Tempo in BPM, e.g. `{tempo: 120}`. [DOC name; format CONV] |
| `{duration: …}`   | —         | time | Song length, e.g. `{duration: 3:14}`. [DOC name; format CONV] |
| `{capo: …}`       | —         | int  | Capo fret number, e.g. `{capo: 2}`. See §5.2.1. |
| `{tag: …}`        | —         | text | Custom organizational tag. |

> Most metadata values are stored as free text for display. **`key`, `capo`, and
> `transpose` carry musical meaning** and interact (see §5.2.1 and §5.7). [DOC]

#### 5.2.1 key + capo interaction [DOC]
- `{key:}` is the key the **player reads**. With a capo, the **sounding** key differs:
  `{key: C}` + `{capo: 2}` → player reads C, audience hears D.
- A read-only `_key` metadata item reflects transposition; it **cannot** be set directly.

### 5.3 Visible comments / instructions
| Directive | Short | Effect |
|-----------|-------|--------|
| `{comment: …}`        | `{c: …}`  | Visible comment (historically grey background). For section labels and playing instructions. |
| `{comment_italic: …}` | `{ci: …}` | Visible comment, italic. |
| `{comment_box: …}`    | `{cb: …}` | Visible comment, boxed. |
| `{highlight: …}`      | —         | Alternative to `{comment}`. |

### 5.4 Section environments (blocks) [DOC]
Each is a `start_of_X` … `end_of_X` pair. A label may be given as the start argument:
`{start_of_chorus: Chorus 2}`.

| Start | End | Short | Purpose |
|-------|-----|-------|---------|
| `{start_of_verse}`  | `{end_of_verse}`  | `{sov}`/`{eov}` | Verse block. |
| `{start_of_chorus}` | `{end_of_chorus}` | `{soc}`/`{eoc}` | Chorus block (rendered distinctly). |
| `{start_of_bridge}` | `{end_of_bridge}` | `{sob}`/`{eob}` | Bridge block. |
| `{start_of_tab}`    | `{end_of_tab}`    | `{sot}`/`{eot}` | Verbatim **monospace** ASCII tablature; alignment preserved. |
| `{start_of_grid}`   | `{end_of_grid}`   | `{sog}`/`{eog}` | Chord-grid (measure/bar) block. |

> Unlike old ChordPro (4.x), **`{start_of_verse}` exists in v6.** Prefer it over loose
> blank-line-separated verses when verse structure matters. [DOC]

> **Recommended verse style — keep verses bare.** Emit verses as a plain
> `{start_of_verse}` … `{end_of_verse}` pair with **no** label, no preceding comment, and
> no attribute. The renderer already handles/numbers verses, so a label adds nothing and
> only clutters the source. [CONV]
>
> ```chordpro
> {start_of_verse}                       ✅ recommended — bare
> ...
> {end_of_verse}
> ```
> Do **not** decorate verses in any of these ways:
> ```chordpro
> {comment: Verse 2}                     ❌ stray visible comment before the verse
> {start_of_verse}
>
> {start_of_verse: Verse 2}              ❌ unnecessary label argument
> {start_of_verse: comment="Verse 2"}    ❌ comment attribute
> ```
> (Choruses are the exception: a distinguishing label such as `{start_of_chorus: Chorus 2}`
> is acceptable when a song genuinely has more than one distinct chorus.)

### 5.5 Chorus recall [DOC]
| Directive | Effect |
|-----------|--------|
| `{chorus}` | Re-insert the most recent preceding chorus. |
| `{chorus: Label}` | Same, with a label. |

### 5.6 Chord diagrams [DOC]
| Directive | Effect |
|-----------|--------|
| `{define: …}` | Define a custom chord diagram. Full grammar in §6. |
| `{chord: …}`  | Define/show a chord inline at this point. See <https://www.chordpro.org/chordpro/directives-chord/>. |

### 5.7 Transposition [DOC]
| Directive | Effect |
|-----------|--------|
| `{transpose: n}` | Transpose chords from this point on by `n` semitones. **Positive ⇒ sharps, negative ⇒ flats.** Append `s` or `f` to force accidental type (`{transpose: 2f}`). Empty `{transpose:}` cancels. Affects chords after it, not the `{key}` directive; auto-creates a `key_actual`. |

### 5.8 Delegated environments (embedded notation) [DOC]
Wrap content handed to another engine. Use only when you actually have such content.

| Start / End | Embeds |
|-------------|--------|
| `{start_of_abc}` … `{end_of_abc}` | ABC music notation. |
| `{start_of_ly}` … `{end_of_ly}`   | Lilypond notation. |
| `{start_of_svg}` … `{end_of_svg}` | SVG graphics. |
| `{start_of_textblock}` … `{end_of_textblock}` | Formatted text block. |

### 5.9 Generic metadata + substitution [DOC]
- **Set:** `{meta: name value}` — e.g. `{meta: artist The Beatles}`. Repeat for multiple
  values: two `{meta: composer …}` lines record two composers.
- **Substitute into text** with `%{ … }`:
  - `%{name}` — value of the item.
  - `%{name|true-text|false-text}` and `%{name|true-text}` — conditional on presence.
  - `%{name=value|true-text|false-text}` — equality test.
  - `%{}` — inside a conditional, the controlling item's own value.
    Example: `%{album|Album: %{}}` → `Album: Yes` when `album` is `Yes`.
  - Indexed access for repeated values: `album.1`, `album.2`, `album.-1` (last).
  - Escape `\ { } |` with a leading backslash. Pipes inside values can break nesting.

### 5.10 Conditional directives (selectors) [DOC]
Append `-selector` to **any** directive name to make it conditional:

```
{define-guitar: Am base-fret 1 frets 0 2 2 1 0 0}
{define-ukulele: Am base-fret 1 frets 2 0 0 0}
{comment-guitar: capo on 2}
```

- A selector matches, in order: the configured **instrument type**, then the **user name**,
  then a **meta item** (true if it exists and is non-empty/non-zero/non-false/non-null).
- **Negate** by appending `!`: `{comment-guitar!: …}` applies when the selector is *false*.

### 5.11 Output / layout (rendering only) [DOC names]
Affect appearance, not content. **Omit during text→ChordPro conversion** unless the user
asks for specific formatting.

| Directive | Short | Role |
|-----------|-------|------|
| `{new_page}` | `{np}` | Logical page break. |
| `{new_physical_page}` | `{npp}` | Physical page break. |
| `{column_break}` | `{colb}` | Break to next column. |
| `{columns: n}` | `{col: n}` | Number of columns. |
| `{pagetype: …}` | — | Page layout style. |
| `{titles: …}` | — | Title display behaviour. |
| `{grid}` / `{no_grid}` | `{g}` / `{ng}` | Show / hide the chord-grid summary. |
| `{diagrams: …}` | — | Control chord-diagram display. |
| `{image: …}` | — | Embed an image. See <https://www.chordpro.org/chordpro/directives-image/>. |
| Font/size/colour families | — | `chord*`, `chorus*`, `text*`, `tab*`, `grid*`, `label*`, `title*`, `toc*`, `footer*` — each with `…font`, `…size`, `…colour` (short forms `tf/ts`, `cf/cs`, etc.). |

---

## 6. `{define}` — Custom Chord Diagrams [DOC]

### 6.1 String instruments (guitar, bass, ukulele, …)
```
{define: NAME base-fret OFFSET frets POS POS … POS}
{define: NAME base-fret OFFSET frets POS POS … POS fingers POS POS … POS}
```
- `NAME` — chord identifier (the name used in `[ ]`).
- `base-fret OFFSET` — fret of the topmost finger; **minimum 1**. (Also accepted: `base_fret`.)
- `frets POS …` — one position **per string, lowest-pitch string first**.
- `fingers POS …` — optional finger assignment per string (`1`–`9` or `A`–`Z`).

**Fret position values:**
| Value | Meaning |
|-------|---------|
| `0` | open string |
| `1`–`9`… | fret number (relative to base-fret) |
| `-1`, `N`, or `x` | muted / not sounded |

**Verified examples:**
```
{define: C7 base-fret 1 frets x 3 2 3 1 0}
{define: D7 base-fret 3 frets x 3 2 3 1 x}
{define: Bes base-fret 1 frets 1 1 3 3 3 1 fingers 1 1 2 3 4 1}
{define: As  base-fret 4 frets 1 3 3 2 1 1 fingers 1 3 4 2 1 1}
{define: A frets 0 0 2 2 2 0 base_fret 1}
```

### 6.2 Keyboard instruments
```
{define: NAME keys NOTE … NOTE}
```
```
{define: D  keys 0 4 7}
{define: D² keys 7 12 16}
```

### 6.3 Modifiers (append to a `{define}`) [DOC]
| Modifier | Effect |
|----------|--------|
| `copy B`    | Inherit diagram from chord `B`. |
| `copyall B` | Copy diagram **and** display properties from `B`. |
| `display C` | Set the displayed name to `C`. |
| `diagram off` | Define the chord but omit its diagram. |
| `format fmt` | Custom format string (escape `%{` as `\%{`). |

> Only emit `{define}` when you actually know the fingering. Never fabricate fret numbers.

---

## 7. Conversion Spec: Plain-Text → ChordPro

The operative procedure for use case (A). Input is usually "chords-over-lyrics" text
and/or ASCII tab.

### 7.1 Pipeline
1. **Extract metadata.** Title/artist/album/year/key/capo/tempo lines at the top →
   the matching directives in §5.2 (`{title:}`, `{artist:}`, `{key:}`, `{capo:}`, …).
   Now that real directives exist, **use them** instead of stuffing everything into
   `{subtitle:}`.
2. **Enrich missing metadata (§7.7).** After extracting what's stated, add metadata that
   is missing but can be reliably determined (key, capo, year, composer, time, etc.).
3. **Classify each block:** metadata, section label, chord-over-lyrics pair, bare chord
   line, ASCII tab, or prose note.
4. **Merge chord/lyric pairs** (§7.3).
5. **Wrap sections** (§7.4): verse → `{sov}/{eov}`, chorus → `{soc}/{eoc}`, bridge →
   `{sob}/{eob}`, tab → `{sot}/{eot}`.
6. **Normalize chords** to §4; convert non-chord brackets to `[*…]` annotations (§3.1).
7. **Preserve stanza spacing** with blank lines.
8. **Verify song structure** (§7.2): Check for presence of verses and choruses;
   flag deviations or missing sections.

### 7.2 Song structure verification [CONV]
**Most songs follow a simple, predictable pattern: verses and choruses.** After classifying
sections, do a secondary pass to validate:

- **Expected baseline:** The song contains both `{start_of_verse}` and `{start_of_chorus}` blocks.
- **Simple tags only:** Use bare directives—no labels, no numbered variants. `{start_of_verse}`
  and `{start_of_chorus}` alone suffice for the vast majority of songs. [CONV]
- **Avoid orphaned comments:** Do **not** use floating `{comment: Pre-Chorus}` blocks in isolation.
  A pre-chorus musically leads **into** a chorus and should be placed immediately before the
  `{start_of_chorus}` block it precedes. If the section truly has no associated chorus, 
  reconsider whether it belongs in the song structure or is better handled differently. [CONV]
- **Organize by musical function:** Place lyrics/chords in order of performance:
  - Intro/Outro content → optional `{comment:}` if brief, or a bare song-line block
  - Verse → `{start_of_verse}` block
  - Pre-Chorus (if present) → lyric lines immediately before the chorus block
  - Chorus → `{start_of_chorus}` block
  - Bridge → `{start_of_bridge}` block
- **Flag deviations:** If a song **lacks verses or choruses** entirely, treat it as a potential
  edge case or mislabeling. Examples:
  - No verses → the content labeled "Verse" may actually be intro or pre-chorus material.
  - No choruses → the content labeled "Chorus" may be structural annotations instead.
- **When in doubt, review the lyrics:** Do the lyrics of each labeled section actually repeat
  or serve the expected function? If not, the original tab labels may be misleading.

### 7.3 Merging "chords over lyrics" into inline brackets
```
       D        G    D
Swing low, sweet chariot,
```
becomes
```
Swing [D]low, sweet [G]chari[D]ot,
```
Algorithm:
- For each chord token, record its **starting column** on the chord line.
- Insert `[chord]` into the lyric at that same column index.
- **Insert right-to-left** so earlier insertions don't shift later columns. [CONV]
- If the column is past end-of-lyric, append the bracket. [CONV]
- A chord line with **no** lyric beneath it (intro/instrumental) → a song line of bare
  brackets, optionally preceded by `{comment: Intro}`. [CONV]

### 7.4 Sections
- `Verse`/`Verse 2` block → a **bare** `{start_of_verse}` … `{end_of_verse}` pair —
  **no label, no comment, no attribute** (see the recommended verse style in §5.4).
- `Chorus`/`[Chorus]` block → `{start_of_chorus}` … `{end_of_chorus}` (a label such as
  `{start_of_chorus: Chorus 2}` is acceptable only for a genuinely distinct second chorus).
- `Bridge` → `{start_of_bridge}` … `{end_of_bridge}`.
- A chorus that simply says "repeat chorus" → `{chorus}` (do not duplicate the lines).

### 7.5 ASCII tablature
Any monospace fret-grid block (e.g. `e|---0---2---|`) → wrap **verbatim** in
`{start_of_tab}` … `{end_of_tab}`. **Do not** add `[chord]` brackets inside tab and **do
not** reflow whitespace — alignment is significant. [DOC]

### 7.6 Keep / drop
- **Keep** all lyrics and chord information.
- **Drop** decorative ASCII rules and page numbers; move "tabbed by …" credits to a `#`
  source comment or `{comment:}`.
- Non-chord inline markers (`N.C.`, `x4`, riff names) → `[*N.C.]`, `[*x4]` (§3.1), or
  `{comment:}` for a whole-line instruction. **Never** leave them as bare `[ ]` chords.

### 7.7 Metadata enrichment — add what's missing (when you can do it reliably) [CONV]
Prefer a complete header. After capturing stated metadata, add any of the following that
is **missing but determinable**, so the output carries full, useful metadata. (See also the
structure verification step in §7.2.)

| Item | How to determine it | Confidence rule |
|------|---------------------|-----------------|
| `{key:}` | Infer from the chord set (e.g. the opening/closing tonic and the diatonic chords present). | Add when the chords clearly imply one key; otherwise leave it out. |
| `{capo:}` | From an explicit "Capo N" note in the source, or a well-known arrangement. | Add `{capo: 0}` only if the source states no-capo; don't assume. |
| `{year:}` | Known release year of the song/recording. | Add only when confident of the specific song/version. |
| `{composer:}` / `{lyricist:}` / `{artist:}` | Well-established songwriting/performer credits. | Add only for clearly identified songs. |
| `{time:}` / `{tempo:}` | Stated in the source, or a well-known value. | Add only when stated or confidently known. |

**Hard rule — do not fabricate.** Only add metadata you can determine with confidence from
the source or from well-established facts about the identified song. If a value is a guess,
**omit it** rather than risk a wrong key/year/credit. When useful, leave a `#` source
comment noting an inferred value (e.g. `# key inferred from chords`) so a human can verify.

---

## 8. Verification / Repair Spec: sloppy ChordPro → compliant

The operative procedure for use case (B). Apply checks in order; fix in place.

### 8.1 Structural validity
- [ ] Every `{` has a matching `}` on the **same line**.
- [ ] Every `[` has a matching `]`; **no spaces** inside chord brackets.
- [ ] Every `{start_of_X}` has a matching `{end_of_X}` of the **same** X, properly nested.
- [ ] No stray directive text outside `{ }`; no rendered content hiding in `#` comments.

### 8.2 Directive hygiene
- [ ] Directive names are real (exist in §5) and spelled exactly. Replace unknown
      directives — look them up at chordpro.org rather than guessing.
- [ ] Argument directives use the **colon** form: `{title: …}`. Fix `{title …}`.
- [ ] No inner brace padding: `{ title : X }` → `{title: X}`.
- [ ] Prefer canonical long forms; expand obscure abbreviations for readability. [CONV]
- [ ] Migrate legacy workarounds to real directives: artist/key/capo dumped into
      `{subtitle:}` → `{artist:}` / `{key:}` / `{capo:}`.

### 8.3 Chords & annotations
- [ ] Every `[ ]` token is either a valid chord (§4) **or** an annotation starting with
      `*` (§3.1). Convert bare non-chords (`[Coda]`, `[N.C.]`) → `[*Coda]`, `[*N.C.]`.
- [ ] Accidentals are `#` / `b`; one chord per bracket; no internal spaces.

### 8.4 Sections
- [ ] Section labels expressed via real environments (`{sov}`, `{soc}`, `{sob}`, `{sot}`)
      rather than ad-hoc `{comment:}` where a true section is intended.
- [ ] **Verses are bare** `{start_of_verse}` … `{end_of_verse}` — strip any verse label,
      comment attribute, or preceding `{comment: Verse N}` (see §5.4).
- [ ] Repeated choruses use `{chorus}` recall instead of copy-pasted lyrics (optional but
      preferred). [CONV]
- [ ] Tab blocks are byte-for-byte preserved inside `{sot}/{eot}`; nothing reflowed.

### 8.5 Metadata
- [ ] `{title:}` present. Other known facts mapped to their real directives (§5.2).
- [ ] **Add missing metadata where it can be reliably determined** — key, capo, year,
      composer, etc. (§7.7). Do not fabricate; omit anything uncertain.
- [ ] `key`/`capo`/`transpose` are musically consistent (§5.2.1, §5.7).

### 8.6 Encoding
- [ ] File is UTF-8; line endings consistent.

> When repairing, **do not delete content you cannot classify** — convert it to a `#`
> source comment so nothing is lost, and flag it for human review.

---

## 9. Worked Examples

**Input (chords-over-lyrics):**
```
Swing Low Sweet Chariot
Traditional   Key: D   Capo: 0

Chorus:
       D            G   D
Swing low, sweet chariot,
            A7
Comin' for to carry me home.
```

**Output:**
```chordpro
{title: Swing Low Sweet Chariot}
{artist: Traditional}
{key: D}
{capo: 0}

{start_of_chorus}
Swing [D]low, sweet [G]chari[D]ot,
Comin' for to [A7]carry me home.
{end_of_chorus}
```

**Verse handling — input:**
```
Verse 2
       G           C        G
I looked over Jordan and what did I see
```
**Output (bare verse — no label/comment):**
```chordpro
{start_of_verse}
I [G]looked over Jordan and [C]what did I [G]see
{end_of_verse}
```

**ASCII-tab input:**
```
Intro riff
e|-----0-----|
B|---1---1---|
G|-0-------0-|
```

**Output:**
```chordpro
{comment: Intro riff}
{start_of_tab}
e|-----0-----|
B|---1---1---|
G|-0-------0-|
{end_of_tab}
```

**Repair example — sloppy input:**
```chordpro
{ t : Mr Blue Sky }
{subtitle: ELO   Key: A   Capo: 4}
{comment: Verse 2}
{start_of_verse: comment="Verse 2"}
[Coda] [A]Sun is shini[D]n in the [E]sky
{end_of_verse}
```
**Compliant output (real directives, enriched + bare verse):**
```chordpro
{title: Mr Blue Sky}
{artist: ELO}
{key: A}
{capo: 4}
{year: 1977}
{start_of_verse}
[*Coda] [A]Sun is shini[D]n in the [E]sky
{end_of_verse}
```

---

## 10. Quick Reference Card

```
COMMENT (hidden)     # text
DIRECTIVE            {name: arg}                     (colon before arg, no inner padding)
INLINE CHORD         lyric[Chord]more
ANNOTATION (inline)  [*text]                         (* prefix = NOT a chord)
CHORD TOKEN          Root[#|b][qualifier][ext][/Bass]   e.g. C  Am7  D/F#  Em7b5

TITLE / SUBTITLE     {title:}  {t:}   |  {subtitle:}  {st:}
ARTIST / COMPOSER    {artist:}        |  {composer:}  {lyricist:}  {arranger:}
ALBUM/YEAR/COPYRIGHT {album:} {year:} {copyright:}
KEY/CAPO/TIME/TEMPO  {key: C} {capo: 2} {time: 4/4} {tempo: 120} {duration: 3:14}
GENERIC META         {meta: name value}              substitute: %{name} %{name|t|f} %{}
ENRICH METADATA      add missing key/capo/year/… when reliably known — never fabricate

VERSE                {start_of_verse} … {end_of_verse}      {sov}/{eov}   ← keep BARE, no label
CHORUS               {start_of_chorus} … {end_of_chorus}    {soc}/{eoc}   (label ok if 2+ choruses)
BRIDGE               {start_of_bridge} … {end_of_bridge}    {sob}/{eob}
TAB (monospace)      {start_of_tab} … {end_of_tab}          {sot}/{eot}
GRID                 {start_of_grid} … {end_of_grid}        {sog}/{eog}
CHORUS RECALL        {chorus}   {chorus: Label}

COMMENTS (visible)   {comment:} {c:}  | {comment_italic:} {ci:} | {comment_box:} {cb:} | {highlight:}
TRANSPOSE            {transpose: n}   (+=sharps, -=flats; suffix s/f forces; empty cancels)
DEFINE (strings)     {define: C7 base-fret 1 frets x 3 2 3 1 0 fingers …}   (x|N|-1 = muted)
DEFINE (keys)        {define: D keys 0 4 7}
SELECTOR (cond.)     {directive-instrument: …}   negate with {directive-instrument!: …}
```

---

*Targets ChordPro 6 (current). Verify any construct not listed here against the official
docs at <https://www.chordpro.org/chordpro/> before emitting it — do not guess.*
