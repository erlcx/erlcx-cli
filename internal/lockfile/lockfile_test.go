package lockfile

import (
	"strings"
	"testing"
	"time"
)

const validSHA256 = "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"

func TestNewCreatesCurrentEmptyLockFile(t *testing.T) {
	lock := New(Creator{
		Type: CreatorTypeUser,
		ID:   "123456",
	})

	if lock.Version != CurrentVersion {
		t.Fatalf("expected current version, got %d", lock.Version)
	}
	if lock.Files == nil {
		t.Fatal("expected files map to be initialized")
	}
	if len(lock.Files) != 0 {
		t.Fatalf("expected no files, got %d", len(lock.Files))
	}
}

func TestValidateAcceptsValidLockFile(t *testing.T) {
	lock := New(Creator{
		Type: CreatorTypeGroup,
		ID:   "987654",
	})
	lock.Files["Law Enforcement/Falcon Stallion 350 2015/Left.png"] = validEntry()

	if err := Validate(lock); err != nil {
		t.Fatalf("expected valid lock file, got %v", err)
	}
}

func TestValidateRejectsInvalidLockFile(t *testing.T) {
	tests := []struct {
		name    string
		lock    LockFile
		message string
	}{
		{
			name: "unsupported version",
			lock: LockFile{
				Version: CurrentVersion + 1,
				Creator: Creator{
					Type: CreatorTypeUser,
					ID:   "123",
				},
				Files: map[string]Entry{},
			},
			message: "version",
		},
		{
			name: "invalid creator",
			lock: LockFile{
				Version: CurrentVersion,
				Creator: Creator{
					Type: "team",
					ID:   "123",
				},
				Files: map[string]Entry{},
			},
			message: "creator",
		},
		{
			name: "nil files",
			lock: LockFile{
				Version: CurrentVersion,
				Creator: Creator{
					Type: CreatorTypeUser,
					ID:   "123",
				},
			},
			message: "files",
		},
		{
			name: "empty path",
			lock: LockFile{
				Version: CurrentVersion,
				Creator: Creator{
					Type: CreatorTypeUser,
					ID:   "123",
				},
				Files: map[string]Entry{
					"": validEntry(),
				},
			},
			message: "path",
		},
		{
			name: "invalid entry",
			lock: LockFile{
				Version: CurrentVersion,
				Creator: Creator{
					Type: CreatorTypeUser,
					ID:   "123",
				},
				Files: map[string]Entry{
					"Left.png": {},
				},
			},
			message: "Left.png",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := Validate(test.lock)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error containing %q, got %v", test.message, err)
			}
		})
	}
}

func TestValidateCreatorRejectsInvalidCreator(t *testing.T) {
	tests := []Creator{
		{Type: "invalid", ID: "123"},
		{Type: CreatorTypeUser},
		{Type: CreatorTypeGroup},
	}

	for _, creator := range tests {
		if err := ValidateCreator(creator); err == nil {
			t.Fatalf("expected invalid creator %#v", creator)
		}
	}
}

func TestValidateEntryRejectsInvalidEntry(t *testing.T) {
	tests := []struct {
		name    string
		entry   Entry
		message string
	}{
		{
			name:    "bad sha",
			entry:   withValidEntry(func(entry *Entry) { entry.SHA256 = "ABC" }),
			message: "sha256",
		},
		{
			name:    "bad asset type",
			entry:   withValidEntry(func(entry *Entry) { entry.AssetType = "Image" }),
			message: "assetType",
		},
		{
			name:    "missing asset id",
			entry:   withValidEntry(func(entry *Entry) { entry.AssetID = "" }),
			message: "assetId",
		},
		{
			name:    "missing display name",
			entry:   withValidEntry(func(entry *Entry) { entry.DisplayName = "" }),
			message: "displayName",
		},
		{
			name:    "missing uploaded at",
			entry:   withValidEntry(func(entry *Entry) { entry.UploadedAt = time.Time{} }),
			message: "uploadedAt",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := ValidateEntry(test.entry)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error containing %q, got %v", test.message, err)
			}
		})
	}
}

func TestEntryMatchesContent(t *testing.T) {
	entry := validEntry()

	if !entry.MatchesContent(validSHA256, AssetTypeDecal) {
		t.Fatal("expected entry to match same content and asset type")
	}
	if entry.MatchesContent(strings.Repeat("0", 64), AssetTypeDecal) {
		t.Fatal("expected different hash not to match")
	}
	if entry.MatchesContent(validSHA256, "Image") {
		t.Fatal("expected different asset type not to match")
	}

	entry.AssetID = ""
	if entry.MatchesContent(validSHA256, AssetTypeDecal) {
		t.Fatal("expected missing asset ID not to match")
	}
}

func validEntry() Entry {
	return Entry{
		SHA256:      validSHA256,
		AssetType:   AssetTypeDecal,
		AssetID:     "2205400862",
		DisplayName: "Falcon Stallion 350 2015 - Left",
		UploadedAt:  time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
	}
}

func withValidEntry(edit func(*Entry)) Entry {
	entry := validEntry()
	edit(&entry)
	return entry
}
