-- Remove broken blog sources. New replacements are added via SeedDefaults.
DELETE FROM blog_sources WHERE feed_url IN (
    'https://cloud.google.com/feeds/cloudblog-google-cloud.xml',
    'https://shopifyengineering.myshopify.com/blogs/engineering.atom',
    'https://engineering.linkedin.com/blog.rss.html',
    'https://doordash.engineering/feed/',
    'https://blog.x.com/engineering/en_us/blog.rss'
);
