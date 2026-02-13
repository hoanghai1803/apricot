import { useState, useRef } from 'react'
import { ExternalLink, BookOpen, CheckCircle, RotateCcw, Trash2, Plus, X, Tag } from 'lucide-react'
import type { ReadingListItem } from '@/lib/types'
import { cn } from '@/lib/utils'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/confirm-dialog'
import { api } from '@/lib/api'

interface ReadingItemProps {
  item: ReadingListItem
  onStatusChange: (id: number, status: string) => void
  onRemove: (id: number) => void
  onTagsChange: () => void
  allTags: string[]
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

const statusLabels: Record<string, { action: string; description: string }> = {
  reading: {
    action: 'Start Reading',
    description: 'Move this post to your "Reading" list?',
  },
  read: {
    action: 'Mark as Read',
    description: 'Mark this post as finished reading?',
  },
  unread: {
    action: 'Back to Unread',
    description: 'Move this post back to your "Unread" list?',
  },
}

export function ReadingItem({ item, onStatusChange, onRemove, onTagsChange, allTags }: ReadingItemProps) {
  const [statusConfirm, setStatusConfirm] = useState<string | null>(null)
  const [removeConfirm, setRemoveConfirm] = useState(false)
  const [showTagInput, setShowTagInput] = useState(false)
  const [tagInput, setTagInput] = useState('')
  const [suggestionIndex, setSuggestionIndex] = useState(-1)
  const inputRef = useRef<HTMLInputElement>(null)

  const title = item.blog?.title ?? `Blog #${item.blog_id}`
  const url = item.blog?.url
  const source = item.blog?.source

  const confirmInfo = statusConfirm ? statusLabels[statusConfirm] : null

  // Compute suggestions: filter existing tags, excluding already-applied ones.
  const existingSet = new Set(item.tags)
  const suggestions = (() => {
    const lower = tagInput.trim().toLowerCase()
    const available = allTags.filter((t) => !existingSet.has(t))
    if (!lower) return available.slice(0, 8)
    return available.filter((t) => t.includes(lower)).slice(0, 8)
  })()

  async function handleAddTag(tagName: string) {
    const trimmed = tagName.trim().toLowerCase()
    if (!trimmed) return

    setTagInput('')
    setSuggestionIndex(-1)
    setShowTagInput(false)

    try {
      await api.post(`/api/reading-list/${item.id}/tags`, { tag: trimmed })
      onTagsChange()
    } catch {
      // Silently fail
    }
  }

  async function handleRemoveTag(tagName: string) {
    try {
      await api.del(`/api/reading-list/${item.id}/tags/${encodeURIComponent(tagName)}`)
      onTagsChange()
    } catch {
      // Silently fail
    }
  }

  function handleTagKeyDown(e: React.KeyboardEvent) {
    if (e.key === 'ArrowDown') {
      e.preventDefault()
      setSuggestionIndex((prev) => Math.min(prev + 1, suggestions.length - 1))
      return
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault()
      setSuggestionIndex((prev) => Math.max(prev - 1, -1))
      return
    }
    if (e.key === 'Enter') {
      e.preventDefault()
      if (suggestionIndex >= 0 && suggestions[suggestionIndex]) {
        void handleAddTag(suggestions[suggestionIndex])
      } else if (tagInput.trim()) {
        void handleAddTag(tagInput)
      }
      return
    }
    if (e.key === 'Escape') {
      setShowTagInput(false)
      setTagInput('')
      setSuggestionIndex(-1)
    }
  }

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-2">
            <div className="flex items-center gap-2">
              {source && <Badge variant="secondary">{source}</Badge>}
            </div>
            <span className="text-xs text-muted-foreground">
              Added {formatDate(item.added_at)}
            </span>
          </div>
          <CardTitle>
            {url ? (
              <a
                href={url}
                target="_blank"
                rel="noopener noreferrer"
                className="hover:text-primary hover:underline underline-offset-2 transition-colors"
              >
                {title}
              </a>
            ) : (
              title
            )}
          </CardTitle>
        </CardHeader>

        {item.summary && (
          <CardContent className="space-y-3">
            <p className="text-sm leading-relaxed text-foreground/90">
              {item.summary}
            </p>
          </CardContent>
        )}

        {/* Tags section */}
        <CardContent className="pt-0">
          <div className="flex flex-wrap items-center gap-2">
            {item.tags.map((tag) => (
              <Badge
                key={tag}
                variant="outline"
                className="gap-1 border-primary/40 text-primary bg-primary/5 py-1 px-2.5"
              >
                {tag}
                <button
                  onClick={() => handleRemoveTag(tag)}
                  className="ml-0.5 rounded-full hover:bg-primary/20 p-0.5 transition-colors"
                  aria-label={`Remove tag ${tag}`}
                >
                  <X className="size-3" />
                </button>
              </Badge>
            ))}

            {showTagInput ? (
              <div className="relative">
                <input
                  ref={inputRef}
                  type="text"
                  value={tagInput}
                  onChange={(e) => {
                    setTagInput(e.target.value)
                    setSuggestionIndex(-1)
                  }}
                  onKeyDown={handleTagKeyDown}
                  onBlur={() => {
                    setTimeout(() => {
                      setShowTagInput(false)
                      setTagInput('')
                      setSuggestionIndex(-1)
                    }, 200)
                  }}
                  placeholder="Type a tag..."
                  className="h-7 w-36 rounded-md border border-primary/40 bg-background px-2.5 text-xs focus:outline-none focus:ring-1 focus:ring-primary"
                  autoFocus
                />
                {suggestions.length > 0 && (
                  <div className="absolute top-full left-0 z-10 mt-1 w-48 rounded-lg border bg-popover py-1 shadow-lg">
                    {suggestions.map((s, i) => (
                      <button
                        key={s}
                        className={cn(
                          'flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs transition-colors',
                          i === suggestionIndex ? 'bg-accent text-accent-foreground' : 'hover:bg-accent/50'
                        )}
                        onMouseDown={(e) => {
                          e.preventDefault()
                          void handleAddTag(s)
                        }}
                      >
                        <Tag className="size-3 shrink-0 text-muted-foreground" />
                        {s}
                      </button>
                    ))}
                  </div>
                )}
              </div>
            ) : (
              <button
                className="inline-flex items-center gap-1.5 rounded-md border border-dashed border-primary/40 px-2.5 py-1 text-xs font-medium text-primary hover:bg-primary/10 hover:border-primary/60 transition-colors"
                onClick={() => setShowTagInput(true)}
              >
                <Plus className="size-3" />
                Add Tag
              </button>
            )}
          </div>
        </CardContent>

        <CardFooter className="flex-wrap gap-2">
          {item.status === 'unread' && (
            <Button variant="outline" size="sm" onClick={() => setStatusConfirm('reading')}>
              <BookOpen className="size-4" />
              Start Reading
            </Button>
          )}
          {item.status === 'reading' && (
            <>
              <Button variant="outline" size="sm" onClick={() => setStatusConfirm('read')}>
                <CheckCircle className="size-4" />
                Mark as Read
              </Button>
              <Button variant="outline" size="sm" onClick={() => setStatusConfirm('unread')}>
                <RotateCcw className="size-4" />
                Back to Unread
              </Button>
            </>
          )}
          {item.status === 'read' && (
            <Button variant="outline" size="sm" onClick={() => setStatusConfirm('unread')}>
              <RotateCcw className="size-4" />
              Back to Unread
            </Button>
          )}

          {url && (
            <Button variant="outline" size="sm" asChild>
              <a href={url} target="_blank" rel="noopener noreferrer">
                <ExternalLink className="size-4" />
                Read Original
              </a>
            </Button>
          )}

          <Button
            variant="destructive"
            size="sm"
            onClick={() => setRemoveConfirm(true)}
          >
            <Trash2 className="size-4" />
            Remove
          </Button>
        </CardFooter>
      </Card>

      {statusConfirm && confirmInfo && (
        <ConfirmDialog
          open
          onOpenChange={() => setStatusConfirm(null)}
          title={confirmInfo.action}
          description={confirmInfo.description}
          confirmLabel={confirmInfo.action}
          onConfirm={() => onStatusChange(item.id, statusConfirm)}
        />
      )}

      <ConfirmDialog
        open={removeConfirm}
        onOpenChange={setRemoveConfirm}
        title="Remove from Reading List"
        description={`Are you sure you want to remove "${title}" from your reading list? This action cannot be undone.`}
        confirmLabel="Remove"
        variant="destructive"
        onConfirm={() => onRemove(item.id)}
      />
    </>
  )
}
