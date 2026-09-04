# LLL Frontend

Vite + React 18 + TypeScript single-page app for the LifeLongLearn workbench.

## Stack

- **Build**: Vite 5 + `@vitejs/plugin-react` (production bundle in `dist/`)
- **Framework**: React 18 + TypeScript 5
- **Routing**: `react-router-dom@6` (path-param routes, lazy-loaded pages)
- **Server state**: `@tanstack/react-query@5`
- **Client state**: `zustand@4` (slices for ui / project / session / connection)
- **Forms**: `react-hook-form` + `zod`
- **UI primitives**: self-written + `@radix-ui/react-dialog` (headless)
- **Styling**: CSS Modules + `src/styles/tokens.css` (v2 design tokens)
- **Markdown**: `marked` + `dompurify` + `mermaid` (lazy-loaded)
- **Tests**: Vitest + Testing Library + MSW (Playwright deferred)

## Layout

```
frontend/
├── index.html                    Vite entry (just <div id="root">)
├── package.json                  npm scripts + deps
├── vite.config.ts                dev server + proxy + build config
├── tsconfig.json / .node.json
├── public/
│   ├── img/icons.svg             SVG sprite (10 line icons)
│   └── logo-lifelonglearn.svg
└── src/
    ├── app/                      entry, router, layout, SSE, connection state
    ├── features/                 learning, projects, agents, legacy-zones,
    │                             preferences, and settings vertical slices
    │   └── <feature>/index.ts    the only supported cross-feature import surface
    └── shared/                   primitives, HTTP client, query keys, common
                                  hooks, stores, pure helpers, types, and styles
```

## State Management

| Layer | What lives here | Examples |
|---|---|---|
| **TanStack Query** (server) | Anything that lives on the backend | `useProjects`, `useHealth`, `useAgents`, `useRecentSessions` |
| **Zustand** (client) | UI state, selection, SSE buffer | `useUiStore`, `useProjectStore`, `useSessionStore`, `useConnectionStore` |

The boundary is "data we fetched" (server) vs "what the user clicked" (client).

## Routes

| Path | Page | Notes |
|---|---|---|
| `/` | HomePage | dashboard with stats + project grid + recent runs |
| `/project/:id/:zone?` | ProjectPage | zone is a path param; defaults to Explain if absent |
| `/agents` | AgentsPage | registry grid + detail panel |
| `/memory` | redirect | compatibility redirect to the current project flow |
| `*` | redirect to `/` | |

## SSE Strategy

A single `useSSE()` hook is mounted once at `<AppShell />`. It opens one
`/api/events` connection and fans events out to subscribers via
`subscribeToSSE(eventName, handler)`. The connection status flows into
`useConnectionStore`, which `ConnectionBadge` reads.

This eliminates the legacy pattern where every page opened its own
EventSource.

## Development

```bash
# Terminal 1: Go backend on :8787
go run ./backend-go/cmd/lll

# Terminal 2: Vite dev server on :5173
cd frontend
npm install        # one-time
npm run dev
```

Vite proxies `/api`, `/files`, `/events` to `:8787`, so the dev experience
is API-live with hot module reload. Open `http://localhost:5173`.

## Production

```bash
cd frontend
npm run build      # emits dist/
cd ..
go run ./backend-go/cmd/lll
# Open http://localhost:8787 — Go serves dist/ with SPA fallback.
```

The Go binary's `spaHandler` (see `backend-go/internal/transport/httpserver/router.go`)
serves `dist/index.html` for any path that is not a known API route and
does not match a real file. That lets react-router own client-side routes
like `/project/abc/Explain`.

## Testing

```bash
npm run test       # Vitest run (44 files / 149 tests today)
npm run test:watch # Vitest watch mode
```

E2E with Playwright is deferred (W6 of the refactor plan).

## Legacy Reference

`frontend/legacy/` is archive/reference material from the older static UI.
It is not a product fallback and should not be treated as the active frontend contract.

## Decisions

See the refactor plan at `~/.claude/plans/snazzy-launching-beaver.md` and
`docs/00-product-and-architecture/AGENT_PRIMITIVES.md` for the design
rationale (CSS Modules over Tailwind, single SSE connection, lazy-loaded
mermaid, etc.).
