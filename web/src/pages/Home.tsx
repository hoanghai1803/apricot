import { useState, useEffect, useMemo } from 'react'
import { useBlocker } from 'react-router-dom'
import { Sparkles, Shuffle, AlertCircle, ChevronDown, ChevronUp, AlertTriangle } from 'lucide-react'
import type { DiscoverResult, DiscoverResponse, FailedFeed, ReadingListItem, Preferences } from '@/lib/types'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { BlogCard } from '@/components/blog-card'
import { ConfirmDialog } from '@/components/confirm-dialog'

function formatLastDiscovered(dateStr: string, timezone: string): string {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return ''
  const date = new Date(dateStr)
  return date.toLocaleString('en-US', {
    timeZone: timezone,
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  })
}

export function Home() {
  const [results, setResults] = useState<DiscoverResult[]>([])
  const [failedFeeds, setFailedFeeds] = useState<FailedFeed[]>([])
  const [filter, setFilter] = useState<'all' | 'new' | 'added'>('all')
  const [failedExpanded, setFailedExpanded] = useState(false)
  const [loading, setLoading] = useState(false)
  const [loadingLatest, setLoadingLatest] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [addedIds, setAddedIds] = useState<Set<number>>(new Set())
  const [hasSearched, setHasSearched] = useState(false)
  const [lastDiscoveredAt, setLastDiscoveredAt] = useState('')
  const [timezone, setTimezone] = useState('UTC')

  const blocker = useBlocker(loading)

  const filterCounts = useMemo(() => {
    const newCount = results.filter((b) => !addedIds.has(b.id)).length
    const addedCount = results.filter((b) => addedIds.has(b.id)).length
    return { all: results.length, new: newCount, added: addedCount }
  }, [results, addedIds])

  useEffect(() => {
    async function loadLatest() {
      try {
        const [discoverData, readingList, prefs] = await Promise.all([
          api.get<DiscoverResponse>('/api/discover/latest').catch(() => null),
          api.get<ReadingListItem[]>('/api/reading-list').catch((): ReadingListItem[] => []),
          api.get<Preferences>('/api/preferences').catch(() => null),
        ])

        if (prefs?.timezone) {
          setTimezone(prefs.timezone)
        }

        if (discoverData?.results && discoverData.results.length > 0) {
          setResults(discoverData.results)
          setFailedFeeds(discoverData.failed_feeds ?? [])
          setLastDiscoveredAt(discoverData.created_at)
          setHasSearched(true)
        }

        if (readingList.length > 0) {
          setAddedIds(new Set(readingList.map((item) => item.blog_id)))
        }
      } finally {
        setLoadingLatest(false)
      }
    }

    void loadLatest()
  }, [])

  async function handleDiscover(mode: 'normal' | 'serendipity' = 'normal') {
    setLoading(true)
    setError(null)
    setResults([])
    setFailedFeeds([])
    setAddedIds(new Set())
    setFailedExpanded(false)

    try {
      const data = await api.post<DiscoverResponse>('/api/discover', { mode })
      setResults(data.results)
      setFailedFeeds(data.failed_feeds ?? [])
      setLastDiscoveredAt(new Date().toISOString())
      setHasSearched(true)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to discover blogs')
      setHasSearched(true)
    } finally {
      setLoading(false)
    }
  }

  async function handleAddToReadingList(blogId: number) {
    setAddedIds((prev) => new Set(prev).add(blogId))

    try {
      await api.post('/api/reading-list', { blog_id: blogId })
    } catch {
      // Keep as added — most likely a duplicate (already in reading list).
    }
  }

  if (loadingLatest) {
    return (
      <div className="space-y-8">
        <div className="space-y-3 text-center">
          <Skeleton className="mx-auto h-10 w-80" />
          <Skeleton className="mx-auto h-5 w-96" />
        </div>
        <div className="flex justify-center">
          <Skeleton className="h-11 w-48" />
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <div className="space-y-3 text-center">
        <h1 className="text-3xl font-bold tracking-tight sm:text-4xl">
          Discover Engineering Blogs
        </h1>
        <p className="mx-auto max-w-2xl text-muted-foreground">
          AI-curated posts from top tech companies, tailored to your interests
        </p>
      </div>

      <div className="flex flex-col items-center gap-2">
        <div className="flex items-center gap-3">
          <Button
            size="lg"
            onClick={() => handleDiscover('normal')}
            disabled={loading}
            className="gap-2"
          >
            <Sparkles className="size-5" />
            {loading ? 'Discovering...' : 'Collect Fancy Blogs'}
          </Button>
          <Button
            size="lg"
            variant="outline"
            onClick={() => handleDiscover('serendipity')}
            disabled={loading}
            className="gap-2"
          >
            <Shuffle className="size-5" />
            Surprise Me
          </Button>
        </div>

        {!loading && formatLastDiscovered(lastDiscoveredAt, timezone) && (
          <p className="text-xs text-muted-foreground">
            Last discovered: {formatLastDiscovered(lastDiscoveredAt, timezone)}
          </p>
        )}
      </div>

      {error && (
        <div className="mx-auto flex max-w-2xl items-start gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm">
          <AlertCircle className="mt-0.5 size-4 shrink-0 text-destructive" />
          <p>{error}</p>
        </div>
      )}

      {loading && (
        <div className="space-y-4">
          {Array.from({ length: 4 }).map((_, i) => (
            <div key={i} className="space-y-4 rounded-xl border p-6">
              <div className="flex items-center justify-between">
                <Skeleton className="h-5 w-24" />
                <Skeleton className="h-4 w-20" />
              </div>
              <Skeleton className="h-6 w-3/4" />
              <div className="space-y-2">
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-full" />
                <Skeleton className="h-4 w-2/3" />
              </div>
              <Skeleton className="h-4 w-1/2" />
              <div className="flex gap-2">
                <Skeleton className="h-8 w-32" />
                <Skeleton className="h-8 w-40" />
              </div>
            </div>
          ))}
        </div>
      )}

      {!loading && results.length > 0 && (
        <Tabs value={filter} onValueChange={(v) => setFilter(v as typeof filter)}>
          <TabsList>
            {([
              ['all', 'All'],
              ['new', 'New'],
              ['added', 'Added'],
            ] as const).map(([value, label]) => (
              <TabsTrigger key={value} value={value} className="gap-2">
                {label}
                <Badge variant="secondary" className="ml-1 px-1.5 py-0 text-[10px]">
                  {filterCounts[value]}
                </Badge>
              </TabsTrigger>
            ))}
          </TabsList>

          {(['all', 'new', 'added'] as const).map((tab) => {
            const filtered =
              tab === 'new'
                ? results.filter((b) => !addedIds.has(b.id))
                : tab === 'added'
                  ? results.filter((b) => addedIds.has(b.id))
                  : results

            return (
              <TabsContent key={tab} value={tab} className="mt-6">
                {filtered.length > 0 ? (
                  <div className="space-y-4">
                    {filtered.map((blog) => (
                      <BlogCard
                        key={blog.id}
                        blog={blog}
                        onAddToReadingList={handleAddToReadingList}
                        isAdded={addedIds.has(blog.id)}
                      />
                    ))}
                  </div>
                ) : (
                  <p className="py-8 text-center text-muted-foreground">
                    No {tab === 'new' ? 'new' : 'added'} posts to show.
                  </p>
                )}
              </TabsContent>
            )
          })}
        </Tabs>
      )}

      {!loading && failedFeeds.length > 0 && (
        <div className="mx-auto max-w-2xl rounded-lg border border-yellow-500/50 bg-yellow-500/10 p-4">
          <button
            type="button"
            onClick={() => setFailedExpanded((prev) => !prev)}
            className="flex w-full items-center justify-between text-sm font-medium text-yellow-700 dark:text-yellow-400"
          >
            <span className="flex items-center gap-2">
              <AlertTriangle className="size-4" />
              Could not reach {failedFeeds.length} source{failedFeeds.length === 1 ? '' : 's'} due to network connection issues
            </span>
            {failedExpanded ? (
              <ChevronUp className="size-4" />
            ) : (
              <ChevronDown className="size-4" />
            )}
          </button>

          {failedExpanded && (
            <ul className="mt-3 space-y-1 pl-6 text-sm text-yellow-700 dark:text-yellow-400 list-disc">
              {failedFeeds.map((feed) => (
                <li key={feed.source}>{feed.source}</li>
              ))}
            </ul>
          )}
        </div>
      )}

      {!loading && hasSearched && results.length === 0 && !error && (
        <div className="py-12 text-center">
          <p className="text-muted-foreground">
            No results found. Try updating your{' '}
            <a href="/preferences" className="text-foreground underline underline-offset-4">
              preferences
            </a>{' '}
            or enabling more{' '}
            <a href="/sources" className="text-foreground underline underline-offset-4">
              sources
            </a>
            .
          </p>
        </div>
      )}

      <ConfirmDialog
        open={blocker.state === 'blocked'}
        onOpenChange={(open) => {
          if (!open) blocker.reset?.()
        }}
        title="Discovery in progress"
        description="Blog discovery is still running. If you leave now, the current results will be lost. Are you sure?"
        confirmLabel="Leave"
        cancelLabel="Stay"
        variant="destructive"
        onConfirm={() => blocker.proceed?.()}
      />
    </div>
  )
}
