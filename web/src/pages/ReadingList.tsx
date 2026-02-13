import { useState, useEffect, useCallback } from 'react'
import { AlertCircle, X } from 'lucide-react'
import type { ReadingListItem } from '@/lib/types'
import { api } from '@/lib/api'
import { Tabs, TabsList, TabsTrigger, TabsContent } from '@/components/ui/tabs'
import { Badge } from '@/components/ui/badge'
import { Skeleton } from '@/components/ui/skeleton'
import { ReadingItem } from '@/components/reading-item'

type TabStatus = 'unread' | 'reading' | 'read'

const tabConfig: { value: TabStatus; label: string; emptyMessage: string }[] = [
  { value: 'unread', label: 'Unread', emptyMessage: 'No unread posts' },
  { value: 'reading', label: 'Reading', emptyMessage: 'No posts in progress' },
  { value: 'read', label: 'Read', emptyMessage: 'No completed posts' },
]

export function ReadingList() {
  const [activeTab, setActiveTab] = useState<TabStatus>('unread')
  const [items, setItems] = useState<Record<TabStatus, ReadingListItem[]>>({
    unread: [],
    reading: [],
    read: [],
  })
  const [counts, setCounts] = useState<Record<TabStatus, number>>({
    unread: 0,
    reading: 0,
    read: 0,
  })
  const [loading, setLoading] = useState<Record<TabStatus, boolean>>({
    unread: true,
    reading: true,
    read: true,
  })
  const [error, setError] = useState<string | null>(null)
  const [allTags, setAllTags] = useState<string[]>([])
  const [selectedTag, setSelectedTag] = useState<string | null>(null)

  const fetchTags = useCallback(async () => {
    try {
      const data = await api.get<string[]>('/api/tags')
      setAllTags(data)
    } catch {
      // Non-critical — autocomplete just won't work
    }
  }, [])

  const fetchTab = useCallback(async (status: TabStatus) => {
    setLoading((prev) => ({ ...prev, [status]: true }))

    try {
      const data = await api.get<ReadingListItem[]>(`/api/reading-list?status=${status}`)
      setItems((prev) => ({ ...prev, [status]: data }))
      setCounts((prev) => ({ ...prev, [status]: data.length }))
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load reading list')
    } finally {
      setLoading((prev) => ({ ...prev, [status]: false }))
    }
  }, [])

  const fetchAll = useCallback(async () => {
    await Promise.all([
      ...tabConfig.map((tab) => fetchTab(tab.value)),
      fetchTags(),
    ])
  }, [fetchTab, fetchTags])

  useEffect(() => {
    void fetchAll()
  }, [fetchAll])

  async function handleStatusChange(id: number, newStatus: string) {
    const oldItem = Object.values(items)
      .flat()
      .find((item) => item.id === id)
    if (!oldItem) return

    const oldStatus = oldItem.status

    setItems((prev) => ({
      ...prev,
      [oldStatus]: prev[oldStatus].filter((item) => item.id !== id),
      [newStatus as TabStatus]: [
        { ...oldItem, status: newStatus as TabStatus },
        ...prev[newStatus as TabStatus],
      ],
    }))
    setCounts((prev) => ({
      ...prev,
      [oldStatus]: prev[oldStatus] - 1,
      [newStatus as TabStatus]: prev[newStatus as TabStatus] + 1,
    }))

    // Jump to the target tab so the user sees the moved item.
    setActiveTab(newStatus as TabStatus)

    try {
      await api.patch(`/api/reading-list/${id}`, { status: newStatus })
    } catch {
      setItems((prev) => ({
        ...prev,
        [oldStatus]: [...prev[oldStatus], oldItem],
        [newStatus as TabStatus]: prev[newStatus as TabStatus].filter(
          (item) => item.id !== id
        ),
      }))
      setCounts((prev) => ({
        ...prev,
        [oldStatus]: prev[oldStatus] + 1,
        [newStatus as TabStatus]: prev[newStatus as TabStatus] - 1,
      }))
    }
  }

  async function handleRemove(id: number) {
    const oldItem = Object.values(items)
      .flat()
      .find((item) => item.id === id)
    if (!oldItem) return

    const status = oldItem.status

    setItems((prev) => ({
      ...prev,
      [status]: prev[status].filter((item) => item.id !== id),
    }))
    setCounts((prev) => ({
      ...prev,
      [status]: prev[status] - 1,
    }))

    try {
      await api.del(`/api/reading-list/${id}`)
    } catch {
      setItems((prev) => ({
        ...prev,
        [status]: [...prev[status], oldItem],
      }))
      setCounts((prev) => ({
        ...prev,
        [status]: prev[status] + 1,
      }))
    }
  }

  function handleTagsChange() {
    void fetchAll()
  }

  function getFilteredItems(tabItems: ReadingListItem[]) {
    if (!selectedTag) return tabItems
    return tabItems.filter((item) => item.tags.includes(selectedTag))
  }

  // Collect all unique tags from currently loaded items for the filter bar.
  const usedTags = Array.from(
    new Set(Object.values(items).flat().flatMap((item) => item.tags))
  ).sort()

  return (
    <div className="space-y-8">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Reading List</h1>
        <p className="mt-1 text-muted-foreground">
          Your saved blog posts, organized by reading status.
        </p>
      </div>

      {error && (
        <div className="flex items-start gap-3 rounded-lg border border-destructive/50 bg-destructive/10 p-4 text-sm">
          <AlertCircle className="mt-0.5 size-4 shrink-0 text-destructive" />
          <p>{error}</p>
        </div>
      )}

      {/* Tag filter bar */}
      {usedTags.length > 0 && (
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm font-medium text-muted-foreground">Filter by tag:</span>
          {usedTags.map((tag) => (
            <Badge
              key={tag}
              variant={selectedTag === tag ? 'default' : 'outline'}
              className={`cursor-pointer transition-colors ${
                selectedTag === tag
                  ? 'bg-primary text-primary-foreground'
                  : 'border-primary/30 text-primary hover:bg-primary/10'
              }`}
              onClick={() => setSelectedTag(selectedTag === tag ? null : tag)}
            >
              {tag}
              {selectedTag === tag && <X className="ml-1 size-3" />}
            </Badge>
          ))}
          {selectedTag && (
            <button
              className="text-xs text-muted-foreground hover:text-foreground transition-colors"
              onClick={() => setSelectedTag(null)}
            >
              Clear filter
            </button>
          )}
        </div>
      )}

      <Tabs
        value={activeTab}
        onValueChange={(v) => setActiveTab(v as TabStatus)}
      >
        <TabsList>
          {tabConfig.map((tab) => (
            <TabsTrigger key={tab.value} value={tab.value} className="gap-2">
              {tab.label}
              <Badge variant="secondary" className="ml-1 px-1.5 py-0 text-[10px]">
                {counts[tab.value]}
              </Badge>
            </TabsTrigger>
          ))}
        </TabsList>

        {tabConfig.map((tab) => {
          const filtered = getFilteredItems(items[tab.value])
          return (
            <TabsContent key={tab.value} value={tab.value} className="mt-6">
              {loading[tab.value] ? (
                <div className="space-y-4">
                  {Array.from({ length: 3 }).map((_, i) => (
                    <div key={i} className="space-y-4 rounded-xl border p-6">
                      <div className="flex items-center justify-between">
                        <Skeleton className="h-5 w-24" />
                        <Skeleton className="h-4 w-20" />
                      </div>
                      <Skeleton className="h-6 w-3/4" />
                      <div className="space-y-2">
                        <Skeleton className="h-4 w-full" />
                        <Skeleton className="h-4 w-2/3" />
                      </div>
                      <div className="flex gap-2">
                        <Skeleton className="h-8 w-28" />
                        <Skeleton className="h-8 w-28" />
                      </div>
                    </div>
                  ))}
                </div>
              ) : filtered.length === 0 ? (
                <div className="py-12 text-center">
                  <p className="text-muted-foreground">
                    {selectedTag
                      ? `No ${tab.label.toLowerCase()} posts with tag "${selectedTag}"`
                      : tab.emptyMessage}
                  </p>
                </div>
              ) : (
                <div className="space-y-4">
                  {filtered.map((item) => (
                    <ReadingItem
                      key={item.id}
                      item={item}
                      onStatusChange={handleStatusChange}
                      onRemove={handleRemove}
                      onTagsChange={handleTagsChange}
                      allTags={allTags}
                    />
                  ))}
                </div>
              )}
            </TabsContent>
          )
        })}
      </Tabs>
    </div>
  )
}
