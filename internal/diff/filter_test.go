package diff

import (
	"testing"

	"github.com/dotbrains/prr/internal/agent"
)

func TestFilter_IgnoreLockFiles(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "main.go"},
		{Path: "go.sum"},
		{Path: "package-lock.json"},
		{Path: "yarn.lock"},
		{Path: "src/app.ts"},
	}

	patterns := []string{"*.lock", "go.sum", "package-lock.json"}

	kept, filtered := Filter(files, patterns)
	if filtered != 3 {
		t.Errorf("expected 3 filtered, got %d", filtered)
	}
	if len(kept) != 2 {
		t.Fatalf("expected 2 kept, got %d", len(kept))
	}
	if kept[0].Path != "main.go" {
		t.Errorf("expected main.go, got %q", kept[0].Path)
	}
	if kept[1].Path != "src/app.ts" {
		t.Errorf("expected src/app.ts, got %q", kept[1].Path)
	}
}

func TestFilter_IgnoreVendorDir(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "main.go"},
		{Path: "vendor/github.com/pkg/errors/errors.go"},
		{Path: "vendor/golang.org/x/sys/unix.go"},
	}

	patterns := []string{"vendor/**"}

	kept, filtered := Filter(files, patterns)
	if filtered != 2 {
		t.Errorf("expected 2 filtered, got %d", filtered)
	}
	if len(kept) != 1 {
		t.Fatalf("expected 1 kept, got %d", len(kept))
	}
}

func TestFilter_IgnoreMinifiedFiles(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "src/app.js"},
		{Path: "dist/bundle.min.js"},
		{Path: "dist/style.min.css"},
	}

	patterns := []string{"*.min.js", "*.min.css"}

	kept, filtered := Filter(files, patterns)
	if filtered != 2 {
		t.Errorf("expected 2 filtered, got %d", filtered)
	}
	if len(kept) != 1 {
		t.Fatalf("expected 1 kept, got %d", len(kept))
	}
}

func TestFilter_NoPatterns(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "main.go"},
		{Path: "go.sum"},
	}

	kept, filtered := Filter(files, nil)
	if filtered != 0 {
		t.Errorf("expected 0 filtered, got %d", filtered)
	}
	if len(kept) != 2 {
		t.Errorf("expected 2 kept, got %d", len(kept))
	}
}

func TestFilter_EmptyFiles(t *testing.T) {
	kept, filtered := Filter(nil, []string{"*.lock"})
	if filtered != 0 {
		t.Errorf("expected 0 filtered, got %d", filtered)
	}
	if len(kept) != 0 {
		t.Errorf("expected 0 kept, got %d", len(kept))
	}
}

func TestFilter_NodeModules(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "src/index.ts"},
		{Path: "node_modules/express/index.js"},
		{Path: "node_modules/@types/node/index.d.ts"},
	}

	patterns := []string{"node_modules/**"}

	kept, filtered := Filter(files, patterns)
	if filtered != 2 {
		t.Errorf("expected 2 filtered, got %d", filtered)
	}
	if len(kept) != 1 {
		t.Fatalf("expected 1 kept, got %d", len(kept))
	}
}

func TestFilter_GeneratedFiles(t *testing.T) {
	files := []agent.FileDiff{
		{Path: "src/main.go"},
		{Path: "src/types.generated.go"},
		{Path: "proto/api.generated.ts"},
	}

	patterns := []string{"*.generated.*"}

	kept, filtered := Filter(files, patterns)
	if filtered != 2 {
		t.Errorf("expected 2 filtered, got %d", filtered)
	}
	if len(kept) != 1 {
		t.Fatalf("expected 1 kept, got %d", len(kept))
	}
}
