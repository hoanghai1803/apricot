# Go Style Guide

Follow the [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md) and avoid the [100 Go Mistakes](https://100go.co/). Key rules for this project:

## Error Handling

- Handle errors exactly once -- either log OR return, never both
- Wrap errors with context: `fmt.Errorf("fetching feed %s: %w", url, err)`
- Use `errors.Is` for sentinel errors, `errors.As` for error types (not `==` or type assertions)
- Always handle defer errors where they matter (e.g., `db.Close()`, `resp.Body.Close()`)
- Return after replying to an HTTP request -- don't let handlers fall through

```go
// Good: wrap with context, return once
func (s *Store) GetBlog(id int64) (*Blog, error) {
    row := s.db.QueryRow("SELECT id, name, url FROM blogs WHERE id = ?", id)
    var b Blog
    if err := row.Scan(&b.ID, &b.Name, &b.URL); err != nil {
        return nil, fmt.Errorf("getting blog %d: %w", id, err)
    }
    return &b, nil
}

// Good: handle defer error
func (s *Store) ListBlogs() ([]Blog, error) {
    rows, err := s.db.Query("SELECT id, name, url FROM blogs")
    if err != nil {
        return nil, fmt.Errorf("querying blogs: %w", err)
    }
    defer rows.Close()
    // ...
}
```

## Interfaces

- Define interfaces on the consumer side, not the producer side
- Return concrete types, accept interfaces
- Verify interface compliance at compile time: `var _ AIProvider = (*AnthropicProvider)(nil)`
- Don't create interfaces until you have a real need (avoid interface pollution)
- Never use `any` unless you genuinely accept any type

```go
// Good: consumer defines the interface it needs
// In internal/api/handlers/discover.go
type ContentFilter interface {
    FilterAndRank(ctx context.Context, posts []models.Post, prefs models.Preferences) ([]models.Post, error)
}

// Good: compile-time check in the producer package
var _ ai.Provider = (*AnthropicProvider)(nil)
```

## Concurrency

- Every goroutine must have a known exit path -- never fire-and-forget
- Use `errgroup` for concurrent work with error propagation (already a dependency)
- Channel sizes should be 0 or 1; larger buffers need clear justification
- Use channels for communication, mutexes for protecting shared state
- Never copy sync types (`sync.Mutex`, `sync.WaitGroup`, etc.) -- pass by pointer
- Protect entire slice/map operations with mutexes, not just individual field access

```go
// Good: errgroup with context cancellation
func (f *Fetcher) FetchAll(ctx context.Context, urls []string) ([]Feed, error) {
    g, ctx := errgroup.WithContext(ctx)
    results := make([]Feed, len(urls))
    for i, url := range urls {
        g.Go(func() error {
            feed, err := f.fetch(ctx, url)
            if err != nil {
                return fmt.Errorf("fetching %s: %w", url, err)
            }
            results[i] = feed // safe: each goroutine writes to its own index
            return nil
        })
    }
    if err := g.Wait(); err != nil {
        return nil, err
    }
    return results, nil
}
```

## Naming & Organization

- No utility packages (`common`, `util`, `shared`, `helpers`) -- use meaningful names
- Package names: concise, lowercase, no underscores
- Don't shadow built-in names (`len`, `cap`, `error`, `copy`)
- Don't shadow variables in inner scopes -- use distinct names
- Reduce nesting with early returns; avoid unnecessary `else` after `return`

```go
// Bad: utility package
package helpers
func FormatDate(t time.Time) string { ... }

// Good: meaningful package
package blog
func FormatPublishDate(t time.Time) string { ... }

// Bad: shadowing
func process(items []string) {
    for _, err := range items {  // shadows the error type
        // ...
    }
}

// Good: early return reduces nesting
func (h *Handler) GetBlog(w http.ResponseWriter, r *http.Request) {
    id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
    if err != nil {
        http.Error(w, "invalid id", http.StatusBadRequest)
        return
    }

    blog, err := h.store.GetBlog(id)
    if err != nil {
        http.Error(w, "not found", http.StatusNotFound)
        return
    }

    json.NewEncoder(w).Encode(blog)
}
```

## Performance

- Pre-allocate slices and maps when size is known or estimable
- Use `strings.Builder` for string concatenation in loops (not `+=`)
- Prefer `strconv` over `fmt` for simple conversions
- Avoid repeated `string` <-> `[]byte` conversions -- use `bytes` package equivalents
- Copy slices/maps at API boundaries to prevent unintended mutation

```go
// Good: pre-allocate
posts := make([]Post, 0, len(rawPosts))
tagMap := make(map[string]int, len(tags))

// Good: strings.Builder
var b strings.Builder
for _, tag := range tags {
    b.WriteString(tag)
    b.WriteByte(',')
}

// Good: copy at boundary
func (s *Store) Tags() []string {
    s.mu.RLock()
    defer s.mu.RUnlock()
    out := make([]string, len(s.tags))
    copy(out, s.tags)
    return out
}
```

## Functions & Methods

- Use pointer receivers for mutation or large structs; value receivers for small immutable types
- Accept `io.Reader`/`io.Writer` instead of filenames for testability
- Use functional options pattern for complex configuration (already used for AI providers)
- Defer arguments are evaluated immediately -- use closures for late evaluation

```go
// Good: functional options
type Option func(*Client)

func WithTimeout(d time.Duration) Option {
    return func(c *Client) { c.timeout = d }
}

func NewClient(opts ...Option) *Client {
    c := &Client{timeout: 30 * time.Second}
    for _, opt := range opts {
        opt(c)
    }
    return c
}

// Good: defer with closure for late evaluation
func process() error {
    start := time.Now()
    defer func() {
        log.Printf("process took %s", time.Since(start))
    }()
    // ...
}
```

## SQL / Resources

- Always close transient resources: `sql.Rows`, `http.Response.Body`, file handles
- Use `defer rows.Close()` immediately after getting rows from a query
- Customize HTTP clients with timeouts -- never use `http.DefaultClient` for external calls

```go
// Good: close rows immediately
rows, err := db.QueryContext(ctx, "SELECT id, title FROM posts WHERE blog_id = ?", blogID)
if err != nil {
    return nil, fmt.Errorf("querying posts for blog %d: %w", blogID, err)
}
defer rows.Close()

// Good: custom HTTP client with timeout
var httpClient = &http.Client{
    Timeout: 15 * time.Second,
    Transport: &http.Transport{
        MaxIdleConnsPerHost: 10,
    },
}
```

---

# Go Unit Testing Guide

This project uses only the Go standard library for testing (`testing`, `net/http/httptest`, `testing/iotest`). No testify, no gomock -- stdlib only.

## File Naming and Placement

Test files live in the same package as the code they test. Use `_test.go` suffix.

```
internal/storage/
    store.go
    store_test.go          # tests for store.go
    migrations.go
    migrations_test.go     # tests for migrations.go

internal/api/handlers/
    discover.go
    discover_test.go
```

Use the same package name (not `_test` suffix package) so you can test unexported functions when needed. Use the `_test` package suffix only when testing the public API to verify the consumer experience:

```go
// internal/storage/store_test.go -- same package, can access unexported
package storage

// internal/storage/store_api_test.go -- external test package
package storage_test
```

## Test Function Naming

Follow `TestXxx` for top-level functions. Use `TestType_Method` for method tests. Use underscores to separate logical parts:

```go
func TestStore_GetBlog(t *testing.T) { ... }
func TestStore_GetBlog_NotFound(t *testing.T) { ... }
func TestFilterAndRank_EmptyInput(t *testing.T) { ... }
func TestParseFeedURL(t *testing.T) { ... }
```

## Table-Driven Tests

The standard pattern for this project. Every test with more than one case should be table-driven.

```go
func TestParseFeedURL(t *testing.T) {
    tests := []struct {
        name    string
        input   string
        want    string
        wantErr bool
    }{
        {
            name:  "valid RSS URL",
            input: "https://blog.example.com/feed.xml",
            want:  "https://blog.example.com/feed.xml",
        },
        {
            name:  "adds scheme if missing",
            input: "blog.example.com/feed.xml",
            want:  "https://blog.example.com/feed.xml",
        },
        {
            name:    "empty string",
            input:   "",
            wantErr: true,
        },
        {
            name:    "invalid URL",
            input:   "://not-a-url",
            wantErr: true,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := ParseFeedURL(tt.input)
            if tt.wantErr {
                if err == nil {
                    t.Fatalf("ParseFeedURL(%q) expected error, got %q", tt.input, got)
                }
                return
            }
            if err != nil {
                t.Fatalf("ParseFeedURL(%q) unexpected error: %v", tt.input, err)
            }
            if got != tt.want {
                t.Errorf("ParseFeedURL(%q) = %q, want %q", tt.input, got, tt.want)
            }
        })
    }
}
```

Key rules for table-driven tests:
- Always include a `name` field -- it becomes the subtest name
- Use `t.Run` so failures identify which case broke
- Use `t.Fatalf` for conditions that make further assertions meaningless (guards)
- Use `t.Errorf` for conditions where the test can continue checking other assertions
- Keep test struct fields minimal -- only what varies between cases

## Subtests with t.Run

Use `t.Run` for logical groupings within a test function:

```go
func TestStore_ReadingList(t *testing.T) {
    store := newTestStore(t)

    t.Run("add item", func(t *testing.T) {
        item := models.ReadingListItem{PostID: 1, Note: "interesting"}
        id, err := store.AddToReadingList(item)
        if err != nil {
            t.Fatalf("AddToReadingList: %v", err)
        }
        if id == 0 {
            t.Error("expected non-zero ID")
        }
    })

    t.Run("list items", func(t *testing.T) {
        items, err := store.ListReadingList()
        if err != nil {
            t.Fatalf("ListReadingList: %v", err)
        }
        if len(items) == 0 {
            t.Error("expected at least one item")
        }
    })

    t.Run("delete item", func(t *testing.T) {
        err := store.RemoveFromReadingList(1)
        if err != nil {
            t.Fatalf("RemoveFromReadingList: %v", err)
        }
    })
}
```

## Test Helpers with t.Helper

Mark helper functions with `t.Helper()` so failure line numbers point to the caller, not the helper. Accept `testing.TB` to work with both tests and benchmarks.

```go
func newTestStore(t testing.TB) *Store {
    t.Helper()
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        t.Fatalf("opening test db: %v", err)
    }
    t.Cleanup(func() { db.Close() })

    store, err := New(db)
    if err != nil {
        t.Fatalf("creating store: %v", err)
    }
    return store
}

func assertEqualJSON(t testing.TB, got, want string) {
    t.Helper()
    var gotVal, wantVal any
    if err := json.Unmarshal([]byte(got), &gotVal); err != nil {
        t.Fatalf("unmarshaling got: %v", err)
    }
    if err := json.Unmarshal([]byte(want), &wantVal); err != nil {
        t.Fatalf("unmarshaling want: %v", err)
    }
    if !reflect.DeepEqual(gotVal, wantVal) {
        t.Errorf("JSON mismatch:\ngot:  %s\nwant: %s", got, want)
    }
}
```

## t.Cleanup for Resource Teardown

Use `t.Cleanup` instead of manual teardown. It runs after the test (and all subtests) complete, even on failure.

```go
func newTestServer(t *testing.T, handler http.Handler) *httptest.Server {
    t.Helper()
    srv := httptest.NewServer(handler)
    t.Cleanup(srv.Close)
    return srv
}

func newTestDB(t *testing.T) *sql.DB {
    t.Helper()
    dir := t.TempDir() // automatically cleaned up
    db, err := sql.Open("sqlite", filepath.Join(dir, "test.db"))
    if err != nil {
        t.Fatalf("opening db: %v", err)
    }
    t.Cleanup(func() { db.Close() })
    return db
}
```

## HTTP Handler Testing with httptest

Use `httptest.NewRecorder` for unit testing individual handlers. Use `httptest.NewServer` for integration tests that need a real TCP connection.

### Unit testing a handler

```go
func TestDiscoverHandler_POST(t *testing.T) {
    store := newTestStore(t)
    // seed test data
    store.SetPreferences(models.Preferences{Topics: []string{"go", "rust"}})

    handler := NewDiscoverHandler(store, &mockAIProvider{})

    body := strings.NewReader(`{}`)
    req := httptest.NewRequest(http.MethodPost, "/api/discover", body)
    req.Header.Set("Content-Type", "application/json")
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusOK {
        t.Fatalf("status = %d, want %d; body: %s", rec.Code, http.StatusOK, rec.Body.String())
    }

    var resp DiscoverResponse
    if err := json.NewDecoder(rec.Body).Decode(&resp); err != nil {
        t.Fatalf("decoding response: %v", err)
    }
    if len(resp.Posts) == 0 {
        t.Error("expected posts in response")
    }
}
```

### Testing HTTP method routing

```go
func TestDiscoverHandler_MethodNotAllowed(t *testing.T) {
    handler := NewDiscoverHandler(nil, nil)

    for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
        t.Run(method, func(t *testing.T) {
            req := httptest.NewRequest(method, "/api/discover", nil)
            rec := httptest.NewRecorder()
            handler.ServeHTTP(rec, req)

            if rec.Code != http.StatusMethodNotAllowed {
                t.Errorf("%s: status = %d, want %d", method, rec.Code, http.StatusMethodNotAllowed)
            }
        })
    }
}
```

### Testing with a full server (integration)

```go
func TestAPIIntegration(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }

    router := setupTestRouter(t)
    srv := httptest.NewServer(router)
    t.Cleanup(srv.Close)

    resp, err := http.Get(srv.URL + "/api/preferences")
    if err != nil {
        t.Fatalf("GET /api/preferences: %v", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode != http.StatusOK {
        t.Fatalf("status = %d, want %d", resp.StatusCode, http.StatusOK)
    }
}
```

## Mocking with Interfaces (No Frameworks)

Define small interfaces in the consumer package. Create test doubles as structs with function fields for maximum flexibility.

```go
// In test file: configurable mock
type mockAIProvider struct {
    filterFunc    func(ctx context.Context, posts []models.Post, prefs models.Preferences) ([]models.Post, error)
    summarizeFunc func(ctx context.Context, content string) (string, error)
}

func (m *mockAIProvider) FilterAndRank(ctx context.Context, posts []models.Post, prefs models.Preferences) ([]models.Post, error) {
    if m.filterFunc != nil {
        return m.filterFunc(ctx, posts, prefs)
    }
    return posts, nil // default: pass through
}

func (m *mockAIProvider) Summarize(ctx context.Context, content string) (string, error) {
    if m.summarizeFunc != nil {
        return m.summarizeFunc(ctx, content)
    }
    return "test summary", nil // default: stub
}
```

Use the mock in tests:

```go
func TestDiscoverHandler_AIError(t *testing.T) {
    provider := &mockAIProvider{
        filterFunc: func(ctx context.Context, posts []models.Post, prefs models.Preferences) ([]models.Post, error) {
            return nil, fmt.Errorf("API rate limited")
        },
    }

    handler := NewDiscoverHandler(newTestStore(t), provider)
    req := httptest.NewRequest(http.MethodPost, "/api/discover", strings.NewReader(`{}`))
    rec := httptest.NewRecorder()

    handler.ServeHTTP(rec, req)

    if rec.Code != http.StatusInternalServerError {
        t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
    }
}
```

## SQLite Testing with In-Memory Databases

Use `:memory:` SQLite databases for fast, isolated tests. Run the same migrations as production.

```go
func newTestStore(t *testing.T) *Store {
    t.Helper()
    db, err := sql.Open("sqlite", ":memory:")
    if err != nil {
        t.Fatalf("opening test db: %v", err)
    }
    t.Cleanup(func() { db.Close() })

    // Enable WAL and foreign keys like production
    if _, err := db.Exec("PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;"); err != nil {
        t.Fatalf("setting pragmas: %v", err)
    }

    store, err := NewWithMigrations(db)
    if err != nil {
        t.Fatalf("running migrations: %v", err)
    }
    return store
}

func TestStore_CreateAndGetBlog(t *testing.T) {
    store := newTestStore(t)

    blog := models.BlogSource{
        Name:    "Go Blog",
        FeedURL: "https://go.dev/blog/feed.atom",
    }

    id, err := store.CreateBlogSource(blog)
    if err != nil {
        t.Fatalf("CreateBlogSource: %v", err)
    }

    got, err := store.GetBlogSource(id)
    if err != nil {
        t.Fatalf("GetBlogSource(%d): %v", id, err)
    }
    if got.Name != blog.Name {
        t.Errorf("Name = %q, want %q", got.Name, blog.Name)
    }
    if got.FeedURL != blog.FeedURL {
        t.Errorf("FeedURL = %q, want %q", got.FeedURL, blog.FeedURL)
    }
}
```

## Error Assertion Patterns (Without Testify)

```go
// Assert specific error
func TestStore_GetBlog_NotFound(t *testing.T) {
    store := newTestStore(t)

    _, err := store.GetBlogSource(999)
    if err == nil {
        t.Fatal("expected error for non-existent blog")
    }
    if !errors.Is(err, ErrNotFound) {
        t.Errorf("error = %v, want %v", err, ErrNotFound)
    }
}

// Assert error contains message
func assertErrorContains(t testing.TB, err error, substr string) {
    t.Helper()
    if err == nil {
        t.Fatalf("expected error containing %q, got nil", substr)
    }
    if !strings.Contains(err.Error(), substr) {
        t.Errorf("error %q does not contain %q", err.Error(), substr)
    }
}

// Assert error type
func TestFetcher_Timeout(t *testing.T) {
    _, err := fetcher.Fetch(ctx, slowURL)
    if err == nil {
        t.Fatal("expected timeout error")
    }
    var netErr net.Error
    if !errors.As(err, &netErr) || !netErr.Timeout() {
        t.Errorf("expected timeout error, got: %v", err)
    }
}
```

## Golden File Testing

Use golden files for large expected outputs (JSON responses, rendered templates). Store them in `testdata/` directories which `go test` ignores.

```go
var update = flag.Bool("update", false, "update golden files")

func TestRenderFeed_Golden(t *testing.T) {
    feed := buildTestFeed()
    got, err := RenderFeed(feed)
    if err != nil {
        t.Fatalf("RenderFeed: %v", err)
    }

    golden := filepath.Join("testdata", t.Name()+".golden")

    if *update {
        if err := os.WriteFile(golden, got, 0o644); err != nil {
            t.Fatalf("updating golden file: %v", err)
        }
    }

    want, err := os.ReadFile(golden)
    if err != nil {
        t.Fatalf("reading golden file (run with -update to create): %v", err)
    }

    if !bytes.Equal(got, want) {
        t.Errorf("output mismatch (run with -update to accept):\ngot:\n%s\nwant:\n%s", got, want)
    }
}
```

Run `go test -update ./...` to regenerate golden files after intentional changes.

## Parallel Tests

Use `t.Parallel()` for tests that don't share mutable state. In table-driven tests, capture the loop variable:

```go
func TestParseFeedURL_Parallel(t *testing.T) {
    tests := []struct {
        name  string
        input string
        want  string
    }{
        {"with scheme", "https://example.com/feed", "https://example.com/feed"},
        {"without scheme", "example.com/feed", "https://example.com/feed"},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            t.Parallel()
            got, err := ParseFeedURL(tt.input)
            if err != nil {
                t.Fatalf("unexpected error: %v", err)
            }
            if got != tt.want {
                t.Errorf("got %q, want %q", got, tt.want)
            }
        })
    }
}
```

Do NOT use `t.Parallel()` in tests that:
- Share an in-memory SQLite database (SQLite is single-writer)
- Depend on sequential execution (test A creates data test B reads)
- Modify package-level state

## Benchmark Tests

Place benchmarks in `_test.go` files. Run with `go test -bench=. -benchmem ./...`.

```go
func BenchmarkFilterPosts(b *testing.B) {
    posts := generateTestPosts(1000)
    prefs := models.Preferences{Topics: []string{"go", "rust", "kubernetes"}}

    b.ResetTimer()
    for b.Loop() {
        FilterPosts(posts, prefs)
    }
}

func BenchmarkStore_ListPosts(b *testing.B) {
    store := newTestStore(b) // accepts testing.TB
    seedPosts(b, store, 500)

    b.ResetTimer()
    for b.Loop() {
        if _, err := store.ListPosts(); err != nil {
            b.Fatalf("ListPosts: %v", err)
        }
    }
}
```

## Fuzz Tests

Use fuzz tests to find edge cases in parsers and validators. Go 1.18+.

```go
func FuzzParseFeedURL(f *testing.F) {
    // Seed corpus with known inputs
    f.Add("https://example.com/feed.xml")
    f.Add("example.com/feed")
    f.Add("")
    f.Add("://broken")

    f.Fuzz(func(t *testing.T, input string) {
        result, err := ParseFeedURL(input)
        if err != nil {
            return // invalid input is fine
        }
        // If parsing succeeded, result must be a valid URL
        u, err := url.Parse(result)
        if err != nil {
            t.Errorf("ParseFeedURL(%q) returned invalid URL %q: %v", input, result, err)
        }
        if u.Scheme != "https" && u.Scheme != "http" {
            t.Errorf("ParseFeedURL(%q) returned scheme %q, want http(s)", input, u.Scheme)
        }
    })
}
```

Run with `go test -fuzz=FuzzParseFeedURL ./internal/feeds/`.

## Test Flags

```bash
go test ./...                    # run all tests
go test -race ./...              # ALWAYS use in CI -- detects data races
go test -count=1 ./...           # disable test caching (useful for flaky test debugging)
go test -short ./...             # skip slow/integration tests
go test -v ./...                 # verbose output with t.Log messages
go test -run TestStore ./...     # run only tests matching pattern
go test -bench=. -benchmem ./... # run benchmarks with memory stats
go test -cover ./...             # show coverage percentage
go test -coverprofile=cover.out ./... && go tool cover -html=cover.out  # HTML coverage report
```

Guard slow tests with `-short`:

```go
func TestFullDiscoveryPipeline(t *testing.T) {
    if testing.Short() {
        t.Skip("skipping integration test in short mode")
    }
    // expensive test that hits real DB, etc.
}
```

## Testing io.Reader/io.Writer with iotest

Use `testing/iotest` to test code that accepts `io.Reader` or `io.Writer`:

```go
func TestParseConfig_BrokenReader(t *testing.T) {
    // ErrReader always returns an error
    _, err := ParseConfig(iotest.ErrReader(fmt.Errorf("disk failure")))
    if err == nil {
        t.Fatal("expected error from broken reader")
    }
}

func TestParseConfig_OneByteReader(t *testing.T) {
    // OneByteReader reads one byte at a time -- stresses buffering logic
    data := []byte(`[server]\nport = 8080`)
    _, err := ParseConfig(iotest.OneByteReader(bytes.NewReader(data)))
    if err != nil {
        t.Fatalf("unexpected error with one-byte reader: %v", err)
    }
}
```

## Test Organization Checklist

1. One `_test.go` per source file
2. `newTestX(t)` helpers for test fixtures -- always use `t.Helper()` and `t.Cleanup()`
3. Table-driven tests for any function with multiple cases
4. `t.Run` for subtests within a logical group
5. `testdata/` directory for golden files and fixture data
6. No `time.Sleep` -- use channels, sync primitives, or `httptest`
7. No global test state -- each test creates its own fixtures
8. `-race` in CI always; `-short` to skip integration tests locally
