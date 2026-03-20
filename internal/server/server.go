package server

import (
	"embed"
	"encoding/json"
	"io/fs"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/dotbrains/prr/internal/writer"
)

//go:embed static/*
var staticFS embed.FS

// Server serves the prr web UI and API.
type Server struct {
	reviewsDir string
	mux        *http.ServeMux
}

// New creates a new Server that reads reviews from the given directory.
func New(reviewsDir string) *Server {
	s := &Server{reviewsDir: reviewsDir}
	s.mux = http.NewServeMux()
	s.routes()
	return s
}

// ListenAndServe starts the HTTP server on the given address.
func (s *Server) ListenAndServe(addr string) error {
	srv := &http.Server{
		Addr:         addr,
		Handler:      s.mux,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return srv.ListenAndServe()
}

// Handler returns the underlying http.Handler (for testing).
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/reviews", s.handleListReviews)
	s.mux.HandleFunc("/api/reviews/", s.handleGetReview)

	// Serve embedded static files, falling back to index.html for SPA routing.
	staticSub, _ := fs.Sub(staticFS, "static")
	fileServer := http.FileServer(http.FS(staticSub))

	s.mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" {
			path = "/index.html"
		}

		// Check if the file exists in the embedded FS.
		f, err := staticSub.Open(strings.TrimPrefix(path, "/"))
		if err == nil {
			f.Close()
			// Local UI should always pick up latest assets after rebuilds.
			w.Header().Set("Cache-Control", "no-cache")
			fileServer.ServeHTTP(w, r)
			return
		}

		// Fall back to index.html for SPA routes.
		r.URL.Path = "/"
		fileServer.ServeHTTP(w, r)
	})
}

// reviewListItem is the JSON shape for the reviews list endpoint.
type reviewListItem struct {
	Name      string         `json:"name"`
	PRNumber  int            `json:"pr_number"`
	RepoSlug  string         `json:"repo_slug,omitempty"`
	AgentName string         `json:"agent_name"`
	Model     string         `json:"model,omitempty"`
	CreatedAt string         `json:"created_at"`
	Summary   string         `json:"summary"`
	Stats     map[string]int `json:"stats"`
}

func (s *Server) handleListReviews(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	entries, err := writer.ListReviewDirs(s.reviewsDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	var items []reviewListItem
	for _, e := range entries {
		meta, err := writer.ReadMetadata(e.Path)
		if err != nil {
			continue // skip dirs without valid metadata
		}

		stats := make(map[string]int)
		for _, c := range meta.Comments {
			stats[c.Severity]++
		}

		items = append(items, reviewListItem{
			Name:      e.Name,
			PRNumber:  meta.PRNumber,
			RepoSlug:  meta.RepoSlug,
			AgentName: meta.AgentName,
			Model:     meta.Model,
			CreatedAt: meta.CreatedAt,
			Summary:   meta.Summary,
			Stats:     stats,
		})
	}

	if items == nil {
		items = []reviewListItem{}
	}

	writeJSON(w, items)
}

func (s *Server) handleGetReview(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	name := strings.TrimPrefix(r.URL.Path, "/api/reviews/")
	if name == "" {
		http.Error(w, "review name required", http.StatusBadRequest)
		return
	}

	// Validate name to prevent path traversal.
	if strings.Contains(name, "/") || strings.Contains(name, "..") {
		http.Error(w, "invalid review name", http.StatusBadRequest)
		return
	}

	// Direct read instead of scanning all directories.
	reviewPath := filepath.Join(s.reviewsDir, name)
	meta, err := writer.ReadMetadata(reviewPath)
	if err != nil {
		http.Error(w, "review not found", http.StatusNotFound)
		return
	}

	writeJSON(w, meta)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
