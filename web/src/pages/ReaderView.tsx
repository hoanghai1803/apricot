import { useState, useEffect } from 'react'
import { useParams, useNavigate } from 'react-router-dom'
import { ArrowLeft, ExternalLink, Clock, Loader2, CheckCircle } from 'lucide-react'
import type { ReadingListItem } from '@/lib/types'
import { api } from '@/lib/api'
import { formatReadingTime } from '@/lib/reading'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Toast } from '@/components/toast'

export function ReaderView() {
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()

  const [item, setItem] = useState<ReadingListItem | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [iframeLoading, setIframeLoading] = useState(true)
  const [status, setStatus] = useState<string>('unread')
  const [toastVisible, setToastVisible] = useState(false)
  const [toastMessage, setToastMessage] = useState('')

  function showToast(message: string) {
    setToastMessage(message)
    setToastVisible(true)
  }

  // Fetch the reading list item.
  useEffect(() => {
    if (!id) return

    async function load() {
      setLoading(true)
      setError(null)
      try {
        const data = await api.get<ReadingListItem>(`/api/reading-list/${id}`)
        setItem(data)
        setStatus(data.status)

        // Auto-set status to "reading" if currently "unread".
        if (data.status === 'unread') {
          try {
            await api.patch(`/api/reading-list/${id}`, { status: 'reading' })
            setStatus('reading')
          } catch {
            // Non-critical
          }
        }
      } catch {
        setError('Failed to load article')
      } finally {
        setLoading(false)
      }
    }

    void load()
  }, [id])

  async function handleMarkAsRead() {
    if (!id) return
    try {
      await api.patch(`/api/reading-list/${id}`, { status: 'read' })
      // Also set progress to 100.
      await api.patch(`/api/reading-list/${id}/progress`, { progress: 100 }).catch(() => {})
      setStatus('read')
      showToast('Article marked as read')
    } catch {
      // Ignore
    }
  }

  if (loading) {
    return (
      <div className="flex min-h-screen items-center justify-center bg-background">
        <Loader2 className="size-6 animate-spin text-muted-foreground" />
      </div>
    )
  }

  if (error || !item) {
    return (
      <div className="flex min-h-screen flex-col items-center justify-center gap-4 bg-background px-4">
        <p className="text-muted-foreground">{error ?? 'Article not found'}</p>
        <Button variant="outline" onClick={() => navigate('/reading-list')}>
          <ArrowLeft className="size-4" />
          Back to Reading List
        </Button>
      </div>
    )
  }

  const blog = item.blog
  const blogUrl = blog?.url
  const source = blog?.source
  const readingTime = formatReadingTime(blog?.reading_time_minutes)

  return (
    <div className="flex h-screen flex-col bg-background text-foreground">
      {/* Header bar */}
      <header className="shrink-0 border-b border-border/50 bg-background/80 backdrop-blur-xl">
        <div className="flex items-center gap-3 px-4 py-2">
          <Button
            variant="ghost"
            size="icon"
            onClick={() => navigate('/reading-list')}
            aria-label="Back to reading list"
          >
            <ArrowLeft className="size-5" />
          </Button>

          <div className="flex min-w-0 flex-1 items-center gap-2">
            {source && <Badge variant="secondary" className="shrink-0">{source}</Badge>}
            {readingTime && (
              <span className="flex shrink-0 items-center gap-1 text-xs text-muted-foreground">
                <Clock className="size-3" />
                {readingTime}
              </span>
            )}
            <span className="min-w-0 truncate text-sm font-medium">
              {blog?.title}
            </span>
          </div>

          <div className="flex shrink-0 items-center gap-1">
            {status !== 'read' && (
              <Button variant="outline" size="sm" onClick={handleMarkAsRead}>
                <CheckCircle className="size-4" />
                <span className="hidden sm:inline">Mark as Read</span>
              </Button>
            )}
            {status === 'read' && (
              <Badge variant="default" className="bg-green-600">Read</Badge>
            )}
            {blogUrl && (
              <Button variant="ghost" size="sm" asChild>
                <a href={blogUrl} target="_blank" rel="noopener noreferrer">
                  <ExternalLink className="size-4" />
                  <span className="hidden sm:inline">Open</span>
                </a>
              </Button>
            )}
          </div>
        </div>
      </header>

      {/* Embedded original page via proxy */}
      {blogUrl ? (
        <div className="relative flex-1">
          {iframeLoading && (
            <div className="absolute inset-0 z-10 flex items-center justify-center bg-background">
              <Loader2 className="size-6 animate-spin text-muted-foreground" />
            </div>
          )}
          <iframe
            src={`/api/proxy?url=${encodeURIComponent(blogUrl)}`}
            title={blog?.title ?? 'Article'}
            className="size-full border-0"
            onLoad={() => setIframeLoading(false)}
            sandbox="allow-scripts allow-same-origin allow-popups allow-popups-to-escape-sandbox"
          />
        </div>
      ) : (
        <div className="flex flex-1 flex-col items-center justify-center gap-4 px-4">
          <p className="text-muted-foreground">No URL available for this article.</p>
          <Button variant="outline" onClick={() => navigate('/reading-list')}>
            <ArrowLeft className="size-4" />
            Back to Reading List
          </Button>
        </div>
      )}

      <Toast
        message={toastMessage}
        visible={toastVisible}
        onClose={() => setToastVisible(false)}
      />
    </div>
  )
}
