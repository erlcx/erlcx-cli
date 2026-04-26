package scanner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSHA256File(t *testing.T) {
	path := filepath.Join(t.TempDir(), "image.png")
	if err := os.WriteFile(path, []byte("abc"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	got, err := SHA256File(path)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestSHA256FileReturnsContextualError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.png")

	_, err := SHA256File(path)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestHashImageFilesAddsHashesWithoutMutatingInput(t *testing.T) {
	root := t.TempDir()
	first := filepath.Join(root, "first.png")
	second := filepath.Join(root, "second.png")
	if err := os.WriteFile(first, []byte("first"), 0o600); err != nil {
		t.Fatalf("write first file: %v", err)
	}
	if err := os.WriteFile(second, []byte("second"), 0o600); err != nil {
		t.Fatalf("write second file: %v", err)
	}

	images := []ImageFile{
		{AbsPath: first, RelPath: "first.png"},
		{AbsPath: second, RelPath: "second.png"},
	}

	hashed, err := HashImageFiles(images)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if images[0].SHA256 != "" {
		t.Fatalf("expected original input to remain unchanged, got %q", images[0].SHA256)
	}
	if hashed[0].SHA256 == "" {
		t.Fatal("expected first image hash")
	}
	if hashed[1].SHA256 == "" {
		t.Fatal("expected second image hash")
	}
	if hashed[0].SHA256 == hashed[1].SHA256 {
		t.Fatal("expected different hashes for different files")
	}
}
