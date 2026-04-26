package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestBuildTemplateIndexMatchesExactHash(t *testing.T) {
	root := t.TempDir()
	templatePath := filepath.Join(root, "Vehicle", "Left.png")
	writeBytes(t, templatePath, []byte("same image"))
	writeBytes(t, filepath.Join(root, "Vehicle", "Right.png"), []byte("other image"))

	index, err := BuildTemplateIndex(root)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if index.Count() != 2 {
		t.Fatalf("expected 2 templates, got %d", index.Count())
	}

	sha256, err := SHA256File(templatePath)
	if err != nil {
		t.Fatalf("hash template: %v", err)
	}

	match, ok := index.MatchSHA256(sha256)

	if !ok {
		t.Fatal("expected exact hash match")
	}
	if match.RelPath != "Vehicle/Left.png" {
		t.Fatalf("expected first template match, got %q", match.RelPath)
	}
}

func TestBuildTemplateIndexIgnoresUnsupportedFiles(t *testing.T) {
	root := t.TempDir()
	writeBytes(t, filepath.Join(root, "readme.txt"), []byte("not an image"))
	writeBytes(t, filepath.Join(root, "Top.PNG"), []byte("image"))

	index, err := BuildTemplateIndex(root)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if index.Count() != 1 {
		t.Fatalf("expected 1 template, got %d", index.Count())
	}
}

func TestTemplateIndexMatchImage(t *testing.T) {
	root := t.TempDir()
	templatePath := filepath.Join(root, "Back.png")
	writeBytes(t, templatePath, []byte("same image"))

	index, err := BuildTemplateIndex(root)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	sha256, err := SHA256File(templatePath)
	if err != nil {
		t.Fatalf("hash template: %v", err)
	}

	match, ok := index.MatchImage(ImageFile{SHA256: sha256})

	if !ok {
		t.Fatal("expected image match")
	}
	if match.RelPath != "Back.png" {
		t.Fatalf("expected Back.png match, got %q", match.RelPath)
	}
}

func TestTemplateIndexDoesNotMatchMissingOrEmptyHash(t *testing.T) {
	index := TemplateIndex{}

	if _, ok := index.MatchSHA256("missing"); ok {
		t.Fatal("expected no match for nil index")
	}
	if _, ok := index.MatchImage(ImageFile{}); ok {
		t.Fatal("expected no match for image without hash")
	}
}

func TestBuildTemplateIndexRejectsMissingRoot(t *testing.T) {
	_, err := BuildTemplateIndex(filepath.Join(t.TempDir(), "missing"))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func writeBytes(t *testing.T, path string, content []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
