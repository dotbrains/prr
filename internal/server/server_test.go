package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/dotbrains/prr/internal/agent"
	"github.com/dotbrains/prr/internal/writer"
)

func writeFixtureMetadata(t *testing.T, dir string, meta *writer.ReviewMetadata) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writer.WriteMetadata(dir, meta); err != nil {
		t.Fatal(err)
	}
}

func TestListReviews_Empty(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodGet, "/api/reviews", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []reviewListItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("expected 0 reviews, got %d", len(items))
	}
}

func TestListReviews_WithReviews(t *testing.T) {
	tmp := t.TempDir()

	writeFixtureMetadata(t, filepath.Join(tmp, "pr-100-20250311-140000"), &writer.ReviewMetadata{
		PRNumber:  100,
		AgentName: "claude",
		Model:     "opus",
		CreatedAt: "2025-03-11T14:00:00Z",
		Summary:   "Looks good overall.",
		Comments: []agent.ReviewComment{
			{File: "main.go", StartLine: 10, EndLine: 10, Severity: "suggestion", Body: "Use a constant here."},
			{File: "main.go", StartLine: 20, EndLine: 25, Severity: "critical", Body: "Nil pointer dereference."},
		},
	})

	srv := New(tmp)
	req := httptest.NewRequest(http.MethodGet, "/api/reviews", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var items []reviewListItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 review, got %d", len(items))
	}
	if items[0].PRNumber != 100 {
		t.Errorf("expected PR 100, got %d", items[0].PRNumber)
	}
	if items[0].Stats["critical"] != 1 {
		t.Errorf("expected 1 critical, got %d", items[0].Stats["critical"])
	}
	if items[0].Stats["suggestion"] != 1 {
		t.Errorf("expected 1 suggestion, got %d", items[0].Stats["suggestion"])
	}
}

func TestGetReview(t *testing.T) {
	tmp := t.TempDir()
	dirName := "pr-200-20250311-150000"

	writeFixtureMetadata(t, filepath.Join(tmp, dirName), &writer.ReviewMetadata{
		PRNumber:  200,
		AgentName: "gpt",
		Model:     "gpt-4o",
		CreatedAt: "2025-03-11T15:00:00Z",
		Summary:   "Several issues found.",
		Comments: []agent.ReviewComment{
			{File: "auth.go", StartLine: 42, EndLine: 42, Severity: "critical", Body: "Deadlock risk."},
		},
	})

	srv := New(tmp)

	// Existing review.
	req := httptest.NewRequest(http.MethodGet, "/api/reviews/"+dirName, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}

	var meta writer.ReviewMetadata
	if err := json.Unmarshal(w.Body.Bytes(), &meta); err != nil {
		t.Fatal(err)
	}
	if meta.PRNumber != 200 {
		t.Errorf("expected PR 200, got %d", meta.PRNumber)
	}
	if len(meta.Comments) != 1 {
		t.Errorf("expected 1 comment, got %d", len(meta.Comments))
	}
}

func TestGetReview_NotFound(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodGet, "/api/reviews/nonexistent-review", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestStaticFileServing(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	// Root should serve index.html.
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /, got %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct == "" {
		t.Error("expected Content-Type header")
	}
}

func TestMethodNotAllowed(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodPost, "/api/reviews", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /api/reviews, got %d", w.Code)
	}

	req2 := httptest.NewRequest(http.MethodPost, "/api/reviews/some-review", nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)
	if w2.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405 for POST /api/reviews/name, got %d", w2.Code)
	}
}

func TestGetReview_PathTraversal(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	for _, path := range []string{
		"/api/reviews/..%2F..%2Fetc%2Fpasswd",
		"/api/reviews/foo..bar",
		"/api/reviews/foo%2Fbar",
		"/api/reviews/",
	} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		w := httptest.NewRecorder()
		srv.Handler().ServeHTTP(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("expected 400 for %s, got %d", path, w.Code)
		}
	}
}

func TestCacheHeaders(t *testing.T) {
	tmp := t.TempDir()
	writeFixtureMetadata(t, filepath.Join(tmp, "pr-300-20250311-160000"), &writer.ReviewMetadata{
		PRNumber: 300, AgentName: "claude", CreatedAt: "2025-03-11T16:00:00Z",
	})
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodGet, "/api/reviews", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if cc := w.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected no-cache for API, got %q", cc)
	}

	req2 := httptest.NewRequest(http.MethodGet, "/style.css", nil)
	w2 := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w2, req2)
	if cc := w2.Header().Get("Cache-Control"); cc != "no-cache" {
		t.Errorf("expected no-cache for static, got %q", cc)
	}
}

func TestSPAFallback(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodGet, "/some/spa/route", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for SPA fallback, got %d", w.Code)
	}
	if w.Body.Len() == 0 {
		t.Error("expected non-empty body for SPA fallback")
	}
}

func TestListReviews_SkipsInvalidDirs(t *testing.T) {
	tmp := t.TempDir()
	writeFixtureMetadata(t, filepath.Join(tmp, "pr-400-20250311-170000"), &writer.ReviewMetadata{
		PRNumber: 400, AgentName: "claude", CreatedAt: "2025-03-11T17:00:00Z",
	})
	// Dir without metadata.
	if err := os.MkdirAll(filepath.Join(tmp, "pr-401-20250311-180000"), 0o755); err != nil {
		t.Fatal(err)
	}

	srv := New(tmp)
	req := httptest.NewRequest(http.MethodGet, "/api/reviews", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)

	var items []reviewListItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Errorf("expected 1 review (skipping invalid), got %d", len(items))
	}
}

func TestStaticFiles_Favicon(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodGet, "/favicon.svg", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /favicon.svg, got %d", w.Code)
	}
}

func TestStaticFiles_JS(t *testing.T) {
	tmp := t.TempDir()
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodGet, "/app.js", nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 for /app.js, got %d", w.Code)
	}
}

func TestGetReview_ContentType(t *testing.T) {
	tmp := t.TempDir()
	dirName := "pr-500-20250311-190000"
	writeFixtureMetadata(t, filepath.Join(tmp, dirName), &writer.ReviewMetadata{
		PRNumber: 500, AgentName: "claude", CreatedAt: "2025-03-11T19:00:00Z",
	})
	srv := New(tmp)

	req := httptest.NewRequest(http.MethodGet, "/api/reviews/"+dirName, nil)
	w := httptest.NewRecorder()
	srv.Handler().ServeHTTP(w, req)
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("expected application/json, got %q", ct)
	}
}
