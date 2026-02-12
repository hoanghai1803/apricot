import { useState, useEffect } from 'react'
import { AlertCircle, Info } from 'lucide-react'
import type { BlogSource } from '@/lib/types'
import { api } from '@/lib/api'
import { Skeleton } from '@/components/ui/skeleton'
import { SourceToggle } from '@/components/source-toggle'

export function Sources() {
  const [sources, setSources] = useState<BlogSource[]>([])
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)

  useEffect(() => {
    async function fetchSources() {
      setLoading(true)
      setError(null)

      try {
        const data = await api.get<BlogSource[]>('/api/sources')
        data.sort((a, b) => a.company.localeCompare(b.company))
        setSources(data)
      } catch (err) {
        setError(err instanceof Error ? err.message : 'Failed to load sources')
      } finally {
        setLoading(false)
      }
    }

    void fetchSources()
  }, [])

  async function handleToggle(id: number, active: boolean) {
    setSources((prev) =>
      prev.map((s) => (s.id === id ? { ...s, is_active: active } : s))
    )

    try {
      await api.put(`/api/sources/${id}`, { is_active: active })
    } catch {
      setSources((prev) =>
        prev.map((s) => (s.id === id ? { ...s, is_active: !active } : s))
      )
    }
  }

  const activeCount = sources.filter((s) => s.is_active).length
  const totalCount = sources.length

  if (loading) {
    return (
      <div className="space-y-8">
        <div className="space-y-2">
          <Skeleton className="h-8 w-48" />
          <Skeleton className="h-5 w-80" />
        </div>
        <Skeleton className="h-5 w-40" />
        <div className="space-y-3">
          {Array.from({ length: 6 }).map((_, i) => (
            <Skeleton key={i} className="h-[76px] w-full rounded-lg" />
          ))}
        </div>
      </div>
    )
  }

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Blog Sources</h1>
        <p className="mt-1 text-muted-foreground">
          Enable or disable sources for blog discovery.
        </p>
      </div>

      {error && (
        <div className="flex items-start gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm">
          <AlertCircle className="mt-0.5 size-4 shrink-0 text-destructive" />
          <p>{error}</p>
        </div>
      )}

      <div className="flex items-start gap-3 rounded-lg border border-blue-500/30 bg-blue-500/5 p-4 text-sm">
        <Info className="mt-0.5 size-4 shrink-0 text-blue-600 dark:text-blue-400" />
        <p className="text-muted-foreground">
          Some blogs may be unreachable depending on your network. If a source
          consistently fails to fetch, consider disabling it.
        </p>
      </div>

      <p className="text-sm text-muted-foreground">
        {activeCount} of {totalCount} sources active
      </p>

      {sources.length === 0 ? (
        <div className="py-12 text-center">
          <p className="text-muted-foreground">No sources configured yet.</p>
        </div>
      ) : (
        <div className="space-y-3">
          {sources.map((source) => (
            <SourceToggle
              key={source.id}
              source={source}
              onToggle={handleToggle}
            />
          ))}
        </div>
      )}
    </div>
  )
}
