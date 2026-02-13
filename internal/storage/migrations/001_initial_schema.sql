-- Blog sources (the engineering blogs we track)
CREATE TABLE blog_sources (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL,
    company     TEXT NOT NULL,
    feed_url    TEXT NOT NULL UNIQUE,
    site_url    TEXT NOT NULL,
    is_active   INTEGER NOT NULL DEFAULT 1,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Individual blog posts discovered from RSS feeds
CREATE TABLE blogs (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    source_id       INTEGER NOT NULL REFERENCES blog_sources(id),
    title           TEXT NOT NULL,
    url             TEXT NOT NULL UNIQUE,
    description     TEXT,
    full_content    TEXT,
    published_at    TEXT,
    fetched_at      TEXT NOT NULL DEFAULT (datetime('now')),
    content_hash    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX idx_blogs_published ON blogs(published_at DESC);
CREATE INDEX idx_blogs_source ON blogs(source_id);
CREATE INDEX idx_blogs_url ON blogs(url);

-- Cached AI-generated summaries
CREATE TABLE blog_summaries (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    blog_id     INTEGER NOT NULL UNIQUE REFERENCES blogs(id),
    summary     TEXT NOT NULL,
    model_used  TEXT NOT NULL,
    created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- User preferences (single user, so no user_id needed)
CREATE TABLE preferences (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    key         TEXT NOT NULL UNIQUE,
    value       TEXT NOT NULL,
    updated_at  TEXT NOT NULL DEFAULT (datetime('now'))
);

-- Reading list / wishlist
CREATE TABLE reading_list (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    blog_id     INTEGER NOT NULL UNIQUE REFERENCES blogs(id),
    status      TEXT NOT NULL DEFAULT 'unread',
    notes       TEXT,
    added_at    TEXT NOT NULL DEFAULT (datetime('now')),
    read_at     TEXT
);

CREATE INDEX idx_reading_status ON reading_list(status);

-- Discovery sessions (audit trail of each discovery run)
CREATE TABLE discovery_sessions (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,
    preferences_snapshot TEXT NOT NULL,
    blogs_considered     INTEGER NOT NULL,
    blogs_selected       TEXT NOT NULL,
    model_used           TEXT NOT NULL,
    input_tokens         INTEGER,
    output_tokens        INTEGER,
    created_at           TEXT NOT NULL DEFAULT (datetime('now'))
);
