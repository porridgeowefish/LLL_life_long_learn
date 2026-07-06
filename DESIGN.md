# Design

## Theme

Warm, paper-like, scholarly. Light mode. A cream "paper" canvas with brown-ink
text and a sage-green accent; soft warm hairline borders and shadows. Editorial
calm. Strengthen the existing identity — do not replace it.

## Color

Existing tokens (`frontend/src/styles/tokens.css`) are the source of truth — evolve them.

```css
--bg: #faf8f4;       /* cream paper — body */
--fg: #3d3229;       /* warm brown ink — text */
--card: #ffffff;     /* surfaces */
--border: #e8e2d9;   /* warm hairline */
--muted: #9e9588;    /* MUST stay ≥4.5:1 on cream — verify, bump toward ink if close */
--accent: #7c9a72;   /* sage — the one accent */
--accent2: #6b8a62;  /* deeper sage — hover/active */
--orange: #e8945a; --sky: #87b5d4; --pink: #c4715e;  /* sparing, semantic only */
```

New tokens to add (strengthen, not break):

```css
--accent-ink: #4f6b47;   /* sage darkened to AA on cream — for accent-colored text/links */
--surface-raised: #ffffff;
--surface-sunken: #f3eee6;  /* warm recessed tint for thinking blocks, inputs hover */
--surface-tint: #eef3ea;    /* faint sage tint for active/hover states */
--shadow-sm: 0 1px 2px rgba(61,50,41,.05);
--shadow-md: 0 4px 14px rgba(61,50,41,.07);
--shadow-lg: 0 16px 40px rgba(61,50,41,.12);
--radius-sm: 6px; --radius: 8px; --radius-lg: 12px; --radius-pill: 999px;
```

Rule: the ONLY saturated accent is sage. No blue (`#4c6ef5`) anywhere — the prior
iter-06 components used off-brand blue and must be reskinned to sage.

## Typography

- Body: `DM Sans, 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif`. 15px root, rendered at 125% via the existing `html { zoom: 1.25 }`.
- Headings: upright (italic disabled globally). Use weight + size for hierarchy, not color or italics.
- Reading measure: 65–75ch for explain/prose columns.
- Line-height 1.5 body; tighter (1.15–1.25) for display headings.

## Components

- **Buttons** (primitive): sage primary (solid `--accent`, ink-on-sage or white text per contrast), outline (warm border), ghost (transparent), danger (warm red-brown, not pure red). Pill or 8px radius; clear `:hover`/`:active`/`:disabled`.
- **Cards**: `--card` on `--bg`, `--radius`, `--shadow-sm`, `--border` hairline. No nested cards.
- **Modal** (Radix dialog): cream surface, `--shadow-lg`, warm border.
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

- Blue accents (`#4c6ef5` etc.) — off-brand; the iter-06 components must be reskinned to sage.
- Gradient text, side-stripe accents, glassmorphism-as-default.
- All-gray low-contrast body text on cream.
- Italic emphasis.
- Identical card grids, eyebrow kickers above every section, numbered section scaffolding.
