import { ExternalLink, BookOpen, CheckCircle, RotateCcw, Trash2 } from 'lucide-react'
import type { ReadingListItem } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardContent, CardFooter } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'

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

export function ReadingItem({ item, onStatusChange, onRemove }: ReadingItemProps) {
  const title = item.blog?.title ?? `Blog #${item.blog_id}`
  const url = item.blog?.url
  const source = item.blog?.source

  return (
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
              className="hover:underline"
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
        {item.status === 'unread' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onStatusChange(item.id, 'reading')}
          >
            <BookOpen className="size-4" />
            Start Reading
          </Button>
        )}
        {item.status === 'reading' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onStatusChange(item.id, 'read')}
          >
            <CheckCircle className="size-4" />
            Mark as Read
          </Button>
        )}
        {item.status === 'read' && (
          <Button
            variant="outline"
            size="sm"
            onClick={() => onStatusChange(item.id, 'unread')}
          >
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
          onClick={() => onRemove(item.id)}
        >
          <Trash2 className="size-4" />
          Remove
        </Button>
      </CardFooter>
    </Card>
  )
}
