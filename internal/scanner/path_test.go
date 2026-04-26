package scanner

import (
	"path/filepath"
	"testing"
)

func TestNormalizeRelativePathUsesForwardSlashes(t *testing.T) {
	root := t.TempDir()
	filePath := filepath.Join(root, "Law Enforcement", "Falcon Stallion 350 2015", "Left.png")

	got, err := NormalizeRelativePath(root, filePath)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	want := "Law Enforcement/Falcon Stallion 350 2015/Left.png"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestNormalizeRelativePathRejectsOutsideRoot(t *testing.T) {
	root := t.TempDir()
	other := filepath.Join(t.TempDir(), "Left.png")

	_, err := NormalizeRelativePath(root, other)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestNormalizeRelativePathRejectsRootItself(t *testing.T) {
	root := t.TempDir()

	_, err := NormalizeRelativePath(root, root)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestCleanRelativePath(t *testing.T) {
	tests := map[string]string{
		`Vehicle\Left.png`:             "Vehicle/Left.png",
		"Vehicle/./Left.png":           "Vehicle/Left.png",
		"Vehicle/Body/../Left.png":     "Vehicle/Left.png",
		"./Vehicle/Left.png":           "Vehicle/Left.png",
		".":                            "",
		"Law Enforcement/Vehicle.webp": "Law Enforcement/Vehicle.webp",
	}

	for input, want := range tests {
		if got := CleanRelativePath(input); got != want {
			t.Fatalf("CleanRelativePath(%q) = %q, want %q", input, got, want)
		}
	}
}
