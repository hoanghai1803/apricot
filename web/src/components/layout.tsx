import { useState, useEffect, useCallback, useRef } from 'react'
import { Outlet, NavLink } from 'react-router-dom'
import {
  BookOpen, Home, Settings, Search as SearchIcon, Menu, X,
  Sun, Moon, Monitor, ExternalLink, BookmarkPlus, BookmarkCheck, Loader2, Clock,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { Button } from '@/components/ui/button'
import { Separator } from '@/components/ui/separator'
import { Badge } from '@/components/ui/badge'
import { type Theme, getStoredTheme, setStoredTheme, applyTheme } from '@/lib/theme'
import { api } from '@/lib/api'
import type { Blog } from '@/lib/types'
import { ConfirmDialog } from '@/components/confirm-dialog'

const navItems = [
  { to: '/', label: 'Home', icon: Home },
  { to: '/reading-list', label: 'Reading List', icon: BookOpen },
  { to: '/preferences', label: 'Preferences', icon: Settings },
] as const

const themeOptions: { value: Theme; icon: typeof Sun; label: string }[] = [
  { value: 'light', icon: Sun, label: 'Light' },
  { value: 'dark', icon: Moon, label: 'Dark' },
  { value: 'system', icon: Monitor, label: 'System' },
]

// --- Search history helpers ---

const SEARCH_HISTORY_KEY = 'apricot-search-history'
const MAX_HISTORY = 8

function getSearchHistory(): string[] {
  try {
    const raw = localStorage.getItem(SEARCH_HISTORY_KEY)
    return raw ? JSON.parse(raw) : []
  } catch {
    return []
  }
}

function saveSearchHistory(history: string[]) {
  localStorage.setItem(SEARCH_HISTORY_KEY, JSON.stringify(history.slice(0, MAX_HISTORY)))
}

function addToHistory(query: string) {
  const trimmed = query.trim()
  if (!trimmed) return
  const history = getSearchHistory().filter((h) => h !== trimmed)
  history.unshift(trimmed)
  saveSearchHistory(history)
}

function removeFromHistory(query: string) {
  const history = getSearchHistory().filter((h) => h !== query)
  saveSearchHistory(history)
}

// --- Search result overlay item ---

function SearchResultItem({
  blog,
  onAdd,
  isAdded,
}: {
  blog: Blog
  onAdd: (blogId: number) => void
  isAdded: boolean
}) {
  const [confirmOpen, setConfirmOpen] = useState(false)

  const snippet = blog.description
    ? blog.description.length > 200
      ? blog.description.slice(0, 200) + '...'
      : blog.description
    : null

  return (
    <>
      <div className="flex items-start gap-3 rounded-lg p-4 hover:bg-accent/50 transition-colors">
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-2">
            {blog.source && <Badge variant="secondary" className="text-[10px]">{blog.source}</Badge>}
            {blog.published_at && (
              <span className="text-xs text-muted-foreground">
                {new Date(blog.published_at).toLocaleDateString('en-US', {
                  month: 'short', day: 'numeric', year: 'numeric',
                })}
              </span>
            )}
          </div>
          <a
            href={blog.url}
            target="_blank"
            rel="noopener noreferrer"
            className="mt-1 block text-sm font-medium hover:text-primary hover:underline underline-offset-2 transition-colors"
          >
            {blog.title}
          </a>
          {snippet && (
            <p className="mt-1 text-xs leading-relaxed text-muted-foreground line-clamp-2">
              {snippet}
            </p>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-1 pt-1">
          <Button variant="ghost" size="icon" className="size-8" asChild>
            <a href={blog.url} target="_blank" rel="noopener noreferrer" title="Open original">
              <ExternalLink className="size-3.5" />
            </a>
          </Button>
          <Button
            variant="ghost"
            size="icon"
            className="size-8"
            onClick={() => setConfirmOpen(true)}
            disabled={isAdded}
            title={isAdded ? 'Already in reading list' : 'Add to reading list'}
          >
            {isAdded ? (
              <BookmarkCheck className="size-3.5 text-primary" />
            ) : (
              <BookmarkPlus className="size-3.5" />
            )}
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={confirmOpen}
        onOpenChange={setConfirmOpen}
        title="Save to Reading List"
        description={`Add "${blog.title}" to your reading list?`}
        confirmLabel="Save"
        onConfirm={() => onAdd(blog.id)}
      />
    </>
  )
}

// --- Main Layout ---

export function Layout() {
  const [mobileOpen, setMobileOpen] = useState(false)
  const [theme, setTheme] = useState<Theme>(getStoredTheme)

  // Search input state
  const [searchOpen, setSearchOpen] = useState(false)
  const [searchQuery, setSearchQuery] = useState('')
  const [history, setHistory] = useState<string[]>(getSearchHistory)
  const [historyIndex, setHistoryIndex] = useState(-1)
  const inputRef = useRef<HTMLInputElement>(null)
  const dropdownRef = useRef<HTMLDivElement>(null)

  // Results overlay state (separate from input dropdown)
  const [resultsOpen, setResultsOpen] = useState(false)
  const [resultsQuery, setResultsQuery] = useState('')
  const [searchResults, setSearchResults] = useState<Blog[]>([])
  const [searchLoading, setSearchLoading] = useState(false)
  const [addedIds, setAddedIds] = useState<Set<number>>(new Set())

  const cycleTheme = useCallback(() => {
    const order: Theme[] = ['light', 'dark', 'system']
    const next = order[(order.indexOf(theme) + 1) % order.length]
    setTheme(next)
    setStoredTheme(next)
    applyTheme(next)
  }, [theme])

  useEffect(() => {
    applyTheme(theme)
  }, [theme])

  // Close history dropdown when clicking outside the search area.
  useEffect(() => {
    function handleClickOutside(e: MouseEvent) {
      if (
        dropdownRef.current &&
        !dropdownRef.current.contains(e.target as Node) &&
        inputRef.current &&
        !inputRef.current.contains(e.target as Node)
      ) {
        setSearchOpen(false)
      }
    }
    document.addEventListener('mousedown', handleClickOutside)
    return () => document.removeEventListener('mousedown', handleClickOutside)
  }, [])

  // Keyboard shortcut: Cmd/Ctrl+K to open search.
  useEffect(() => {
    function handleGlobalKeyDown(e: KeyboardEvent) {
      if ((e.metaKey || e.ctrlKey) && e.key === 'k') {
        e.preventDefault()
        setSearchOpen(true)
        setTimeout(() => inputRef.current?.focus(), 0)
      }
    }
    document.addEventListener('keydown', handleGlobalKeyDown)
    return () => document.removeEventListener('keydown', handleGlobalKeyDown)
  }, [])

  // Close results overlay on Escape.
  useEffect(() => {
    function handleGlobalKeyDown(e: KeyboardEvent) {
      if (e.key === 'Escape' && resultsOpen) {
        setResultsOpen(false)
      }
    }
    document.addEventListener('keydown', handleGlobalKeyDown)
    return () => document.removeEventListener('keydown', handleGlobalKeyDown)
  }, [resultsOpen])

  async function executeSearch(q: string) {
    const trimmed = q.trim()
    if (!trimmed) return

    // Save to history, close dropdown, open results overlay.
    addToHistory(trimmed)
    setHistory(getSearchHistory())
    setSearchQuery(trimmed)
    setSearchOpen(false)
    setResultsQuery(trimmed)
    setResultsOpen(true)
    setSearchLoading(true)

    try {
      const data = await api.get<Blog[]>(`/api/search?q=${encodeURIComponent(trimmed)}&limit=20`)
      setSearchResults(data)
    } catch {
      setSearchResults([])
    } finally {
      setSearchLoading(false)
    }
  }

  function handleDeleteHistory(entry: string, e: React.MouseEvent) {
    e.preventDefault()
    e.stopPropagation()
    removeFromHistory(entry)
    setHistory(getSearchHistory())
    setHistoryIndex(-1)
  }

  function handleInputKeyDown(e: React.KeyboardEvent) {
    const showingHistory = searchOpen && history.length > 0

    if (showingHistory) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setHistoryIndex((prev) => Math.min(prev + 1, history.length - 1))
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setHistoryIndex((prev) => Math.max(prev - 1, -1))
        return
      }
      if (e.key === 'Enter' && historyIndex >= 0) {
        e.preventDefault()
        void executeSearch(history[historyIndex])
        return
      }
    }

    if (e.key === 'Enter' && searchQuery.trim()) {
      e.preventDefault()
      void executeSearch(searchQuery)
      return
    }

    if (e.key === 'Escape') {
      setSearchOpen(false)
      inputRef.current?.blur()
    }
  }

  async function handleAddToReadingList(blogId: number) {
    try {
      await api.post('/api/reading-list', { blog_id: blogId })
      setAddedIds((prev) => new Set(prev).add(blogId))
    } catch {
      setAddedIds((prev) => new Set(prev).add(blogId))
    }
  }

  const currentThemeOption = themeOptions.find((t) => t.value === theme)!
  const ThemeIcon = currentThemeOption.icon

  // Show history dropdown: input is focused, has history entries.
  // When user starts typing, the text replaces what's in the field but history still shows
  // (filtered by typed text). On Enter, it searches; on select it searches.
  const filteredHistory = searchQuery.trim()
    ? history.filter((h) => h.toLowerCase().includes(searchQuery.toLowerCase()))
    : history
  const showHistoryDropdown = searchOpen && filteredHistory.length > 0

  return (
    <div className="min-h-screen bg-background text-foreground">
      <header className="sticky top-0 z-50 border-b border-border/50 bg-background/80 backdrop-blur-xl">
        <div className="mx-auto flex h-14 max-w-6xl items-center px-4 sm:px-6">
          <NavLink to="/" className="mr-8 flex items-center gap-2">
            <span className="text-lg font-bold tracking-tight text-primary">Apricot</span>
          </NavLink>

          <nav className="hidden items-center gap-1 md:flex">
            {navItems.map(({ to, label, icon: Icon }) => (
              <NavLink
                key={to}
                to={to}
                end={to === '/'}
                className={({ isActive }) =>
                  cn(
                    'inline-flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                    isActive
                      ? 'bg-primary/10 text-primary'
                      : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                  )
                }
              >
                <Icon className="size-4" />
                {label}
              </NavLink>
            ))}
          </nav>

          <div className="ml-auto flex items-center gap-1">
            {/* Search input */}
            <div className="relative">
              <div className="relative flex items-center">
                <SearchIcon className="absolute left-2.5 size-3.5 text-muted-foreground pointer-events-none" />
                <input
                  ref={inputRef}
                  type="text"
                  value={historyIndex >= 0 ? filteredHistory[historyIndex] ?? searchQuery : searchQuery}
                  onChange={(e) => {
                    setSearchQuery(e.target.value)
                    setHistoryIndex(-1)
                  }}
                  onFocus={() => setSearchOpen(true)}
                  onKeyDown={handleInputKeyDown}
                  placeholder="Search..."
                  className="h-8 w-40 rounded-md border border-input bg-background pl-8 pr-2 text-sm focus:outline-none focus:ring-1 focus:ring-primary sm:w-56"
                />
              </div>

              {/* History dropdown */}
              {showHistoryDropdown && (
                <div
                  ref={dropdownRef}
                  className="absolute right-0 top-full mt-1.5 w-80 rounded-lg border bg-popover py-1.5 shadow-lg sm:w-96"
                >
                  {filteredHistory.map((entry, i) => (
                    <button
                      key={entry}
                      className={cn(
                        'flex w-full items-center gap-3 px-3 py-2.5 text-left text-sm transition-colors',
                        i === historyIndex ? 'bg-accent' : 'hover:bg-accent/50'
                      )}
                      onMouseDown={(e) => {
                        e.preventDefault()
                        void executeSearch(entry)
                      }}
                    >
                      <span className="flex size-8 shrink-0 items-center justify-center rounded-full bg-muted">
                        <Clock className="size-4 text-muted-foreground" />
                      </span>
                      <span className="flex-1 truncate">{entry}</span>
                      <button
                        className="shrink-0 rounded-full p-1 text-muted-foreground/50 hover:text-foreground hover:bg-muted transition-colors"
                        onMouseDown={(e) => handleDeleteHistory(entry, e)}
                        aria-label={`Remove "${entry}" from search history`}
                      >
                        <X className="size-4" />
                      </button>
                    </button>
                  ))}
                </div>
              )}
            </div>

            <Button
              variant="ghost"
              size="icon"
              onClick={cycleTheme}
              aria-label={`Theme: ${currentThemeOption.label}. Click to change.`}
              title={`Theme: ${currentThemeOption.label}`}
            >
              <ThemeIcon className="size-4" />
            </Button>

            <div className="md:hidden">
              <Button
                variant="ghost"
                size="icon"
                onClick={() => setMobileOpen(!mobileOpen)}
                aria-label="Toggle navigation"
              >
                {mobileOpen ? <X className="size-5" /> : <Menu className="size-5" />}
              </Button>
            </div>
          </div>
        </div>

        {mobileOpen && (
          <div className="border-t md:hidden">
            <nav className="mx-auto flex max-w-6xl flex-col gap-1 px-4 py-3 sm:px-6">
              {navItems.map(({ to, label, icon: Icon }) => (
                <NavLink
                  key={to}
                  to={to}
                  end={to === '/'}
                  onClick={() => setMobileOpen(false)}
                  className={({ isActive }) =>
                    cn(
                      'inline-flex items-center gap-2 rounded-md px-3 py-2 text-sm font-medium transition-colors',
                      isActive
                        ? 'bg-primary/10 text-primary'
                        : 'text-muted-foreground hover:bg-accent hover:text-foreground'
                    )
                  }
                >
                  <Icon className="size-4" />
                  {label}
                </NavLink>
              ))}
            </nav>
          </div>
        )}
      </header>

      <Separator />

      <main className="mx-auto max-w-6xl px-4 py-8 sm:px-6">
        <Outlet />
      </main>

      {/* Search results overlay */}
      {resultsOpen && (
        <div
          className="fixed inset-0 z-[60] flex items-start justify-center bg-black/50 backdrop-blur-sm pt-[10vh]"
          onClick={(e) => {
            if (e.target === e.currentTarget) setResultsOpen(false)
          }}
        >
          <div className="w-full max-w-2xl rounded-xl border bg-popover shadow-2xl animate-in fade-in slide-in-from-top-4 duration-200">
            {/* Overlay header with search input */}
            <div className="flex items-center gap-3 border-b px-4 py-3">
              <SearchIcon className="size-4 shrink-0 text-muted-foreground" />
              <input
                type="text"
                value={resultsQuery}
                onChange={(e) => setResultsQuery(e.target.value)}
                onKeyDown={(e) => {
                  if (e.key === 'Enter' && resultsQuery.trim()) {
                    void executeSearch(resultsQuery)
                  }
                  if (e.key === 'Escape') {
                    setResultsOpen(false)
                  }
                }}
                className="flex-1 bg-transparent text-sm focus:outline-none"
                autoFocus
              />
              <button
                onClick={() => setResultsOpen(false)}
                className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
              >
                <X className="size-4" />
              </button>
            </div>

            {/* Results body */}
            <div className="max-h-[60vh] overflow-y-auto">
              {searchLoading ? (
                <div className="flex items-center justify-center gap-2 py-12">
                  <Loader2 className="size-4 animate-spin text-muted-foreground" />
                  <span className="text-sm text-muted-foreground">Searching...</span>
                </div>
              ) : searchResults.length === 0 ? (
                <div className="py-12 text-center">
                  <SearchIcon className="mx-auto size-8 text-muted-foreground/30" />
                  <p className="mt-3 text-sm text-muted-foreground">
                    No results found for &ldquo;{resultsQuery}&rdquo;
                  </p>
                </div>
              ) : (
                <div className="divide-y divide-border/50">
                  <div className="px-4 py-2 text-xs text-muted-foreground">
                    {searchResults.length} result{searchResults.length !== 1 ? 's' : ''} for &ldquo;{resultsQuery}&rdquo;
                  </div>
                  {searchResults.map((blog) => (
                    <SearchResultItem
                      key={blog.id}
                      blog={blog}
                      onAdd={handleAddToReadingList}
                      isAdded={addedIds.has(blog.id)}
                    />
                  ))}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  )
}
