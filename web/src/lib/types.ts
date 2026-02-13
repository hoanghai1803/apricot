export interface BlogSource {
  id: number
  name: string
  company: string
  feed_url: string
  site_url: string
  is_active: boolean
  last_fetch_at?: string
  last_fetch_ok: boolean
  last_error?: string
  created_at: string
}

export interface Blog {
  id: number
  source_id: number
  source?: string
  title: string
  url: string
  description?: string
  full_content?: string
  published_at?: string
  fetched_at: string
  content_hash?: string
  reading_time_minutes?: number
  created_at: string
}

export interface ReadingListItem {
  id: number
  blog_id: number
  blog?: Blog
  summary?: string
  status: 'unread' | 'reading' | 'read'
  progress: number
  notes?: string
  tags: string[]
  added_at: string
  read_at?: string
}

export interface DiscoverResult {
  id: number
  title: string
  url: string
  source: string
  published_at?: string
  reading_time_minutes?: number
  summary: string
  reason: string
}

export interface FailedFeed {
  source: string
  error: string
}

export interface DiscoverResponse {
  results: DiscoverResult[]
  failed_feeds: FailedFeed[]
  session_id: number
  created_at: string
}

export interface Preferences {
  topics?: string
  selected_sources?: number[]
  feed_mode?: string
  max_articles_per_feed?: number
  lookback_days?: number
  max_results?: number
  timezone?: string
  [key: string]: unknown
}
