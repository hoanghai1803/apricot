# Apricot

AI-powered tech blog curator that runs on your machine.

Uses your chosen AI provider (Anthropic Claude, OpenAI) to filter, rank, and summarize engineering blog posts from top tech companies based on your interests. All data stays local — no cloud, no accounts, no tracking.

## Quick Start

```bash
# 1. Clone the repository
git clone https://github.com/hoanghai1803/apricot.git
cd apricot

# 2. Copy the example config and add your API key
cp config.example.toml config.toml
# Edit config.toml: set your AI provider and API key

# 3. Run
make run
# Browser opens to http://localhost:8080
```

## Features

- **Curated feeds** — Pulls from 21 engineering blogs: Netflix, Meta, Uber, AWS, Google, Spotify, Stripe, Cloudflare, LinkedIn, Figma, Vercel, Datadog, and more
- **AI-powered ranking** — Your LLM filters posts to the most relevant for your interests (configurable 5-20 results)
- **Smart summaries** — 4-5 sentence technical summaries so you can decide what's worth a full read
- **Reading list** — Save posts, track reading progress (unread / reading / read), add tags, write notes
- **Custom blog URLs** — Add any blog post URL to your reading list with auto-extracted metadata and AI summary
- **Full-text search** — Search across all cached blog posts from the nav bar
- **Filter tabs** — Filter discovery results by All / New / Added status
- **Configurable feed settings** — Choose between "most recent N posts" or "posts from last N days" per source
- **Persistent results** — Discovery results are saved and restored on page reload (no redundant API calls)
- **Dark / light theme** — Dark navy theme with apricot accent, plus light mode and system preference detection
- **Runs locally** — Single binary, SQLite database, your data never leaves your machine
- **Bring your own key** — Works with Anthropic Claude or OpenAI, you control the cost

## Requirements

**From source:** Go 1.22+ and Node.js 20+

**Pre-built binary:** Just download and run (no dependencies)

An API key for your chosen AI provider:
| Provider | Models | Est. Cost |
|----------|--------|-----------|
| Anthropic | Claude Haiku 4.5, Claude Sonnet 4.5 | ~$1-6/mo |
| OpenAI | GPT-4o-mini, GPT-4o | ~$2-8/mo |

## Configuration

Copy `config.example.toml` to `config.toml` and edit:

```toml
[ai]
provider = "anthropic"          # "anthropic" or "openai"
api_key = ""                    # Your API key
model = "claude-haiku-4-5"      # See supported models above

[server]
port = 8080
auto_open_browser = true

[feeds]
refresh_interval_minutes = 60
max_articles_per_feed = 20
lookback_days = 7
```

**API key** can also be set via environment variable (takes priority over config file):

```bash
AI_API_KEY=sk-... make run              # Generic (works for any provider)
ANTHROPIC_API_KEY=sk-ant-... make run   # Provider-specific
OPENAI_API_KEY=sk-... make run
```

Feed settings (post count vs time range, slider values) are also configurable per-user in the Preferences page and stored in the database.

## How It Works

```
You click "Collect Fancy Blogs"
        |
        v
  Go backend fetches RSS feeds from 21 blogs (parallel)
  + HTML scraping for sites without RSS (e.g., LinkedIn)
        |
        v
  AI filters & ranks by your interests
  (titles + descriptions only -- fast & cheap)
        |
        v
  Top N posts selected (configurable 5-20)
        |
        v
  Full article text extracted for each
        |
        v
  AI generates 4-5 sentence summaries
        |
        v
  Results displayed with summaries & match reasons
  Results cached -- reload the page, they're still there
```

## Blog Sources

21 default sources, all toggleable in Preferences:

Netflix, Meta, Uber, AWS, Google (Research + Cloud), Spotify, LinkedIn, Figma, Datadog, Stripe, Airbnb, Grab, Cloudflare, Slack, GitHub, Vercel, Dropbox, Instacart, Pinterest, Lyft

Some sources use RSS feeds, LinkedIn uses HTML scraping. Sources that are unreachable from your network can be disabled in Preferences.

## Development

```bash
make dev    # Vite dev server (:5173) + Go backend (:8080) with live reload
make test   # Run Go + frontend tests
make clean  # Remove build artifacts
```

In dev mode, open `http://localhost:5173`. Vite proxies API calls to the Go backend.

### Tech Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go, chi router, SQLite (pure Go, no CGO) |
| Frontend | React 19, TypeScript, Vite, shadcn/ui, Tailwind CSS v4 |
| AI | Pluggable provider (Anthropic / OpenAI) via raw HTTP |
| RSS | gofeed (parsing), go-readability (content extraction) |
| Scraping | golang.org/x/net/html (LinkedIn fallback) |

### Project Structure

```
cmd/server/          Entry point
internal/
  config/            TOML config parsing
  models/            Domain types
  storage/           SQLite layer (CRUD)
    migrations/      SQL migration files (embedded via go:embed)
  feeds/             RSS fetching, HTML scraping, content extraction
  ai/                LLM provider interface & implementations
  api/               HTTP router, handlers, embedded SPA
web/                 React SPA
  src/pages/         Home (discovery), Preferences, ReadingList
  src/components/    BlogCard, ReadingItem, ConfirmDialog, Toast, Layout
  src/lib/           API client, types, utils, theme
docs/                Internal brainstorm & planning docs (gitignored)
```

## License

[MIT](LICENSE)
