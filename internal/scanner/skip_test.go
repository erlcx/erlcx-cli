package scanner

import "testing"

func TestNewSkipMatcherHasNoBuiltInPatterns(t *testing.T) {
	matcher, err := NewSkipMatcher(nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !matcher.Empty() {
		t.Fatal("expected matcher to be empty")
	}
	if pattern, ok := matcher.Match("Vehicle/template.png", "template.png"); ok {
		t.Fatalf("expected no built-in template skip, matched %q", pattern)
	}
}

func TestSkipMatcherMatchesConfiguredFileNamePattern(t *testing.T) {
	matcher, err := NewSkipMatcher([]string{"*_raw.png"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	pattern, ok := matcher.Match("Vehicle/Left_raw.png", "Left_raw.png")

	if !ok {
		t.Fatal("expected configured skip match")
	}
	if pattern != "*_raw.png" {
		t.Fatalf("expected pattern *_raw.png, got %q", pattern)
	}
}

func TestSkipMatcherMatchesConfiguredRelativePathPattern(t *testing.T) {
	matcher, err := NewSkipMatcher([]string{"*/raw/*.png"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, ok := matcher.Match("Vehicle/raw/Left.png", "Left.png")

	if !ok {
		t.Fatal("expected relative path skip match")
	}
}

func TestSkipMatcherNormalizesWindowsStylePatterns(t *testing.T) {
	matcher, err := NewSkipMatcher([]string{`Vehicle\raw\*.png`})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, ok := matcher.Match("Vehicle/raw/Left.png", "Left.png")

	if !ok {
		t.Fatal("expected normalized pattern to match")
	}
}

func TestSkipMatcherIgnoresEmptyPatterns(t *testing.T) {
	matcher, err := NewSkipMatcher([]string{"", "   "})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !matcher.Empty() {
		t.Fatal("expected empty patterns to be ignored")
	}
}

func TestNewSkipMatcherRejectsInvalidPattern(t *testing.T) {
	_, err := NewSkipMatcher([]string{"["})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSkipMatcherMatchImage(t *testing.T) {
	matcher, err := NewSkipMatcher([]string{"*_reference.png"})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	_, ok := matcher.MatchImage(ImageFile{
		RelPath: "Vehicle/Top_reference.png",
		Name:    "Top_reference.png",
	})

	if !ok {
		t.Fatal("expected image match")
	}
}
