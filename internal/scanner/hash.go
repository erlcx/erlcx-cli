package scanner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
)

func SHA256File(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", fmt.Errorf("hash %s: %w", path, err)
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

func HashImageFiles(images []ImageFile) ([]ImageFile, error) {
	hashed := make([]ImageFile, len(images))
	for i, image := range images {
		sha256, err := SHA256File(image.AbsPath)
		if err != nil {
			return nil, err
		}

		image.SHA256 = sha256
		hashed[i] = image
	}

	return hashed, nil
}
