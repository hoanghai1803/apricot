# React / TypeScript Style Guide

Stack: React 18+, TypeScript (strict mode), Vite, shadcn/ui (Radix UI + Tailwind CSS), React Router.

## File Naming and Organization

```
web/src/
  components/
    ui/                    # shadcn/ui primitives (button, dialog, card, etc.)
    blog-card.tsx          # kebab-case file names
    discovery-feed.tsx
    reading-list-item.tsx
  hooks/
    use-discover.ts        # custom hooks: use-kebab-case.ts
    use-preferences.ts
    use-reading-list.ts
  lib/
    api-client.ts          # centralized fetch wrapper
    utils.ts               # cn() helper from shadcn, small pure functions
  pages/
    discover.tsx           # route-level components
    reading-list.tsx
    settings.tsx
  types/
    models.ts              # shared domain types matching Go models
    api.ts                 # API request/response types
  App.tsx
  main.tsx
```

Rules:
- File names: `kebab-case.tsx` for components, `kebab-case.ts` for non-component modules
- Component names: `PascalCase` inside the file (`BlogCard`, `DiscoveryFeed`)
- One component per file (small sub-components used only in that file are acceptable)
- Co-locate tests: `blog-card.test.tsx` next to `blog-card.tsx`

## TypeScript Conventions

### Strict mode is mandatory

`tsconfig.json` must have `"strict": true`. No `@ts-ignore` or `@ts-expect-error` without a comment explaining why.

### Types vs Interfaces

Use `interface` for object shapes that may be extended (props, API responses). Use `type` for unions, intersections, and computed types.

```tsx
// Interface for props -- can be extended
interface BlogCardProps {
  blog: BlogPost;
  onSave: (id: string) => void;
  variant?: "compact" | "full";
}

// Type for unions and computed types
type Status = "idle" | "loading" | "error" | "success";
type BlogPost = Blog & { summary: string };

// Type for function signatures
type FetchFn = (url: string) => Promise<Response>;
```

### Prop typing

Name prop interfaces `[ComponentName]Props`. Keep props flat -- avoid deeply nested option objects.

```tsx
// Good: explicit, flat props
interface SourceListProps {
  sources: BlogSource[];
  selectedIds: Set<string>;
  onToggle: (id: string) => void;
  onSelectAll: () => void;
}

// Bad: mystery bag
interface SourceListProps {
  options: {
    data: { sources: BlogSource[]; selected: string[] };
    callbacks: { toggle: Function; selectAll: Function };
  };
}
```

### Avoid these

```tsx
// Bad: any
function processData(data: any) { ... }

// Bad: non-null assertion without reason
const name = user!.name;

// Good: narrow the type
function processData(data: unknown) {
  if (!isValidPayload(data)) throw new Error("invalid payload");
  // data is now narrowed
}

// Good: handle the null case
const name = user?.name ?? "Anonymous";
```

## Component Patterns

### Function components only. No class components.

```tsx
// Standard component with props
export function BlogCard({ blog, onSave, variant = "full" }: BlogCardProps) {
  return (
    <Card className={variant === "compact" ? "p-3" : "p-6"}>
      <CardHeader>
        <CardTitle>{blog.title}</CardTitle>
      </CardHeader>
      <CardContent>
        <p>{blog.summary}</p>
      </CardContent>
      <CardFooter>
        <Button onClick={() => onSave(blog.id)}>Save</Button>
      </CardFooter>
    </Card>
  );
}
```

Rules:
- Use named exports, not default exports (better refactoring, better tree-shaking)
- Destructure props in the function signature
- Provide defaults for optional props in the destructure
- Return `null` for conditional non-render, not an empty fragment

### Children pattern

Use `React.ReactNode` for children, not `React.FC`.

```tsx
interface PageLayoutProps {
  title: string;
  children: React.ReactNode;
}

export function PageLayout({ title, children }: PageLayoutProps) {
  return (
    <div className="container mx-auto px-4 py-8">
      <h1 className="text-2xl font-bold mb-6">{title}</h1>
      {children}
    </div>
  );
}
```

## Event Handlers

Name handler props `onXxx`. Name handler implementations `handleXxx`.

```tsx
interface SearchBarProps {
  onSearch: (query: string) => void;
  onClear: () => void;
}

export function SearchBar({ onSearch, onClear }: SearchBarProps) {
  const [query, setQuery] = useState("");

  function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    onSearch(query);
  }

  function handleKeyDown(e: React.KeyboardEvent) {
    if (e.key === "Escape") {
      setQuery("");
      onClear();
    }
  }

  return (
    <form onSubmit={handleSubmit}>
      <Input
        value={query}
        onChange={(e) => setQuery(e.target.value)}
        onKeyDown={handleKeyDown}
        placeholder="Search posts..."
      />
    </form>
  );
}
```

Inline arrow functions are fine for simple one-liners like `onChange`. Extract to a named function when the handler has logic or is used in multiple places.

## State Management

### useState for local UI state

```tsx
const [isOpen, setIsOpen] = useState(false);
const [query, setQuery] = useState("");
```

### useReducer for complex state with multiple transitions

```tsx
type DiscoverState = {
  status: "idle" | "loading" | "error" | "success";
  posts: BlogPost[];
  error: string | null;
};

type DiscoverAction =
  | { type: "start" }
  | { type: "success"; posts: BlogPost[] }
  | { type: "error"; message: string }
  | { type: "reset" };

function discoverReducer(state: DiscoverState, action: DiscoverAction): DiscoverState {
  switch (action.type) {
    case "start":
      return { ...state, status: "loading", error: null };
    case "success":
      return { status: "success", posts: action.posts, error: null };
    case "error":
      return { status: "error", posts: [], error: action.message };
    case "reset":
      return { status: "idle", posts: [], error: null };
  }
}
```

### Context for truly global state (preferences, theme)

Use context sparingly. Not every shared state needs context. Props are fine for 2-3 levels.

```tsx
interface PreferencesContextType {
  preferences: Preferences;
  updatePreferences: (prefs: Partial<Preferences>) => Promise<void>;
}

const PreferencesContext = createContext<PreferencesContextType | null>(null);

export function usePreferences(): PreferencesContextType {
  const ctx = useContext(PreferencesContext);
  if (!ctx) {
    throw new Error("usePreferences must be used within PreferencesProvider");
  }
  return ctx;
}
```

## Data Fetching

Use custom hooks that encapsulate fetch logic, loading state, and error handling. Centralize all API calls through a typed client.

### API client

```tsx
// lib/api-client.ts
const BASE_URL = "/api";

class ApiError extends Error {
  constructor(
    public status: number,
    message: string,
  ) {
    super(message);
    this.name = "ApiError";
  }
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await fetch(`${BASE_URL}${path}`, {
    headers: { "Content-Type": "application/json", ...options?.headers },
    ...options,
  });

  if (!res.ok) {
    const body = await res.text().catch(() => "unknown error");
    throw new ApiError(res.status, body);
  }

  return res.json() as Promise<T>;
}

export const api = {
  getPreferences: () => request<Preferences>("/preferences"),
  updatePreferences: (prefs: Preferences) =>
    request<Preferences>("/preferences", {
      method: "PUT",
      body: JSON.stringify(prefs),
    }),
  discover: () => request<DiscoverResponse>("/discover", { method: "POST" }),
  getReadingList: () => request<ReadingListItem[]>("/reading-list"),
  addToReadingList: (item: AddReadingListRequest) =>
    request<ReadingListItem>("/reading-list", {
      method: "POST",
      body: JSON.stringify(item),
    }),
  deleteFromReadingList: (id: string) =>
    request<void>(`/reading-list/${id}`, { method: "DELETE" }),
  getSources: () => request<BlogSource[]>("/sources"),
};
```

### Custom data-fetching hook

```tsx
// hooks/use-discover.ts
interface UseDiscoverReturn {
  posts: BlogPost[];
  status: "idle" | "loading" | "error" | "success";
  error: string | null;
  discover: () => Promise<void>;
  reset: () => void;
}

export function useDiscover(): UseDiscoverReturn {
  const [state, dispatch] = useReducer(discoverReducer, {
    status: "idle",
    posts: [],
    error: null,
  });

  async function discover() {
    dispatch({ type: "start" });
    try {
      const data = await api.discover();
      dispatch({ type: "success", posts: data.posts });
    } catch (err) {
      const message = err instanceof ApiError ? err.message : "Something went wrong";
      dispatch({ type: "error", message });
    }
  }

  function reset() {
    dispatch({ type: "reset" });
  }

  return { ...state, discover, reset };
}
```

### Loading and error states

Always handle all states. Never assume data is available.

```tsx
export function DiscoverPage() {
  const { posts, status, error, discover, reset } = useDiscover();

  if (status === "error") {
    return (
      <Alert variant="destructive">
        <AlertDescription>{error}</AlertDescription>
        <Button variant="outline" onClick={reset}>Try again</Button>
      </Alert>
    );
  }

  return (
    <PageLayout title="Discover">
      <Button onClick={discover} disabled={status === "loading"}>
        {status === "loading" ? "Collecting..." : "Collect Fancy Blogs"}
      </Button>

      {status === "loading" && <DiscoverSkeleton />}

      {status === "success" && posts.length === 0 && (
        <p className="text-muted-foreground">No posts found. Try adjusting your preferences.</p>
      )}

      {posts.length > 0 && (
        <div className="grid gap-4 md:grid-cols-2">
          {posts.map((post) => (
            <BlogCard key={post.id} blog={post} onSave={handleSave} />
          ))}
        </div>
      )}
    </PageLayout>
  );
}
```

## Conditional Rendering

```tsx
// Good: early return for whole-component conditions
if (!user) return null;

// Good: logical AND for simple presence check
{posts.length > 0 && <PostList posts={posts} />}

// Good: ternary for two-branch toggle
{isEditing ? <EditForm /> : <DisplayView />}

// Bad: nested ternaries
{isLoading ? <Spinner /> : hasError ? <Error /> : data ? <Content /> : <Empty />}

// Good: extract to a variable or helper
function renderContent() {
  if (isLoading) return <Spinner />;
  if (hasError) return <ErrorAlert error={error} />;
  if (!data || data.length === 0) return <EmptyState />;
  return <ContentList data={data} />;
}
```

## Key Props in Lists

Always use stable, unique identifiers. Never use array index as key (unless the list is static and never reordered).

```tsx
// Good: unique ID from the data model
{posts.map((post) => (
  <BlogCard key={post.id} blog={post} />
))}

// Bad: array index as key
{posts.map((post, index) => (
  <BlogCard key={index} blog={post} />
))}
```

## Custom Hook Extraction

Extract a custom hook when:
- State logic is reused across components
- A component has more than 2-3 `useState`/`useEffect` calls for the same concern
- The logic is independently testable

Do NOT extract when:
- The hook would be used in exactly one component and is trivial
- It just wraps a single `useState` with no additional logic

```tsx
// Good: reusable hook with real logic
export function useReadingList() {
  const [items, setItems] = useState<ReadingListItem[]>([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    api.getReadingList().then((data) => {
      if (!cancelled) {
        setItems(data);
        setLoading(false);
      }
    });
    return () => { cancelled = true; };
  }, []);

  async function addItem(postId: string, note: string) {
    const item = await api.addToReadingList({ postId, note });
    setItems((prev) => [...prev, item]);
  }

  async function removeItem(id: string) {
    await api.deleteFromReadingList(id);
    setItems((prev) => prev.filter((item) => item.id !== id));
  }

  return { items, loading, addItem, removeItem };
}
```

## Tailwind CSS Conventions

### Utility-first. Avoid @apply.

Write utilities directly on elements. Do not create CSS classes with `@apply` -- it defeats the purpose of Tailwind.

```tsx
// Good: utility classes
<div className="flex items-center gap-3 rounded-lg border p-4 hover:bg-accent">

// Bad: @apply in CSS file
.blog-card { @apply flex items-center gap-3 rounded-lg border p-4; }
```

### Use the cn() helper for conditional classes

shadcn/ui provides a `cn()` utility (clsx + tailwind-merge). Use it for conditional and merged class names.

```tsx
import { cn } from "@/lib/utils";

<div className={cn(
  "rounded-lg border p-4 transition-colors",
  isSelected && "border-primary bg-primary/5",
  isDisabled && "opacity-50 cursor-not-allowed",
)} />
```

### Responsive design

Use Tailwind breakpoints. Mobile-first.

```tsx
<div className="grid grid-cols-1 gap-4 md:grid-cols-2 lg:grid-cols-3">
```

### Color tokens from shadcn/ui theme

Use semantic color variables, not raw Tailwind colors.

```tsx
// Good: semantic tokens that respect theme
<p className="text-muted-foreground">
<div className="bg-card border">
<span className="text-destructive">

// Bad: hardcoded colors that break in dark mode
<p className="text-gray-500">
<div className="bg-white border-gray-200">
```

## shadcn/ui Usage

shadcn/ui components are copied into `components/ui/`. They are your code -- customize them.

### Adding new components

```bash
npx shadcn@latest add button card dialog alert
```

### Composition over modification

Build complex components by composing shadcn primitives, not by forking them.

```tsx
// Good: compose primitives
export function ConfirmDialog({ title, description, onConfirm, children }: ConfirmDialogProps) {
  return (
    <Dialog>
      <DialogTrigger asChild>{children}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>{title}</DialogTitle>
          <DialogDescription>{description}</DialogDescription>
        </DialogHeader>
        <DialogFooter>
          <DialogClose asChild>
            <Button variant="outline">Cancel</Button>
          </DialogClose>
          <Button variant="destructive" onClick={onConfirm}>
            Confirm
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
```

### Extending shadcn components with variants

Use `cva` (class-variance-authority) for component variants, which is how shadcn/ui itself works.

```tsx
// Already done in shadcn's button.tsx -- follow the same pattern
const badgeVariants = cva(
  "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold",
  {
    variants: {
      variant: {
        default: "bg-primary text-primary-foreground",
        outline: "border text-foreground",
        topic: "bg-blue-100 text-blue-800 dark:bg-blue-900 dark:text-blue-200",
      },
    },
    defaultVariants: { variant: "default" },
  },
);
```

## Import Ordering

Group imports in this order, separated by blank lines:

```tsx
// 1. React and framework imports
import { useState, useEffect } from "react";
import { useNavigate, useParams } from "react-router-dom";

// 2. Third-party libraries
import { toast } from "sonner";

// 3. Internal absolute imports (using @ alias)
import { api } from "@/lib/api-client";
import { usePreferences } from "@/hooks/use-preferences";
import { BlogCard } from "@/components/blog-card";

// 4. shadcn/ui components
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

// 5. Types (type-only imports)
import type { BlogPost, Preferences } from "@/types/models";
```

Use `import type` for type-only imports -- it is erased at compile time and prevents circular dependency issues.

## Error Boundaries

Use error boundaries to catch render errors and show fallback UI. Place them at route boundaries.

```tsx
import { Component } from "react";

interface ErrorBoundaryProps {
  fallback: React.ReactNode;
  children: React.ReactNode;
}

interface ErrorBoundaryState {
  hasError: boolean;
}

// Error boundaries must be class components (React limitation)
export class ErrorBoundary extends Component<ErrorBoundaryProps, ErrorBoundaryState> {
  constructor(props: ErrorBoundaryProps) {
    super(props);
    this.state = { hasError: false };
  }

  static getDerivedStateFromError(): ErrorBoundaryState {
    return { hasError: true };
  }

  componentDidCatch(error: Error, info: React.ErrorInfo) {
    console.error("Uncaught error:", error, info);
  }

  render() {
    if (this.state.hasError) {
      return this.props.fallback;
    }
    return this.props.children;
  }
}

// Usage in router
<ErrorBoundary fallback={<FullPageError />}>
  <RouterProvider router={router} />
</ErrorBoundary>
```

Note: Error boundaries are the one case where a class component is acceptable (React requires it).

## Accessibility

Radix UI (via shadcn) handles most ARIA patterns (dialogs, dropdowns, tooltips). Your responsibilities:

```tsx
// Use semantic HTML
<main>           // not <div id="main">
<nav>            // not <div className="nav">
<section>        // for distinct content sections
<article>        // for blog posts

// Accessible form labels
<Label htmlFor="api-key">API Key</Label>
<Input id="api-key" type="password" />

// Announce dynamic content to screen readers
<div role="status" aria-live="polite">
  {status === "loading" && "Loading posts..."}
  {status === "success" && `Found ${posts.length} posts`}
</div>

// Keyboard navigation -- Radix handles this for its components
// For custom interactive elements, ensure:
<button>         // not <div onClick>
<a href="...">   // not <span onClick>

// Images
<img src={logo} alt="Apricot logo" />
<img src={decorative} alt="" />  // empty alt for decorative images

// Skip link for keyboard users
<a href="#main-content" className="sr-only focus:not-sr-only">
  Skip to main content
</a>
```

## Performance

### Do not prematurely optimize

- Do NOT wrap every component in `React.memo` -- only when profiling shows unnecessary re-renders
- Do NOT `useMemo`/`useCallback` everything -- use them when passing callbacks to memoized children or for expensive computations
- DO use React DevTools Profiler to measure before optimizing

### Lazy load routes

```tsx
import { lazy, Suspense } from "react";

const DiscoverPage = lazy(() => import("@/pages/discover"));
const ReadingListPage = lazy(() => import("@/pages/reading-list"));
const SettingsPage = lazy(() => import("@/pages/settings"));

function AppRouter() {
  return (
    <Suspense fallback={<PageSkeleton />}>
      <Routes>
        <Route path="/" element={<DiscoverPage />} />
        <Route path="/reading-list" element={<ReadingListPage />} />
        <Route path="/settings" element={<SettingsPage />} />
      </Routes>
    </Suspense>
  );
}
```

### Image and asset optimization

Vite handles this. Use static imports for images so they go through the build pipeline:

```tsx
import logo from "@/assets/logo.svg";
<img src={logo} alt="Apricot" />
```

## React Router Patterns

```tsx
// Type-safe route params
function BlogDetailPage() {
  const { id } = useParams<{ id: string }>();
  if (!id) return <Navigate to="/" replace />;

  // ...
}

// Programmatic navigation
function BlogCard({ blog, onSave }: BlogCardProps) {
  const navigate = useNavigate();

  function handleClick() {
    navigate(`/blog/${blog.id}`);
  }

  // ...
}

// Active link styling with NavLink
<NavLink
  to="/reading-list"
  className={({ isActive }) =>
    cn("text-sm font-medium transition-colors", isActive ? "text-primary" : "text-muted-foreground")
  }
>
  Reading List
</NavLink>
```

## Patterns to Avoid

```tsx
// Bad: inline object/array literals in JSX cause unnecessary re-renders
<Component style={{ color: "red" }} />
<Component items={[1, 2, 3]} />

// Good: hoist to module scope or useMemo
const style = { color: "red" };
const items = [1, 2, 3];

// Bad: string concatenation for class names (can produce invalid classes)
className={"p-4 " + (isActive ? "bg-primary" : "")}

// Good: cn() utility
className={cn("p-4", isActive && "bg-primary")}

// Bad: useEffect for derived state
const [fullName, setFullName] = useState("");
useEffect(() => {
  setFullName(`${firstName} ${lastName}`);
}, [firstName, lastName]);

// Good: compute during render
const fullName = `${firstName} ${lastName}`;

// Bad: useEffect to handle events
useEffect(() => {
  if (submitted) {
    sendData();
  }
}, [submitted]);

// Good: call in the event handler
function handleSubmit() {
  sendData();
}
```
