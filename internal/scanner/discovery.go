package scanner

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var supportedExtensions = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".bmp":  true,
	".tga":  true,
	".webp": true,
}

type ImageFile struct {
	AbsPath string
	RelPath string
	Name    string
	Ext     string
	Size    int64
}

func DiscoverImages(root string) ([]ImageFile, error) {
	if root == "" {
		root = "."
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}

	info, err := os.Stat(absRoot)
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("scan %s: not a directory", root)
	}

	var images []ImageFile
	err = filepath.WalkDir(absRoot, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(entry.Name()))
		if !supportedExtensions[ext] {
			return nil
		}

		info, err := entry.Info()
		if err != nil {
			return err
		}

		relPath, err := NormalizeRelativePath(absRoot, path)
		if err != nil {
			return err
		}

		images = append(images, ImageFile{
			AbsPath: path,
			RelPath: relPath,
			Name:    entry.Name(),
			Ext:     ext,
			Size:    info.Size(),
		})

		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("scan %s: %w", root, err)
	}

	sort.Slice(images, func(i, j int) bool {
		return strings.ToLower(images[i].RelPath) < strings.ToLower(images[j].RelPath)
	})

	return images, nil
}

func IsSupportedImage(path string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	return supportedExtensions[ext]
}

func SupportedExtensions() []string {
	extensions := make([]string, 0, len(supportedExtensions))
	for ext := range supportedExtensions {
		extensions = append(extensions, ext)
	}
	sort.Strings(extensions)
	return extensions
}

func HasImages(root string) (bool, error) {
	images, err := DiscoverImages(root)
	if err != nil {
		return false, err
	}
	return len(images) > 0, nil
}

func RequireImages(root string) ([]ImageFile, error) {
	images, err := DiscoverImages(root)
	if err != nil {
		return nil, err
	}
	if len(images) == 0 {
		return nil, errors.New("no supported image files found")
	}
	return images, nil
}
