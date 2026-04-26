package scanner

import (
	"fmt"
	"sort"
)

type TemplateIndex struct {
	byHash map[string][]ImageFile
}

func BuildTemplateIndex(root string) (TemplateIndex, error) {
	images, err := DiscoverImages(root)
	if err != nil {
		return TemplateIndex{}, err
	}

	hashed, err := HashImageFiles(images)
	if err != nil {
		return TemplateIndex{}, fmt.Errorf("index templates %s: %w", root, err)
	}

	index := TemplateIndex{
		byHash: map[string][]ImageFile{},
	}
	for _, image := range hashed {
		index.byHash[image.SHA256] = append(index.byHash[image.SHA256], image)
	}

	for hash := range index.byHash {
		sort.Slice(index.byHash[hash], func(i, j int) bool {
			return index.byHash[hash][i].RelPath < index.byHash[hash][j].RelPath
		})
	}

	return index, nil
}

func (index TemplateIndex) MatchSHA256(sha256 string) (ImageFile, bool) {
	if index.byHash == nil {
		return ImageFile{}, false
	}

	matches := index.byHash[sha256]
	if len(matches) == 0 {
		return ImageFile{}, false
	}

	return matches[0], true
}

func (index TemplateIndex) MatchImage(image ImageFile) (ImageFile, bool) {
	if image.SHA256 == "" {
		return ImageFile{}, false
	}
	return index.MatchSHA256(image.SHA256)
}

func (index TemplateIndex) Count() int {
	total := 0
	for _, matches := range index.byHash {
		total += len(matches)
	}
	return total
}
