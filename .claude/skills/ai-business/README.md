# AI Business Logic Skills

This folder contains skills for the AI-powered features of Apricot: prompt templates, LLM integration patterns, and content processing pipelines.

## Planned Contents

### Prompt Templates
- **Filter & Rank** -- Prompt template for the two-pass discovery pipeline (Pass 1: RSS title/description filtering, Pass 2: full-content ranking)
- **Summarize** -- Prompt template for generating concise blog post summaries
- **Future: Rewrite** -- Prompt templates for content transformation (simplify, translate, extract key points)

### LLM Provider Integration
- Provider abstraction patterns (Anthropic, OpenAI, future providers)
- Token budget management and cost estimation
- Rate limiting and retry strategies
- Response parsing and structured output extraction

### Content Processing Pipelines
- RSS feed ingestion and deduplication
- Full-text extraction with go-readability
- Two-pass discovery flow (cheap filter, then expensive summarize)
- Caching strategy for AI-generated content in SQLite

## Status

This folder will be populated as we build the AI features. The current implementation lives in:
- `internal/ai/provider.go` -- AIProvider interface and factory
- `internal/ai/anthropic.go` -- Anthropic Claude implementation
- `internal/ai/openai.go` -- OpenAI GPT implementation
- `internal/ai/skills.go` -- Shared prompt templates (filter & rank, summarize)
