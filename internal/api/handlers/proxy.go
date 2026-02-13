package handlers

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// proxyClient is a dedicated HTTP client for proxying blog pages. It mirrors
// the feeds fetcher transport settings (TLS 1.2+, browser-like User-Agent,
// 20s TLS handshake timeout).
var proxyClient = &http.Client{
	Timeout: 30 * time.Second,
	Transport: &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          50,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:  20 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	},
}

var headTagRe = regexp.MustCompile(`(?i)<head[^>]*>`)

// ProxyPage handles GET /api/proxy?url=<encoded-url>. It fetches the target
// page, strips X-Frame-Options and CSP frame-ancestors headers so it can be
// embedded in an iframe, and injects <base target="_blank"> so all links
// inside the iframe open in a new tab instead of navigating within it.
func ProxyPage() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		targetURL := r.URL.Query().Get("url")
		if targetURL == "" {
			writeError(w, http.StatusBadRequest, "url parameter is required")
			return
		}

		parsed, err := url.Parse(targetURL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			writeError(w, http.StatusBadRequest, "url must be a valid HTTP or HTTPS URL")
			return
		}

		req, err := http.NewRequestWithContext(r.Context(), "GET", targetURL, nil)
		if err != nil {
			writeError(w, http.StatusInternalServerError, "failed to create request")
			return
		}

		// Browser-like headers to avoid bot detection.
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36")
		req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
		req.Header.Set("Accept-Language", "en-US,en;q=0.9")

		resp, err := proxyClient.Do(req)
		if err != nil {
			slog.Warn("proxy fetch failed", "url", targetURL, "error", err)
			writeError(w, http.StatusBadGateway, "failed to fetch page")
			return
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")

		// For HTML pages: strip iframe-blocking headers, inject <base> tag.
		if strings.Contains(contentType, "text/html") {
			body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10 MB limit
			if err != nil {
				writeError(w, http.StatusBadGateway, "failed to read page")
				return
			}

			html := string(body)

			// Inject <base href="original-origin" target="_blank"> after <head>
			// so relative resources resolve correctly and links open in new tabs.
			baseTag := `<base href="` + parsed.Scheme + `://` + parsed.Host + `/" target="_blank">`
			if loc := headTagRe.FindStringIndex(html); loc != nil {
				html = html[:loc[1]] + baseTag + html[loc[1]:]
			}

			w.Header().Set("Content-Type", contentType)
			// Do NOT copy X-Frame-Options or CSP headers from upstream.
			w.WriteHeader(resp.StatusCode)
			io.WriteString(w, html) //nolint:errcheck
			return
		}

		// Non-HTML resources: pass through as-is.
		w.Header().Set("Content-Type", contentType)
		if ce := resp.Header.Get("Content-Encoding"); ce != "" {
			w.Header().Set("Content-Encoding", ce)
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body) //nolint:errcheck
	}
}
