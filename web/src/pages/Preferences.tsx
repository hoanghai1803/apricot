import { useState, useEffect } from 'react'
import { Save, Loader2, AlertCircle, Check } from 'lucide-react'
import type { BlogSource, Preferences as PreferencesType } from '@/lib/types'
import { api } from '@/lib/api'
import { Button } from '@/components/ui/button'
import { Textarea } from '@/components/ui/textarea'
import { Checkbox } from '@/components/ui/checkbox'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'

export function Preferences() {
  const [topics, setTopics] = useState('')
  const [sources, setSources] = useState<BlogSource[]>([])
  const [selectedSources, setSelectedSources] = useState<Set<number>>(new Set())
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
        <h2 className="text-lg font-semibold">Blog Sources</h2>
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
