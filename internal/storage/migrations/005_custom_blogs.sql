-- Add a "Custom" sentinel source for user-added blogs.
INSERT OR IGNORE INTO blog_sources (name, company, feed_url, site_url, is_active)
VALUES ('Other Blog', 'Custom', 'custom://user-added', 'https://custom', 0);

-- Add optional display-name override for user-added blogs.
ALTER TABLE blogs ADD COLUMN custom_source TEXT;
