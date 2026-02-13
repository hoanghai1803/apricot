package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/hoanghai1803/apricot/internal/ai"
	"github.com/hoanghai1803/apricot/internal/api/handlers"
	"github.com/hoanghai1803/apricot/internal/config"
	"github.com/hoanghai1803/apricot/internal/feeds"
	"github.com/hoanghai1803/apricot/internal/storage"
)

//go:embed all:dist
var distFS embed.FS

// NewRouter creates and configures the HTTP router with all API routes and
// static file serving for the React SPA.
func NewRouter(store *storage.Store, aiProvider ai.AIProvider, fetcher *feeds.Fetcher, cfg *config.Config) *chi.Mux {
	r := chi.NewRouter()

	// Global middleware.
	r.Use(RequestLogger)
	r.Use(Recovery)
	r.Use(CORS)

	// API sub-router.
	r.Route("/api", func(api chi.Router) {
		api.Post("/discover", handlers.Discover(store, aiProvider, fetcher, cfg))
		api.Get("/discover/latest", handlers.GetLatestDiscovery(store))

		api.Get("/preferences", handlers.GetPreferences(store))
		api.Put("/preferences", handlers.UpdatePreferences(store))

		api.Get("/reading-list", handlers.GetReadingList(store))
		api.Post("/reading-list", handlers.AddToReadingList(store))
		api.Post("/reading-list/custom", handlers.AddCustomBlog(store, aiProvider, cfg))
		api.Patch("/reading-list/{id}", handlers.UpdateReadingListItem(store))
		api.Delete("/reading-list/{id}", handlers.DeleteReadingListItem(store))
		api.Post("/reading-list/{id}/tags", handlers.AddTagToItem(store))
		api.Delete("/reading-list/{id}/tags/{tag}", handlers.RemoveTagFromItem(store))

		api.Get("/tags", handlers.GetAllTags(store))
		api.Get("/search", handlers.SearchBlogs(store))

		api.Get("/sources", handlers.GetSources(store))
		api.Put("/sources/{id}", handlers.ToggleSource(store))
	})

	// Serve React SPA from the embedded dist/ directory.
	distContent, _ := fs.Sub(distFS, "dist")
	fileServer := http.FileServer(http.FS(distContent))

	// SPA fallback: serve index.html for any non-API GET request that does
	// not match a static file.
	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		// Try to open the file first.
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if the file exists in the embedded FS.
		f, err := distContent.Open(path[1:]) // strip leading /
		if err != nil {
			// File not found — serve index.html for SPA client-side routing.
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}
		f.Close()

		// File exists — serve it directly.
		fileServer.ServeHTTP(w, r)
	})

	return r
}
