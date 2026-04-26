package lockfile

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"time"
)

const CurrentVersion = 1

const (
	AssetTypeDecal = "Decal"

	CreatorTypeUser  = "user"
	CreatorTypeGroup = "group"
)

type LockFile struct {
	Version int              `json:"version"`
	Creator Creator          `json:"creator"`
	Files   map[string]Entry `json:"files"`
}

type Creator struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type Entry struct {
	SHA256      string    `json:"sha256"`
	AssetType   string    `json:"assetType"`
	AssetID     string    `json:"assetId"`
	DisplayName string    `json:"displayName"`
	UploadedAt  time.Time `json:"uploadedAt"`
}

func New(creator Creator) LockFile {
	return LockFile{
		Version: CurrentVersion,
		Creator: creator,
		Files:   map[string]Entry{},
	}
}

func Load(path string) (LockFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LockFile{}, err
	}

	var lock LockFile
	if err := json.Unmarshal(data, &lock); err != nil {
		return LockFile{}, fmt.Errorf("read lock file %s: %w", path, err)
	}

	if err := Validate(lock); err != nil {
		return LockFile{}, fmt.Errorf("read lock file %s: %w", path, err)
	}

	return lock, nil
}

func LoadOrNew(path string, creator Creator) (LockFile, error) {
	lock, err := Load(path)
	if errors.Is(err, os.ErrNotExist) {
		return New(creator), nil
	}
	return lock, err
}

func Save(path string, lock LockFile) error {
	if err := Validate(lock); err != nil {
		return fmt.Errorf("write lock file %s: %w", path, err)
	}

	data, err := json.MarshalIndent(lock, "", "  ")
	if err != nil {
		return fmt.Errorf("write lock file %s: %w", path, err)
	}
	data = append(data, '\n')

	dir := filepath.Dir(path)
	if dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("write lock file %s: %w", path, err)
		}
	}

	tmp, err := os.CreateTemp(dir, filepath.Base(path)+".tmp-*")
	if err != nil {
		return fmt.Errorf("write lock file %s: %w", path, err)
	}
	tmpPath := tmp.Name()
	removeTemp := true
	defer func() {
		if removeTemp {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write lock file %s: %w", path, err)
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write lock file %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("write lock file %s: %w", path, err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("write lock file %s: %w", path, err)
	}
	removeTemp = false

	return nil
}

func Validate(lock LockFile) error {
	if lock.Version != CurrentVersion {
		return fmt.Errorf("unsupported lock file version %d", lock.Version)
	}
	if err := ValidateCreator(lock.Creator); err != nil {
		return fmt.Errorf("creator: %w", err)
	}
	if lock.Files == nil {
		return errors.New("files must not be null")
	}

	for path, entry := range lock.Files {
		if path == "" {
			return errors.New("file path must not be empty")
		}
		if err := ValidateEntry(entry); err != nil {
			return fmt.Errorf("file %q: %w", path, err)
		}
	}

	return nil
}

func ValidateCreator(creator Creator) error {
	switch creator.Type {
	case CreatorTypeUser, CreatorTypeGroup:
	default:
		return fmt.Errorf("type must be %q or %q", CreatorTypeUser, CreatorTypeGroup)
	}
	if creator.ID == "" {
		return errors.New("id must not be empty")
	}

	return nil
}

func ValidateEntry(entry Entry) error {
	if !isSHA256(entry.SHA256) {
		return errors.New("sha256 must be a 64-character lowercase hex string")
	}
	if entry.AssetType != AssetTypeDecal {
		return fmt.Errorf("assetType must be %q", AssetTypeDecal)
	}
	if entry.AssetID == "" {
		return errors.New("assetId must not be empty")
	}
	if entry.DisplayName == "" {
		return errors.New("displayName must not be empty")
	}
	if entry.UploadedAt.IsZero() {
		return errors.New("uploadedAt must not be empty")
	}

	return nil
}

func (entry Entry) MatchesContent(sha256 string, assetType string) bool {
	return entry.SHA256 == sha256 && entry.AssetType == assetType && entry.AssetID != ""
}

var sha256Pattern = regexp.MustCompile(`^[a-f0-9]{64}$`)

func isSHA256(value string) bool {
	return sha256Pattern.MatchString(value)
}
