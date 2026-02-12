import { useState, useEffect } from 'react'
import { Sparkles, AlertCircle, ChevronDown, ChevronUp, AlertTriangle } from 'lucide-react'
import type { DiscoverResult, DiscoverResponse, FailedFeed } from '@/lib/types'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { BlogCard } from '@/components/blog-card'

function formatLastDiscovered(dateStr: string): string {
  if (!dateStr || dateStr === '0001-01-01T00:00:00Z') return ''
  const date = new Date(dateStr)
  return date.toLocaleString('en-US', {
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
  const [failedExpanded, setFailedExpanded] = useState(false)
  const [loading, setLoading] = useState(false)
  const [loadingLatest, setLoadingLatest] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [addedIds, setAddedIds] = useState<Set<number>>(new Set())
  const [hasSearched, setHasSearched] = useState(false)
  const [lastDiscoveredAt, setLastDiscoveredAt] = useState('')

  useEffect(() => {
    async function loadLatest() {
      try {
        const data = await api.get<DiscoverResponse>('/api/discover/latest')
        if (data.results && data.results.length > 0) {
          setResults(data.results)
          setFailedFeeds(data.failed_feeds ?? [])
          setLastDiscoveredAt(data.created_at)
          setHasSearched(true)
        }
      } catch {
        // Silently ignore -- the user can trigger a fresh discovery.
      } finally {
        setLoadingLatest(false)
      }
    }

    void loadLatest()
  }, [])

  async function handleDiscover() {
    setLoading(true)
    setError(null)
    setResults([])
    setFailedFeeds([])
    setAddedIds(new Set())
    setFailedExpanded(false)

    try {
      const data = await api.post<DiscoverResponse>('/api/discover')
      setResults(data.results)
      setFailedFeeds(data.failed_feeds ?? [])
      setLastDiscoveredAt(data.created_at)
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
      setAddedIds((prev) => {
        const next = new Set(prev)
        next.delete(blogId)
        return next
      })
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
        <Button
          size="lg"
          onClick={handleDiscover}
          disabled={loading}
          className="gap-2"
        >
          <Sparkles className="size-5" />
          {loading ? 'Discovering...' : 'Collect Fancy Blogs'}
        </Button>

        {lastDiscoveredAt && !loading && (
          <p className="text-xs text-muted-foreground">
            Last discovered: {formatLastDiscovered(lastDiscoveredAt)}
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
        <div className="space-y-4">
          {results.map((blog) => (
            <BlogCard
              key={blog.id}
              blog={blog}
              onAddToReadingList={handleAddToReadingList}
              isAdded={addedIds.has(blog.id)}
            />
          ))}
        </div>
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
              Could not reach {failedFeeds.length} source{failedFeeds.length === 1 ? '' : 's'}
            </span>
            {failedExpanded ? (
              <ChevronUp className="size-4" />
            ) : (
              <ChevronDown className="size-4" />
            )}
          </button>

          {failedExpanded && (
            <ul className="mt-3 space-y-1 text-sm text-yellow-700 dark:text-yellow-400">
              {failedFeeds.map((feed) => (
                <li key={feed.source} className="flex items-start gap-2">
                  <span className="shrink-0 font-medium">{feed.source}:</span>
                  <span className="text-yellow-600 dark:text-yellow-500">{feed.error}</span>
                </li>
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
    </div>
  )
}
