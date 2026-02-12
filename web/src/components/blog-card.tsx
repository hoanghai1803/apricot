import { useState } from 'react'
import { ExternalLink, BookmarkPlus, BookmarkCheck } from 'lucide-react'
import type { DiscoverResult } from '@/lib/types'
import { Card, CardHeader, CardTitle, CardDescription, CardContent, CardFooter } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import { ConfirmDialog } from '@/components/confirm-dialog'

interface BlogCardProps {
  blog: DiscoverResult
  onAddToReadingList: (blogId: number) => void
  isAdded?: boolean
}

function formatDate(dateStr?: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  return date.toLocaleDateString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
  })
}

export function BlogCard({ blog, onAddToReadingList, isAdded = false }: BlogCardProps) {
  const [confirmOpen, setConfirmOpen] = useState(false)

  return (
    <>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-2">
            <Badge variant="secondary">{blog.source}</Badge>
            {blog.published_at && (
              <span className="text-xs text-muted-foreground">
                {formatDate(blog.published_at)}
              </span>
            )}
          </div>
          <CardTitle>
            <a
              href={blog.url}
              target="_blank"
              rel="noopener noreferrer"
              className="hover:text-primary hover:underline underline-offset-2 transition-colors"
            >
              {blog.title}
            </a>
          </CardTitle>
          <CardDescription className="sr-only">
            From {blog.source}
          </CardDescription>
        </CardHeader>

        <CardContent className="space-y-3">
          <p className="text-sm leading-relaxed text-foreground/90">
            {blog.summary}
          </p>
          <p className="text-sm italic text-muted-foreground">
            Matched because: {blog.reason}
          </p>
        </CardContent>

        <CardFooter className="gap-2">
          <Button variant="outline" size="sm" asChild>
            <a href={blog.url} target="_blank" rel="noopener noreferrer">
              <ExternalLink className="size-4" />
              Read Original
            </a>
          </Button>
          <Button
            variant={isAdded ? 'secondary' : 'default'}
            size="sm"
            onClick={() => setConfirmOpen(true)}
            disabled={isAdded}
          >
            {isAdded ? (
              <>
                <BookmarkCheck className="size-4" />
                Added
              </>
            ) : (
              <>
                <BookmarkPlus className="size-4" />
                Add to Reading List
              </>
            )}
          </Button>
        </CardFooter>
      </Card>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Save to Reading List"
        description={`Add "${blog.title}" to your reading list?`}
        confirmLabel="Save"
        onConfirm={() => onAddToReadingList(blog.id)}
      />
    </>
  )
}
