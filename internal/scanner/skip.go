package scanner

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
)

type SkipMatcher struct {
	patterns []string
}

func NewSkipMatcher(patterns []string) (SkipMatcher, error) {
	matcher := SkipMatcher{
		patterns: make([]string, 0, len(patterns)),
	}

	for _, pattern := range patterns {
		normalized := strings.TrimSpace(pattern)
		if normalized == "" {
			continue
		}
		normalized = filepath.ToSlash(normalized)

		if _, err := path.Match(normalized, "test"); err != nil {
			return SkipMatcher{}, fmt.Errorf("skip pattern %q: %w", pattern, err)
		}

		matcher.patterns = append(matcher.patterns, normalized)
	}

	return matcher, nil
}

func (matcher SkipMatcher) MatchImage(image ImageFile) (string, bool) {
	return matcher.Match(image.RelPath, image.Name)
}

func (matcher SkipMatcher) Match(relPath string, name string) (string, bool) {
	for _, pattern := range matcher.patterns {
		if matchesPattern(pattern, relPath) || matchesPattern(pattern, name) {
			return pattern, true
		}
	}

	return "", false
}

func (matcher SkipMatcher) Empty() bool {
	return len(matcher.patterns) == 0
}

func matchesPattern(pattern string, value string) bool {
	if value == "" {
		return false
	}

	normalized := filepath.ToSlash(value)
	matched, err := path.Match(pattern, normalized)
	if err != nil {
		return false
	}
	return matched
}
