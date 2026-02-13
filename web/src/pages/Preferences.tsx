import { useState, useEffect } from 'react'
import { Save, Loader2, AlertCircle, Info } from 'lucide-react'
import type { BlogSource, Preferences as PreferencesType } from '@/lib/types'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Switch } from '@/components/ui/switch'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import { Toast } from '@/components/toast'

type FeedMode = 'recent_posts' | 'time_range'

export function Preferences() {
  const [topics, setTopics] = useState('')
  const [sources, setSources] = useState<BlogSource[]>([])
  const [selectedSources, setSelectedSources] = useState<Set<number>>(new Set())
  const [feedMode, setFeedMode] = useState<FeedMode>('recent_posts')
  const [maxArticles, setMaxArticles] = useState(10)
  const [lookbackDays, setLookbackDays] = useState(7)
  const [maxResults, setMaxResults] = useState(10)
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
        if (typeof prefsData.max_results === 'number') {
          setMaxResults(prefsData.max_results)
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
        max_results: maxResults,
      })
      setSuccess(true)
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
        <div>
          <h2 className="text-lg font-semibold">Discovery Settings</h2>
          <p className="text-sm text-muted-foreground">
            Control how many results the AI returns per discovery run.
          </p>
        </div>

        <div className="rounded-lg border bg-muted/30 p-4">
          <div className="flex items-baseline justify-between">
            <label htmlFor="max-results" className="text-sm font-medium">
              Discovery results
            </label>
            <span className="text-2xl font-bold tabular-nums">{maxResults}</span>
          </div>
          <input
            id="max-results"
            type="range"
            min={5}
            max={20}
            step={1}
            value={maxResults}
            onChange={(e) => setMaxResults(Number(e.target.value))}
            className="mt-2 w-full accent-primary"
          />
          <div className="mt-1 flex justify-between text-xs text-muted-foreground">
            <span>5 results</span>
            <span>20 results</span>
          </div>
        </div>
      </div>

      <Separator />

      <div className="space-y-4">
        <div>
          <h2 className="text-lg font-semibold">Feed Settings</h2>
          <p className="text-sm text-muted-foreground">
            Choose how to select posts from each blog source.
          </p>
        </div>

        <div className="grid gap-3 sm:grid-cols-2">
          <button
            type="button"
            onClick={() => setFeedMode('recent_posts')}
            className={`rounded-lg border-2 p-4 text-left transition-colors ${
              feedMode === 'recent_posts'
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-muted-foreground/30'
            }`}
          >
            <span className="text-sm font-medium">By Post Count</span>
            <p className="mt-1 text-xs text-muted-foreground">
              Get the latest posts from each source, regardless of when they were published.
            </p>
          </button>

          <button
            type="button"
            onClick={() => setFeedMode('time_range')}
            className={`rounded-lg border-2 p-4 text-left transition-colors ${
              feedMode === 'time_range'
                ? 'border-primary bg-primary/5'
                : 'border-border hover:border-muted-foreground/30'
            }`}
          >
            <span className="text-sm font-medium">By Time Range</span>
            <p className="mt-1 text-xs text-muted-foreground">
              Only get posts published within a specific number of days.
            </p>
          </button>
        </div>

        {feedMode === 'recent_posts' && (
          <div className="rounded-lg border bg-muted/30 p-4">
            <div className="flex items-baseline justify-between">
              <label htmlFor="max-articles" className="text-sm font-medium">
                Posts per source
              </label>
              <span className="text-2xl font-bold tabular-nums">{maxArticles}</span>
            </div>
            <input
              id="max-articles"
              type="range"
              min={5}
              max={20}
              step={1}
              value={maxArticles}
              onChange={(e) => setMaxArticles(Number(e.target.value))}
              className="mt-2 w-full accent-primary"
            />
            <div className="mt-1 flex justify-between text-xs text-muted-foreground">
              <span>5 posts</span>
              <span>20 posts</span>
            </div>
          </div>
        )}

        {feedMode === 'time_range' && (
          <div className="rounded-lg border bg-muted/30 p-4">
            <div className="flex items-baseline justify-between">
              <label htmlFor="lookback-days" className="text-sm font-medium">
                Look back period
              </label>
              <span className="text-2xl font-bold tabular-nums">
                {lookbackDays} {lookbackDays === 1 ? 'day' : 'days'}
              </span>
            </div>
            <input
              id="lookback-days"
              type="range"
              min={1}
              max={30}
              step={1}
              value={lookbackDays}
              onChange={(e) => setLookbackDays(Number(e.target.value))}
              className="mt-2 w-full accent-primary"
            />
            <div className="mt-1 flex justify-between text-xs text-muted-foreground">
              <span>1 day</span>
              <span>30 days</span>
            </div>
          </div>
        )}
      </div>

      <Separator />

      <div className="space-y-4">
        <div>
          <h2 className="text-lg font-semibold">Blog Sources</h2>
          <p className="text-sm text-muted-foreground">
            Choose which engineering blogs to include in discovery.
          </p>
        </div>

        <div className="flex items-start gap-3 rounded-lg border border-primary/30 bg-primary/5 p-4 text-sm">
          <Info className="mt-0.5 size-4 shrink-0 text-primary" />
          <p className="text-muted-foreground">
            Some blogs may be unreachable depending on your network. If a source
            consistently fails to fetch, consider disabling it.
          </p>
        </div>

        {sources.length > 0 && (
          <div className="overflow-hidden rounded-lg border">
            <div className="flex items-center justify-between border-b bg-muted/30 px-4 py-3">
              <div className="flex items-center gap-3">
                <span className="text-sm font-semibold">All sources</span>
                <span className="text-xs tabular-nums text-muted-foreground">
                  {selectedSources.size} of {sources.length}
                </span>
              </div>
              <Switch
                checked={selectedSources.size === sources.length}
                onCheckedChange={(checked: boolean) => {
                  if (checked) {
                    setSelectedSources(new Set(sources.map((s) => s.id)))
                  } else {
                    setSelectedSources(new Set())
                  }
                }}
                aria-label="Toggle all sources"
                className={`scale-125 origin-right ${
                  selectedSources.size > 0 && selectedSources.size < sources.length
                    ? 'data-[state=unchecked]:bg-primary/40'
                    : selectedSources.size === sources.length
                      ? 'data-[state=checked]:bg-emerald-500'
                      : ''
                }`}
              />
            </div>

            <div className="divide-y">
              {sources.map((source) => (
                <div
                  key={source.id}
                  className="flex items-center justify-between gap-4 px-4 py-3"
                >
                  <div className="min-w-0 flex-1">
                    <div className="flex items-baseline gap-2">
                      <span className="font-medium">{source.company}</span>
                      <span className="text-sm text-muted-foreground">{source.name}</span>
                    </div>
                    <p className="mt-1 truncate text-xs text-muted-foreground">
                      {source.feed_url}
                    </p>
                  </div>
                  <Switch
                    checked={selectedSources.has(source.id)}
                    onCheckedChange={(checked: boolean) =>
                      handleSourceToggle(source.id, checked)
                    }
                    aria-label={`Toggle ${source.company} - ${source.name}`}
                  />
                </div>
              ))}
            </div>
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

      <Toast
        message="Preferences saved successfully."
        visible={success}
        onClose={() => setSuccess(false)}
      />
    </div>
  )
}
