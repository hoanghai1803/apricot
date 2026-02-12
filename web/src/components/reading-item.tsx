import { useState } from 'react'
import { ExternalLink, BookOpen, CheckCircle, RotateCcw, Trash2 } from 'lucide-react'
import type { ReadingListItem } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/confirm-dialog'

interface ReadingItemProps {
  item: ReadingListItem
  onStatusChange: (id: number, status: string) => void
  onRemove: (id: number) => void
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

export function ReadingItem({ item, onStatusChange, onRemove }: ReadingItemProps) {
  const [statusConfirm, setStatusConfirm] = useState<string | null>(null)
  const [removeConfirm, setRemoveConfirm] = useState(false)

  const title = item.blog?.title ?? `Blog #${item.blog_id}`
  const url = item.blog?.url
  const source = item.blog?.source

  const nextStatus =
    item.status === 'unread' ? 'reading' :
    item.status === 'reading' ? 'read' : 'unread'

  const statusInfo = statusLabels[nextStatus]

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
          <CardContent>
            <p className="text-sm leading-relaxed text-foreground/90">
              {item.summary}
            </p>
          </CardContent>
        )}

        <CardFooter className="flex-wrap gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => setStatusConfirm(nextStatus)}
          >
            {nextStatus === 'reading' && <BookOpen className="size-4" />}
            {nextStatus === 'read' && <CheckCircle className="size-4" />}
            {nextStatus === 'unread' && <RotateCcw className="size-4" />}
            {statusInfo.action}
          </Button>

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

      {statusConfirm && (
        <ConfirmDialog
          open
          onOpenChange={() => setStatusConfirm(null)}
          title={statusInfo.action}
          description={statusInfo.description}
          confirmLabel={statusInfo.action}
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
