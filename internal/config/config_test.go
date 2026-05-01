package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestLoadForDirReturnsDefaultsWhenConfigDoesNotExist(t *testing.T) {
	dir := t.TempDir()

	cfg, path, err := LoadForDir(dir)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if path != filepath.Join(dir, FileName) {
		t.Fatalf("expected config path in dir, got %q", path)
	}
	if !reflect.DeepEqual(cfg, Defaults()) {
		t.Fatalf("expected defaults, got %#v", cfg)
	}
}

func TestLoadFileMergesWithDefaults(t *testing.T) {
	path := writeConfig(t, `{
		"templatesDir": "./templates",
		"skipNamePatterns": ["*_raw.png"],
		"concurrency": 5
	}`)

	cfg, err := LoadFile(path)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.AssetType != AssetTypeImage {
		t.Fatalf("expected default asset type, got %q", cfg.AssetType)
	}
	if cfg.Creator.Type != CreatorTypeUser {
		t.Fatalf("expected default creator type, got %q", cfg.Creator.Type)
	}
	if cfg.TemplatesDir != "./templates" {
		t.Fatalf("expected templates dir from config, got %q", cfg.TemplatesDir)
	}
	if cfg.OutputFile != "IDs.txt" {
		t.Fatalf("expected default output file, got %q", cfg.OutputFile)
	}
	if cfg.Concurrency != 5 {
		t.Fatalf("expected config concurrency, got %d", cfg.Concurrency)
	}
}

func TestLoadFileSupportsGroupCreator(t *testing.T) {
	path := writeConfig(t, `{
		"creator": {
			"type": "group",
			"groupId": 123456
		}
	}`)

	cfg, err := LoadFile(path)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cfg.Creator.GroupID == nil || *cfg.Creator.GroupID != 123456 {
		t.Fatalf("expected group ID 123456, got %#v", cfg.Creator.GroupID)
	}
}

func TestLoadFileRejectsInvalidConfig(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		message string
	}{
		{
			name:    "unsupported asset type",
			json:    `{"assetType":"Mesh"}`,
			message: "assetType",
		},
		{
			name:    "missing group id",
			json:    `{"creator":{"type":"group"}}`,
			message: "groupId",
		},
		{
			name:    "user creator with group id",
			json:    `{"creator":{"type":"user","groupId":123}}`,
			message: "groupId",
		},
		{
			name:    "empty output file",
			json:    `{"outputFile":""}`,
			message: "outputFile",
		},
		{
			name:    "invalid concurrency",
			json:    `{"concurrency":0}`,
			message: "concurrency",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			path := writeConfig(t, test.json)

			_, err := LoadFile(path)

			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("expected error containing %q, got %v", test.message, err)
			}
		})
	}
}

func TestLoadFileRejectsMalformedJSON(t *testing.T) {
	path := writeConfig(t, `{`)

	_, err := LoadFile(path)

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "read config") {
		t.Fatalf("expected contextual error, got %v", err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, FileName)

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write test config: %v", err)
	}

	return path
}
