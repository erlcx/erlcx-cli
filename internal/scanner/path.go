package scanner

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

func NormalizeRelativePath(root string, filePath string) (string, error) {
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("normalize path %s: %w", filePath, err)
	}

	absPath, err := filepath.Abs(filePath)
	if err != nil {
		return "", fmt.Errorf("normalize path %s: %w", filePath, err)
	}

	relPath, err := filepath.Rel(absRoot, absPath)
	if err != nil {
		return "", fmt.Errorf("normalize path %s: %w", filePath, err)
	}
	if relPath == "." {
		return "", fmt.Errorf("normalize path %s: file path must be inside root and not equal to root", filePath)
	}
	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("normalize path %s: file path is outside root", filePath)
	}

	return CleanRelativePath(relPath), nil
}

func CleanRelativePath(relPath string) string {
	relPath = filepath.ToSlash(relPath)
	relPath = path.Clean(relPath)
	if relPath == "." {
		return ""
	}
	return relPath
}
