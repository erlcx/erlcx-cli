package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

const FileName = ".erlcx-uploader.json"

const (
	AssetTypeDecal = "Decal"
	AssetTypeImage = "Image"

	CreatorTypeUser  = "user"
	CreatorTypeGroup = "group"
)

type Config struct {
	AssetType        string   `json:"assetType"`
	Creator          Creator  `json:"creator"`
	TemplatesDir     string   `json:"templatesDir"`
	SkipNamePatterns []string `json:"skipNamePatterns"`
	OutputFile       string   `json:"outputFile"`
	LockFile         string   `json:"lockFile"`
	Concurrency      int      `json:"concurrency"`
}

type Creator struct {
	Type    string `json:"type"`
	GroupID *int64 `json:"groupId"`
}

func Defaults() Config {
	return Config{
		AssetType: AssetTypeImage,
		Creator: Creator{
			Type: CreatorTypeUser,
		},
		SkipNamePatterns: []string{},
		OutputFile:       "IDs.txt",
		LockFile:         ".erlcx-upload.lock.json",
		Concurrency:      3,
	}
}

func LoadForDir(dir string) (Config, string, error) {
	if dir == "" {
		dir = "."
	}

	path := filepath.Join(dir, FileName)
	cfg, err := LoadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return Defaults(), path, nil
	}

	return cfg, path, err
}

func LoadFile(path string) (Config, error) {
	cfg := Defaults()

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, err
	}

	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	if cfg.SkipNamePatterns == nil {
		cfg.SkipNamePatterns = []string{}
	}

	if err := Validate(cfg); err != nil {
		return Config{}, fmt.Errorf("read config %s: %w", path, err)
	}

	return cfg, nil
}

func Validate(cfg Config) error {
	switch cfg.AssetType {
	case AssetTypeImage, AssetTypeDecal:
	default:
		return fmt.Errorf("assetType must be %q or %q", AssetTypeImage, AssetTypeDecal)
	}

	switch cfg.Creator.Type {
	case CreatorTypeUser:
		if cfg.Creator.GroupID != nil {
			return errors.New("creator.groupId must be omitted when creator.type is user")
		}
	case CreatorTypeGroup:
		if cfg.Creator.GroupID == nil || *cfg.Creator.GroupID <= 0 {
			return errors.New("creator.groupId must be a positive number when creator.type is group")
		}
	default:
		return fmt.Errorf("creator.type must be %q or %q", CreatorTypeUser, CreatorTypeGroup)
	}

	if cfg.OutputFile == "" {
		return errors.New("outputFile must not be empty")
	}
	if cfg.LockFile == "" {
		return errors.New("lockFile must not be empty")
	}
	if cfg.Concurrency < 1 {
		return errors.New("concurrency must be at least 1")
	}

	return nil
}
