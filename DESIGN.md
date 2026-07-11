# Design

## Theme

Calm, scholarly product UI with user-selectable semantic themes. Lychee Paper
preserves the warm paper identity; Mountain Mist, Wisteria Gray, and Night Ink
offer equally restrained alternatives. System mode maps light preference to
Lychee Paper and dark preference to Night Ink.

## Color

`frontend/src/styles/tokens.css` is the source of truth. Components consume
semantic roles and never preset hex values.

```css
--bg: #faf8f4;       /* cream paper — body */
--fg: #3d3229;       /* warm brown ink — text */
--card: #fffdf9;     /* surfaces */
--border: #ddd6cc;   /* warm hairline */
--muted: #6f685f;    /* 5.18:1 on the default background */
--accent: #60775a;   /* primary actions and selected state */
--accent2: #50664b;  /* hover/active */
```

Every preset supplies these roles:

```css
--bg; --fg; --card; --border; --muted;
--accent; --accent2; --accent-ink; --on-accent;
--surface-raised; --surface-sunken; --surface-tint;
--heat-0; --heat-1; --heat-2; --heat-3; --heat-4;
--orange; --sky; --pink;
```

Rules: one restrained accent per preset; semantic warning/danger/info colors do
not become decorative accents. Normal text is at least 4.5:1, essential UI
boundaries at least 3:1, and state never relies on hue alone.

## Typography

- Body: `DM Sans, 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif`. 15px root, rendered at 125% via the existing `html { zoom: 1.25 }`.
- Headings: upright (italic disabled globally). Use weight + size for hierarchy, not color or italics.
- Reading measure: 65–75ch for explain/prose columns.
- Line-height 1.5 body; tighter (1.15–1.25) for display headings.

## Components

- **Buttons** (primitive): semantic primary (`--accent` + `--on-accent`), outline, ghost, and danger; clear `:hover`/`:active`/`:disabled`.
- **Cards**: `--card` on `--bg`, `--radius`, `--shadow-sm`, `--border` hairline. No nested cards.
- **Modal** (Radix dialog): themed surface, `--shadow-lg`, semantic border.
- **Ask-AI panel** (floating): warm `--card` surface, sage primary actions, user bubble in sage tint (`--surface-tint` with `--accent-ink` text — not blue), assistant via MarkdownView, collapsible thinking in `--surface-sunken`.
- **RunProgressBar**: sage fill (`--accent`) on warm track; determinate "N / M 页".
- **Sidebar / Topbar**: minimal warm chrome that recedes; active item in `--surface-tint` + `--accent-ink`.

## Layout

- Minimal sidebar (144px), compact header (44px), maximum content room.
- Flexbox for 1D, Grid for 2D. Responsive grids via `repeat(auto-fit, minmax(…))`.
- Vary spacing for rhythm; don't default to cards.

## Motion

- Ease-out (quart/quint/expo). No bounce/elastic.
- Every animation needs a `@media (prefers-reduced-motion: reduce)` crossfade/instant fallback.
- Animate transform/opacity (and blur/clip-path where it helps); never animate layout properties.

## Absolute bans (slop — refuse on sight)

- Hard-coded preset colors inside components.
- Gradient text, side-stripe accents, glassmorphism-as-default.
- All-gray low-contrast body text on cream.
- Italic emphasis.
- Identical card grids, eyebrow kickers above every section, numbered section scaffolding.
