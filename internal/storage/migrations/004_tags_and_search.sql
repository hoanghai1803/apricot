-- Tags: tag registry + many-to-many join table with reading_list
CREATE TABLE IF NOT EXISTS tags (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    name       TEXT    NOT NULL UNIQUE,
    created_at TEXT    NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS reading_list_tags (
    reading_list_id INTEGER NOT NULL REFERENCES reading_list(id) ON DELETE CASCADE,
    tag_id          INTEGER NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    PRIMARY KEY (reading_list_id, tag_id)
);

CREATE INDEX IF NOT EXISTS idx_reading_list_tags_tag ON reading_list_tags(tag_id);

-- FTS5: full-text search on blogs (title, description, full_content)
CREATE VIRTUAL TABLE IF NOT EXISTS blogs_fts USING fts5(
    title,
    description,
    full_content,
    content='blogs',
    content_rowid='id'
);

-- Triggers to keep FTS in sync with blogs table
CREATE TRIGGER IF NOT EXISTS blogs_fts_insert AFTER INSERT ON blogs BEGIN
    INSERT INTO blogs_fts(rowid, title, description, full_content)
    VALUES (new.id, new.title, COALESCE(new.description, ''), COALESCE(new.full_content, ''));
END;

CREATE TRIGGER IF NOT EXISTS blogs_fts_update AFTER UPDATE ON blogs BEGIN
    INSERT INTO blogs_fts(blogs_fts, rowid, title, description, full_content)
    VALUES ('delete', old.id, old.title, COALESCE(old.description, ''), COALESCE(old.full_content, ''));
    INSERT INTO blogs_fts(rowid, title, description, full_content)
    VALUES (new.id, new.title, COALESCE(new.description, ''), COALESCE(new.full_content, ''));
END;

CREATE TRIGGER IF NOT EXISTS blogs_fts_delete AFTER DELETE ON blogs BEGIN
    INSERT INTO blogs_fts(blogs_fts, rowid, title, description, full_content)
    VALUES ('delete', old.id, old.title, COALESCE(old.description, ''), COALESCE(old.full_content, ''));
END;

-- Backfill FTS with existing blog data
INSERT INTO blogs_fts(rowid, title, description, full_content)
SELECT id, title, COALESCE(description, ''), COALESCE(full_content, '')
FROM blogs;
