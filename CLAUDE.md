# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Apricot is a local, open-source AI-powered tech blog curator. A single Go binary serves a React SPA (embedded via `go:embed`), fetches RSS feeds from 21 engineering blogs (with HTML scraping fallback for sites without RSS), and uses a pluggable LLM provider (Anthropic or OpenAI — user's own API key) to filter, rank, and summarize posts based on user interests. All data stays local in SQLite. No cloud, no auth, no hosting.

## Commands

```bash
make run              # Build frontend + Go binary, start server at localhost:8080
make build            # Build frontend then compile Go binary to bin/apricot
make build-frontend   # npm install + build React app, copy to internal/api/dist/
make dev              # Vite dev server (:5173) + Go backend (:8080) with live reload (air)
make clean            # Remove bin/, tmp/, web/dist/, web/node_modules/, internal/api/dist/
make test             # Run Go tests + frontend tests
go test ./...         # Go tests only
go test ./internal/storage/...  # Single package test
cd web && npm test -- --run     # Frontend tests only
```

## Architecture

```
Go binary (single process)
├── cmd/server/main.go          — Entry point: config, DB, router, auto-open browser
├── internal/config/            — TOML config parsing, defaults, env var overrides
├── internal/models/            — Shared domain types (Blog, BlogSource, ReadingListItem, etc.)
├── internal/storage/           — SQLite layer: CRUD for all tables
│   └── migrations/            — Embedded SQL migration files (go:embed, auto-applied on startup)
├── internal/feeds/             — RSS fetching (gofeed, parallel), HTML scraping (LinkedIn), content extraction
├── internal/ai/                — AIProvider interface + Anthropic/OpenAI implementations
│   └── skills.go               — Shared prompt templates (filter & rank, summarize)
├── internal/api/               — chi router, middleware, embedded SPA serving
│   ├── handlers/               — JSON API handlers (discover, preferences, reading list, sources)
│   └── dist/                   — Embedded React build output (go:embed)
├── (migrations are in internal/storage/migrations/ — embedded via go:embed)
└── web/                        — React SPA (Vite + TypeScript + shadcn/ui + Tailwind)
    └── src/
        ├── pages/              — Home (discovery), Preferences, ReadingList
        ├── components/         — BlogCard, ReadingItem, ConfirmDialog, Toast, Layout
        └── lib/                — API client, types, utils, theme
```

### Key Design Patterns

- **Embedded SPA**: React build output is copied to `internal/api/dist/` and embedded into the Go binary via `go:embed`. The Go server serves static files with `index.html` fallback for client-side routing.
- **Pluggable AI (strategy pattern)**: `AIProvider` interface in `internal/ai/provider.go` with factory function `NewProvider()`. Anthropic and OpenAI are separate implementations sharing prompt templates from `skills.go`.
- **Pure Go SQLite**: Uses `modernc.org/sqlite` (no CGO) for clean cross-compilation. Single writer, WAL mode, foreign keys ON.
- **Two-pass discovery**: Pass 1 uses RSS title/description for AI filtering (cheap). Pass 2 fetches full article text via go-readability only for the top 10 selected posts before summarization.
- **Dual feed modes**: User-configurable "By Post Count" (N most recent per source) or "By Time Range" (posts within N days). Configurable in Preferences UI.
- **HTML scraping fallback**: Sources with `scrape://` feed URLs (e.g., LinkedIn Engineering) are fetched via HTML parsing instead of RSS. See `internal/feeds/scraper.go`.
- **Persistent discovery**: Results are stored in `discovery_sessions` and restored on page reload via `GET /api/discover/latest`, avoiding redundant AI API calls.

### Data Flow: "Collect Fancy Blogs"

`POST /api/discover` → load preferences + feed settings → fetch RSS/scrape feeds (parallel) → AI filter & rank → extract full content for top 10 → AI summarize each → cache in SQLite → persist session → return JSON with results + failed feeds

### API Routes

All under `/api/*` return JSON. Non-API GET requests serve the React SPA.

- `POST /api/discover` — trigger full discovery pipeline
- `GET /api/discover/latest` — return most recent discovery session results
- `GET/PUT /api/preferences` — user preferences (topics, feed mode, selected sources)
- `GET/POST/PATCH/DELETE /api/reading-list` — reading list CRUD
- `GET /api/sources`, `PUT /api/sources/{id}` — blog source management

## Configuration

Config lives in `config.toml` (gitignored). Copy from `config.example.toml`. API key priority: `AI_API_KEY` env > provider-specific env (`ANTHROPIC_API_KEY`/`OPENAI_API_KEY`) > config file.

Feed settings (mode, post count, lookback days) are also configurable per-user via the Preferences UI and stored in SQLite.

## Go Dependencies

| Package | Purpose |
|---------|---------|
| `modernc.org/sqlite` | Pure Go SQLite driver (no CGO) |
| `github.com/BurntSushi/toml` | TOML config parsing |
| `github.com/go-chi/chi/v5` | HTTP router |
| `github.com/mmcdole/gofeed` | RSS/Atom feed parsing |
| `github.com/go-shiori/go-readability` | Article text extraction |
| `golang.org/x/sync/errgroup` | Concurrent feed fetching |
| `golang.org/x/net/html` | HTML parsing for scraper fallback |

## Coding Style

Follow the style guides in `agent-skills/coding/` (tool-agnostic, works with any AI assistant):
- **Go**: [go-style-guide.md](agent-skills/coding/go-style-guide.md) — Uber style guide, 100 Go Mistakes, unit testing patterns
- **React/TypeScript**: [react-style-guide.md](agent-skills/coding/react-style-guide.md) — Component patterns, TypeScript conventions, Tailwind/shadcn usage

AI business logic skills (prompt templates, LLM patterns) are in `agent-skills/ai-business/`.

## Development Notes

- In dev mode (`make dev`), open `http://localhost:5173` (Vite). Vite proxies `/api/*` to Go on `:8080`.
- In production (`make run`), everything is served from `http://localhost:8080`.
- The app binds to localhost only. No auth needed — if it's running on your machine, you are the user.
- SQLite database lives at `data/app.db`. Migrations run automatically on startup.
- `internal/api/dist/index.html` is a placeholder so Go compiles before the React frontend is built.
- Dark theme with apricot (warm orange) primary accent. Light/dark/system toggle in nav bar.
- UI uses confirmation dialogs for destructive actions and floating toasts for success feedback.
