package scanner

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestDiscoverImagesFindsSupportedImagesRecursively(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "Vehicle A", "Left.PNG"))
	writeFile(t, filepath.Join(root, "Vehicle A", "notes.txt"))
	writeFile(t, filepath.Join(root, "Vehicle B", "Front.webp"))
	writeFile(t, filepath.Join(root, "Vehicle B", "nested", "Rear.JpEg"))

	images, err := DiscoverImages(root)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := relPaths(images)
	want := []string{
		"Vehicle A/Left.PNG",
		"Vehicle B/Front.webp",
		"Vehicle B/nested/Rear.JpEg",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected paths %#v, got %#v", want, got)
	}

	if images[0].Ext != ".png" {
		t.Fatalf("expected lowercase extension, got %q", images[0].Ext)
	}
	if images[0].Size == 0 {
		t.Fatal("expected image size to be recorded")
	}
}

func TestDiscoverImagesReturnsDeterministicOrder(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "z.png"))
	writeFile(t, filepath.Join(root, "A.png"))
	writeFile(t, filepath.Join(root, "m.png"))

	images, err := DiscoverImages(root)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	got := relPaths(images)
	want := []string{"A.png", "m.png", "z.png"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected paths %#v, got %#v", want, got)
	}
}

func TestDiscoverImagesRejectsMissingRoot(t *testing.T) {
	_, err := DiscoverImages(filepath.Join(t.TempDir(), "missing"))

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDiscoverImagesRejectsFileRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "image.png")
	writeFile(t, path)

	_, err := DiscoverImages(path)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRequireImagesRejectsEmptyDirectory(t *testing.T) {
	_, err := RequireImages(t.TempDir())

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestIsSupportedImage(t *testing.T) {
	tests := map[string]bool{
		"left.png":    true,
		"left.PNG":    true,
		"right.jpg":   true,
		"right.jpeg":  true,
		"front.bmp":   true,
		"back.tga":    true,
		"top.webp":    true,
		"notes.txt":   false,
		"noextension": false,
	}

	for path, want := range tests {
		if got := IsSupportedImage(path); got != want {
			t.Fatalf("IsSupportedImage(%q) = %v, want %v", path, got, want)
		}
	}
}

func TestSupportedExtensionsReturnsSortedExtensions(t *testing.T) {
	got := SupportedExtensions()
	want := []string{".bmp", ".jpeg", ".jpg", ".png", ".tga", ".webp"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("expected extensions %#v, got %#v", want, got)
	}
}

func relPaths(images []ImageFile) []string {
	paths := make([]string, 0, len(images))
	for _, image := range images {
		paths = append(paths, image.RelPath)
	}
	return paths
}

func writeFile(t *testing.T, path string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create test dir: %v", err)
	}
	if err := os.WriteFile(path, []byte("image"), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
