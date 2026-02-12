import { useState } from 'react'
import { Sparkles, AlertCircle } from 'lucide-react'
import type { DiscoverResult } from '@/lib/types'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Skeleton } from '@/components/ui/skeleton'
import { BlogCard } from '@/components/blog-card'

export function Home() {
  const [results, setResults] = useState<DiscoverResult[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [addedIds, setAddedIds] = useState<Set<number>>(new Set())
  const [hasSearched, setHasSearched] = useState(false)

  async function handleDiscover() {
    setLoading(true)
    setError(null)
    setResults([])
    setAddedIds(new Set())

    try {
      const data = await api.post<DiscoverResult[]>('/api/discover')
      setResults(data)
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

      <div className="flex justify-center">
        <Button
          size="lg"
          onClick={handleDiscover}
          disabled={loading}
          className="gap-2"
        >
          <Sparkles className="size-5" />
          {loading ? 'Discovering...' : 'Collect Fancy Blogs'}
        </Button>
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
