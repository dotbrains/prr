package diff

import (
	"testing"
)

func TestParse_SingleFile(t *testing.T) {
	raw := `diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -1,3 +1,4 @@
 package main
+import "fmt"
 func main() {
+	fmt.Println("hello")
 }`

	files := Parse(raw)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "main.go" {
		t.Errorf("expected path main.go, got %q", files[0].Path)
	}
	if files[0].Status != "modified" {
		t.Errorf("expected status modified, got %q", files[0].Status)
	}
}

func TestParse_MultipleFiles(t *testing.T) {
	raw := `diff --git a/foo.go b/foo.go
--- a/foo.go
+++ b/foo.go
@@ -1 +1 @@
-old
+new
diff --git a/bar.go b/bar.go
--- a/bar.go
+++ b/bar.go
@@ -1 +1 @@
-old2
+new2`

	files := Parse(raw)
	if len(files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(files))
	}
	if files[0].Path != "foo.go" {
		t.Errorf("expected foo.go, got %q", files[0].Path)
	}
	if files[1].Path != "bar.go" {
		t.Errorf("expected bar.go, got %q", files[1].Path)
	}
}

func TestParse_NewFile(t *testing.T) {
	raw := `diff --git a/new.go b/new.go
new file mode 100644
--- /dev/null
+++ b/new.go
@@ -0,0 +1,3 @@
+package main
+
+func hello() {}`

	files := Parse(raw)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != "added" {
		t.Errorf("expected status added, got %q", files[0].Status)
	}
}

func TestParse_DeletedFile(t *testing.T) {
	raw := `diff --git a/old.go b/old.go
deleted file mode 100644
--- a/old.go
+++ /dev/null
@@ -1,3 +0,0 @@
-package main
-
-func old() {}`

	files := Parse(raw)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != "deleted" {
		t.Errorf("expected status deleted, got %q", files[0].Status)
	}
}

func TestParse_RenamedFile(t *testing.T) {
	raw := `diff --git a/old.go b/new.go
rename from old.go
rename to new.go
--- a/old.go
+++ b/new.go`

	files := Parse(raw)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Status != "renamed" {
		t.Errorf("expected status renamed, got %q", files[0].Status)
	}
}

func TestParse_EmptyInput(t *testing.T) {
	files := Parse("")
	if len(files) != 0 {
		t.Errorf("expected 0 files, got %d", len(files))
	}
}

func TestParse_NestedPath(t *testing.T) {
	raw := `diff --git a/src/internal/handler.go b/src/internal/handler.go
--- a/src/internal/handler.go
+++ b/src/internal/handler.go
@@ -1 +1 @@
-old
+new`

	files := Parse(raw)
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}
	if files[0].Path != "src/internal/handler.go" {
		t.Errorf("expected src/internal/handler.go, got %q", files[0].Path)
	}
}

func TestLineCount(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"", 0},
		{"one line", 1},
		{"line1\nline2", 2},
		{"line1\nline2\nline3", 3},
	}

	for _, tt := range tests {
		got := LineCount(tt.input)
		if got != tt.want {
			t.Errorf("LineCount(%q) = %d, want %d", tt.input, got, tt.want)
		}
	}
}
