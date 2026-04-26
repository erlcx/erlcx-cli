package lockfile

import (
	"errors"
	"fmt"
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
