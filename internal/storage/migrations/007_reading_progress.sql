-- Reading progress tracking: estimated reading time per blog, scroll progress per reading list item.
ALTER TABLE blogs ADD COLUMN reading_time_minutes INTEGER;
ALTER TABLE reading_list ADD COLUMN progress INTEGER NOT NULL DEFAULT 0;
