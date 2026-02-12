import { useState, useEffect } from 'react'
import { Save, Loader2, AlertCircle, Check, Info } from 'lucide-react'
import type { BlogSource, Preferences as PreferencesType } from '@/lib/types'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'

type FeedMode = 'recent_posts' | 'time_range'

export function Preferences() {
  const [topics, setTopics] = useState('')
  const [sources, setSources] = useState<BlogSource[]>([])
  const [selectedSources, setSelectedSources] = useState<Set<number>>(new Set())
  const [feedMode, setFeedMode] = useState<FeedMode>('recent_posts')
  const [maxArticles, setMaxArticles] = useState(10)
  const [lookbackDays, setLookbackDays] = useState(7)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState(false)

  useEffect(() => {
    async function fetchData() {
      setLoading(true)
      setError(null)

      try {
        const [sourcesData, prefsData] = await Promise.all([
          api.get<BlogSource[]>('/api/sources'),
          api.get<PreferencesType>('/api/preferences'),
        ])

        setSources(sourcesData)

        if (prefsData.topics) {
          setTopics(prefsData.topics)
        }
        if (prefsData.selected_sources) {
          setSelectedSources(new Set(prefsData.selected_sources))
        }
        if (prefsData.feed_mode === 'recent_posts' || prefsData.feed_mode === 'time_range') {
          setFeedMode(prefsData.feed_mode)
        }
        if (typeof prefsData.max_articles_per_feed === 'number') {
          setMaxArticles(prefsData.max_articles_per_feed)
        }
        if (typeof prefsData.lookback_days === 'number') {
          setLookbackDays(prefsData.lookback_days)
        }
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load preferences')
      } finally {
        setLoading(false)
      }
    }

    void fetchData()
  }, [])

  function handleSourceToggle(sourceId: number, checked: boolean) {
    setSelectedSources((prev) => {
      const next = new Set(prev)
      if (checked) {
        next.add(sourceId)
      } else {
        next.delete(sourceId)
      }
      return next
    })
  }

  async function handleSave() {
    setSaving(true)
    setError(null)
    setSuccess(false)

    try {
      await api.put('/api/preferences', {
        topics,
        selected_sources: Array.from(selectedSources),
        feed_mode: feedMode,
        max_articles_per_feed: maxArticles,
        lookback_days: lookbackDays,
      })
      setSuccess(true)
      setTimeout(() => setSuccess(false), 3000)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to save preferences')
    } finally {
      setSaving(false)
    }
  }

  if (loading) {
    return (
      <div className="space-y-8">
        <div className="space-y-2">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-5 w-72" />
        </div>
        <Skeleton className="h-32 w-full" />
        <Separator />
        <div className="space-y-2">
          <Skeleton className="h-6 w-36" />
          {Array.from({ length: 5 }).map((_, i) => (
            <Skeleton key={i} className="h-6 w-64" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Your Interests</h1>
        <p className="mt-1 text-muted-foreground">
          Tell us what topics you care about and which blogs to follow.
        </p>
      </div>

      {error && (
        <div className="flex items-start gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm">
          <AlertCircle className="mt-0.5 size-4 shrink-0 text-destructive" />
          <p>{error}</p>
        </div>
      )}

      {success && (
        <div className="flex items-center gap-3 rounded-lg border border-green-500/50 bg-green-500/10 p-4 text-sm">
          <Check className="size-4 shrink-0 text-green-600 dark:text-green-400" />
          <p>Preferences saved successfully.</p>
        </div>
      )}

      <div className="space-y-2">
        <label htmlFor="topics" className="text-sm font-medium">
          Topics
        </label>
        <Textarea
          id="topics"
          value={topics}
          onChange={(e) => setTopics(e.target.value)}
          placeholder="e.g., system design, distributed databases, data pipelines, AI/ML infrastructure..."
          rows={4}
          className="resize-y"
        />
        <p className="text-xs text-muted-foreground">
          Enter your interests as a comma-separated list or free-form description.
        </p>
      </div>

      <Separator />

      <div className="space-y-4">
        <h2 className="text-lg font-semibold">Feed Settings</h2>

        <div className="space-y-3">
          <label className="flex cursor-pointer items-start gap-3">
            <input
              type="radio"
              name="feed_mode"
              value="recent_posts"
              checked={feedMode === 'recent_posts'}
              onChange={() => setFeedMode('recent_posts')}
              className="mt-1"
            />
            <div>
              <span className="text-sm font-medium">Most Recent Posts</span>
              <p className="text-xs text-muted-foreground">
                Fetch the N most recent posts from each blog source.
              </p>
            </div>
          </label>

          {feedMode === 'recent_posts' && (
            <div className="ml-6 space-y-1">
              <label htmlFor="max-articles" className="text-sm text-muted-foreground">
                Posts per source: {maxArticles}
              </label>
              <input
                id="max-articles"
                type="range"
                min={5}
                max={20}
                value={maxArticles}
                onChange={(e) => setMaxArticles(Number(e.target.value))}
                className="w-full max-w-xs"
              />
              <div className="flex justify-between text-xs text-muted-foreground" style={{ maxWidth: '20rem' }}>
                <span>5</span>
                <span>20</span>
              </div>
            </div>
          )}

          <label className="flex cursor-pointer items-start gap-3">
            <input
              type="radio"
              name="feed_mode"
              value="time_range"
              checked={feedMode === 'time_range'}
              onChange={() => setFeedMode('time_range')}
              className="mt-1"
            />
            <div>
              <span className="text-sm font-medium">Time Range</span>
              <p className="text-xs text-muted-foreground">
                Fetch posts published within the last N days.
              </p>
            </div>
          </label>

          {feedMode === 'time_range' && (
            <div className="ml-6 space-y-1">
              <label htmlFor="lookback-days" className="text-sm text-muted-foreground">
                Lookback: {lookbackDays} day{lookbackDays === 1 ? '' : 's'}
              </label>
              <input
                id="lookback-days"
                type="range"
                min={1}
                max={30}
                value={lookbackDays}
                onChange={(e) => setLookbackDays(Number(e.target.value))}
                className="w-full max-w-xs"
              />
              <div className="flex justify-between text-xs text-muted-foreground" style={{ maxWidth: '20rem' }}>
                <span>1 day</span>
                <span>30 days</span>
              </div>
            </div>
          )}
        </div>
      </div>

      <Separator />

      <div className="space-y-4">
        <h2 className="text-lg font-semibold">Blog Sources</h2>

        <div className="flex items-start gap-3 rounded-lg border border-blue-500/30 bg-blue-500/5 p-4 text-sm">
          <Info className="mt-0.5 size-4 shrink-0 text-blue-600 dark:text-blue-400" />
          <p className="text-muted-foreground">
            Some blogs may be unreachable from your network. Blogs hosted on Medium
            (Netflix, Airbnb, Pinterest, Lyft) may require a VPN in certain regions. If a
            source consistently fails to fetch, consider disabling it.
          </p>
        </div>

        {sources.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            No sources available. Add sources on the{' '}
            <a href="/sources" className="text-foreground underline underline-offset-4">
              Sources page
            </a>
            .
          </p>
        ) : (
          <div className="space-y-3">
            {sources.map((source) => (
              <label
                key={source.id}
                className="flex cursor-pointer items-center gap-3"
              >
                <Checkbox
                  checked={selectedSources.has(source.id)}
                  onCheckedChange={(checked) =>
                    handleSourceToggle(source.id, checked === true)
                  }
                />
                <span className="text-sm">
                  <span className="font-medium">{source.company}</span>
                  {' '}
                  <span className="text-muted-foreground">- {source.name}</span>
                </span>
              </label>
            ))}
          </div>
        )}
      </div>

      <div>
        <Button onClick={handleSave} disabled={saving} className="gap-2">
          {saving ? (
            <Loader2 className="size-4 animate-spin" />
          ) : (
            <Save className="size-4" />
          )}
          {saving ? 'Saving...' : 'Save Preferences'}
        </Button>
      </div>
    </div>
  )
}
