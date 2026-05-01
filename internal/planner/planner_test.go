package planner

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/erlcx/cli/internal/config"
	"github.com/erlcx/cli/internal/lockfile"
	"github.com/erlcx/cli/internal/scanner"
)

func TestBuildScanPlanClassifiesImages(t *testing.T) {
	packDir := t.TempDir()
	templatesDir := filepath.Join(packDir, "templates")

	writeBytes(t, filepath.Join(packDir, "Vehicle", "upload.png"), []byte("upload"))
	writeBytes(t, filepath.Join(packDir, "Vehicle", "unchanged.png"), []byte("unchanged"))
	writeBytes(t, filepath.Join(packDir, "Vehicle", "template.png"), []byte("template"))
	writeBytes(t, filepath.Join(packDir, "Vehicle", "skip_raw.png"), []byte("skip"))
	writeBytes(t, filepath.Join(templatesDir, "template.png"), []byte("template"))

	unchangedHash := hashFileForTest(t, filepath.Join(packDir, "Vehicle", "unchanged.png"))
	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeGroup, ID: "123456"})
	lock.Files["Vehicle/unchanged.png"] = lockfile.Entry{
		SHA256:      unchangedHash,
		AssetType:   lockfile.AssetTypeImage,
		AssetID:     "999",
		DisplayName: "Vehicle - unchanged",
		UploadedAt:  time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
	}

	groupID := int64(123456)
	cfg := config.Defaults()
	cfg.Creator.Type = config.CreatorTypeGroup
	cfg.Creator.GroupID = &groupID
	cfg.TemplatesDir = "templates"
	cfg.SkipNamePatterns = []string{"*_raw.png"}

	plan, err := BuildScanPlan(packDir, cfg, &lock)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if plan.Counts.Total != 5 {
		t.Fatalf("expected 5 total items, got %d", plan.Counts.Total)
	}
	if plan.Counts.Upload != 1 {
		t.Fatalf("expected 1 upload, got %d", plan.Counts.Upload)
	}
	if plan.Counts.Unchanged != 1 {
		t.Fatalf("expected 1 unchanged, got %d", plan.Counts.Unchanged)
	}
	if plan.Counts.TemplateMatch != 2 {
		t.Fatalf("expected 2 template matches including template directory image, got %d", plan.Counts.TemplateMatch)
	}
	if plan.Counts.ConfiguredSkip != 1 {
		t.Fatalf("expected 1 configured skip, got %d", plan.Counts.ConfiguredSkip)
	}

	assertClass(t, plan, "Vehicle/upload.png", ClassUpload)
	assertClass(t, plan, "Vehicle/unchanged.png", ClassUnchanged)
	assertClass(t, plan, "Vehicle/template.png", ClassTemplateMatch)
	assertClass(t, plan, "Vehicle/skip_raw.png", ClassConfiguredSkip)
}

func TestBuildScanPlanDerivesNamesFromRelativePath(t *testing.T) {
	packDir := t.TempDir()
	writeBytes(t, filepath.Join(packDir, "Law Enforcement", "Coupe - Sedan", "Falcon Stallion 350 2015", "Left.png"), []byte("image"))

	plan, err := BuildScanPlan(packDir, config.Defaults(), nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(plan.Items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(plan.Items))
	}

	item := plan.Items[0]
	if item.VehiclePath != "Law Enforcement/Coupe - Sedan/Falcon Stallion 350 2015" {
		t.Fatalf("expected vehicle path to be derived, got %q", item.VehiclePath)
	}
	if item.VehicleName != "Falcon Stallion 350 2015" {
		t.Fatalf("expected vehicle name to be derived, got %q", item.VehicleName)
	}
	if item.ImageName != "Left" {
		t.Fatalf("expected image name to be derived, got %q", item.ImageName)
	}
	if item.DisplayName != "Falcon Stallion 350 2015 - Left" {
		t.Fatalf("expected display name to be derived, got %q", item.DisplayName)
	}
}

func TestBuildScanPlanRejectsImageWithoutVehicleFolder(t *testing.T) {
	packDir := t.TempDir()
	writeBytes(t, filepath.Join(packDir, "Left.png"), []byte("image"))

	_, err := BuildScanPlan(packDir, config.Defaults(), nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildScanPlanConfiguredSkipWinsOverTemplateMatch(t *testing.T) {
	packDir := t.TempDir()
	templatesDir := filepath.Join(packDir, "templates")
	writeBytes(t, filepath.Join(packDir, "Vehicle", "raw.png"), []byte("same"))
	writeBytes(t, filepath.Join(templatesDir, "raw.png"), []byte("same"))

	cfg := config.Defaults()
	cfg.TemplatesDir = "templates"
	cfg.SkipNamePatterns = []string{"raw.png"}

	plan, err := BuildScanPlan(packDir, cfg, nil)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertClass(t, plan, "Vehicle/raw.png", ClassConfiguredSkip)
}

func TestBuildScanPlanDoesNotUseLockForDifferentCreator(t *testing.T) {
	packDir := t.TempDir()
	imagePath := filepath.Join(packDir, "Vehicle", "Left.png")
	writeBytes(t, imagePath, []byte("same"))

	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeGroup, ID: "123456"})
	lock.Files["Vehicle/Left.png"] = lockfile.Entry{
		SHA256:      hashFileForTest(t, imagePath),
		AssetType:   lockfile.AssetTypeImage,
		AssetID:     "999",
		DisplayName: "Vehicle - Left",
		UploadedAt:  time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
	}

	cfg := config.Defaults()
	cfg.Creator.Type = config.CreatorTypeUser

	plan, err := BuildScanPlan(packDir, cfg, &lock)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertClass(t, plan, "Vehicle/Left.png", ClassUpload)
}

func TestBuildScanPlanForCreatorUsesResolvedUserCreator(t *testing.T) {
	packDir := t.TempDir()
	imagePath := filepath.Join(packDir, "Vehicle", "Left.png")
	writeBytes(t, imagePath, []byte("same"))

	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "456"})
	lock.Files["Vehicle/Left.png"] = lockfile.Entry{
		SHA256:      hashFileForTest(t, imagePath),
		AssetType:   lockfile.AssetTypeImage,
		AssetID:     "999",
		DisplayName: "Vehicle - Left",
		UploadedAt:  time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
	}

	plan, err := BuildScanPlanForCreator(
		packDir,
		config.Defaults(),
		lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "456"},
		&lock,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertClass(t, plan, "Vehicle/Left.png", ClassUnchanged)
}

func TestBuildScanPlanForCreatorDoesNotUseLockForDifferentResolvedUserCreator(t *testing.T) {
	packDir := t.TempDir()
	imagePath := filepath.Join(packDir, "Vehicle", "Left.png")
	writeBytes(t, imagePath, []byte("same"))

	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "456"})
	lock.Files["Vehicle/Left.png"] = lockfile.Entry{
		SHA256:      hashFileForTest(t, imagePath),
		AssetType:   lockfile.AssetTypeImage,
		AssetID:     "999",
		DisplayName: "Vehicle - Left",
		UploadedAt:  time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
	}

	plan, err := BuildScanPlanForCreator(
		packDir,
		config.Defaults(),
		lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "789"},
		&lock,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	assertClass(t, plan, "Vehicle/Left.png", ClassUpload)
}

func TestBuildScanPlanRejectsInvalidSkipPattern(t *testing.T) {
	packDir := t.TempDir()
	writeBytes(t, filepath.Join(packDir, "Vehicle", "Left.png"), []byte("image"))

	cfg := config.Defaults()
	cfg.SkipNamePatterns = []string{"["}

	_, err := BuildScanPlan(packDir, cfg, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildScanPlanRejectsMissingTemplatesDir(t *testing.T) {
	packDir := t.TempDir()
	writeBytes(t, filepath.Join(packDir, "Vehicle", "Left.png"), []byte("image"))

	cfg := config.Defaults()
	cfg.TemplatesDir = "missing"

	_, err := BuildScanPlan(packDir, cfg, nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestBuildScanPlanRejectsEmptyPack(t *testing.T) {
	_, err := BuildScanPlan(t.TempDir(), config.Defaults(), nil)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func assertClass(t *testing.T, plan Plan, relPath string, class Classification) {
	t.Helper()

	for _, item := range plan.Items {
		if item.Image.RelPath == relPath {
			if item.Class != class {
				t.Fatalf("expected %s to be %s, got %s", relPath, class, item.Class)
			}
			return
		}
	}

	t.Fatalf("did not find plan item %s", relPath)
}

func hashFileForTest(t *testing.T, path string) string {
	t.Helper()

	got, err := scanner.SHA256File(path)
	if err != nil {
		t.Fatalf("hash test file: %v", err)
	}
	return got
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
