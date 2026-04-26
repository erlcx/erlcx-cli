package lockfile

import (
	"encoding/json"
	"os"
	"path/filepath"
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

func TestLoadReadsAndValidatesLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".erlcx-upload.lock.json")
	lock := validLockFile()
	writeJSON(t, path, lock)

	loaded, err := Load(path)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if loaded.Creator.ID != lock.Creator.ID {
		t.Fatalf("expected creator ID %q, got %q", lock.Creator.ID, loaded.Creator.ID)
	}
	if len(loaded.Files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(loaded.Files))
	}
}

func TestLoadRejectsMalformedJSON(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".erlcx-upload.lock.json")
	writeText(t, path, `{`)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read lock file") {
		t.Fatalf("expected contextual error, got %v", err)
	}
}

func TestLoadRejectsInvalidLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".erlcx-upload.lock.json")
	lock := validLockFile()
	lock.Files["Left.png"] = Entry{}
	writeJSON(t, path, lock)

	_, err := Load(path)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Left.png") {
		t.Fatalf("expected invalid file context, got %v", err)
	}
}

func TestLoadOrNewReturnsNewLockWhenMissing(t *testing.T) {
	creator := Creator{
		Type: CreatorTypeUser,
		ID:   "123456",
	}

	lock, err := LoadOrNew(filepath.Join(t.TempDir(), ".erlcx-upload.lock.json"), creator)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if lock.Version != CurrentVersion {
		t.Fatalf("expected current version, got %d", lock.Version)
	}
	if lock.Creator != creator {
		t.Fatalf("expected creator %#v, got %#v", creator, lock.Creator)
	}
}

func TestSaveWritesPrettyJSONAtomically(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", ".erlcx-upload.lock.json")
	lock := validLockFile()

	if err := Save(path, lock); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read saved lock: %v", err)
	}
	if !strings.HasSuffix(string(data), "\n") {
		t.Fatalf("expected trailing newline, got %q", string(data))
	}
	if !strings.Contains(string(data), "\n  \"version\"") {
		t.Fatalf("expected indented JSON, got %q", string(data))
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("load saved lock: %v", err)
	}
	if loaded.Creator.ID != lock.Creator.ID {
		t.Fatalf("expected creator ID %q, got %q", lock.Creator.ID, loaded.Creator.ID)
	}
}

func TestSaveRejectsInvalidLockFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".erlcx-upload.lock.json")

	err := Save(path, LockFile{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "write lock file") {
		t.Fatalf("expected contextual error, got %v", err)
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("expected no lock file to be written, stat error was %v", statErr)
	}
}

func validLockFile() LockFile {
	lock := New(Creator{
		Type: CreatorTypeUser,
		ID:   "123456",
	})
	lock.Files["Law Enforcement/Falcon Stallion 350 2015/Left.png"] = validEntry()
	return lock
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

func writeJSON(t *testing.T, path string, value any) {
	t.Helper()

	data, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal test JSON: %v", err)
	}
	writeText(t, path, string(data))
}

func writeText(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test file: %v", err)
	}
}
