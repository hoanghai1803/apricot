# Apricot

AI-powered tech blog curator that runs on your machine.

Uses your chosen AI provider (Anthropic Claude, OpenAI) to filter, rank, and summarize engineering blog posts from 20+ top tech companies based on your interests. All data stays local — no cloud, no accounts, no tracking.

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

- **Curated feeds** — Pulls from 20+ engineering blogs: Netflix, Uber, AWS, Google, Spotify, Meta, Stripe, Cloudflare, and more
- **AI-powered ranking** — Your LLM filters hundreds of posts down to the 10 most relevant to your interests
- **Smart summaries** — 4-5 sentence technical summaries so you can decide what's worth a full read
- **Reading list** — Save posts, track what you've read, add personal notes
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
# Generic (works for any provider)
AI_API_KEY=sk-... make run

# Provider-specific
ANTHROPIC_API_KEY=sk-ant-... make run
OPENAI_API_KEY=sk-... make run
```

## How It Works

```
You click "Collect Fancy Blogs"
        │
        ▼
┌─ Go backend fetches RSS feeds from 20+ blogs (parallel) ─┐
│  Netflix, Uber, AWS, Google, Spotify, Meta, Stripe, ...   │
└───────────────────────┬───────────────────────────────────┘
                        ▼
         AI filters & ranks by your interests
            (titles + descriptions only — fast & cheap)
                        │
                        ▼
              Top 10 posts selected
                        │
                        ▼
         Full article text extracted for each
                        │
                        ▼
         AI generates 4-5 sentence summaries
                        │
                        ▼
     Results displayed with summaries & match reasons
     ┌──────────────────────────────────────────────┐
     │  "Add to Reading List"  │  "Read Original"   │
     └──────────────────────────────────────────────┘
```

Everything is cached in a local SQLite database. Summaries are never regenerated for posts you've already seen.

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
| Frontend | React, TypeScript, Vite, shadcn/ui, Tailwind CSS |
| AI | Pluggable provider (Anthropic / OpenAI) via raw HTTP |
| RSS | gofeed (parsing), go-readability (content extraction) |

### Project Structure

```
cmd/server/          Entry point
internal/
  config/            TOML config parsing
  models/            Domain types
  storage/           SQLite layer (migrations, CRUD)
  feeds/             RSS fetching & content extraction
  ai/                LLM provider interface & implementations
  api/               HTTP router, handlers, embedded SPA
migrations/          SQL migration files
web/                 React SPA (Vite + TypeScript + shadcn/ui)
```

## Pre-Built Binaries

Download from the [Releases](https://github.com/hoanghai1803/apricot/releases) page:

```
apricot-darwin-arm64        macOS Apple Silicon
apricot-darwin-amd64        macOS Intel
apricot-linux-amd64         Linux x86_64
apricot-linux-arm64         Linux ARM64
apricot-windows-amd64.exe   Windows x86_64
```

## License

[MIT](LICENSE)
