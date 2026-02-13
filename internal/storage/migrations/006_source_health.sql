-- Track fetch health per source.
ALTER TABLE blog_sources ADD COLUMN last_fetch_at TEXT;
ALTER TABLE blog_sources ADD COLUMN last_fetch_ok INTEGER NOT NULL DEFAULT 1;
ALTER TABLE blog_sources ADD COLUMN last_error TEXT;
