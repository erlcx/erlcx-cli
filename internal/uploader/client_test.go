package uploader

import (
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCreateAssetRequestBuildsMultipartRobloxRequest(t *testing.T) {
	filePath := writeUploadFile(t, "Left.png", []byte("image"))
	client := Client{BaseURL: "https://example.test"}

	req, err := client.NewCreateAssetRequest(context.Background(), "token", AssetUploadRequest{
		FilePath:    filePath,
		DisplayName: "Vehicle - Left",
		AssetType:   "Decal",
		Creator:     Creator{Type: "group", ID: "123456"},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if req.Method != http.MethodPost {
		t.Fatalf("expected POST, got %s", req.Method)
	}
	if req.URL.String() != "https://example.test/assets/v1/assets" {
		t.Fatalf("expected assets URL, got %s", req.URL.String())
	}
	if req.Header.Get("Authorization") != "Bearer token" {
		t.Fatalf("expected bearer token, got %q", req.Header.Get("Authorization"))
	}

	reader, err := req.MultipartReader()
	if err != nil {
		t.Fatalf("read multipart body: %v", err)
	}

	parts := readMultipartParts(t, reader)
	var payload createAssetPayload
	if err := json.Unmarshal([]byte(parts["request"]), &payload); err != nil {
		t.Fatalf("decode request payload: %v", err)
	}
	if payload.AssetType != "ASSET_TYPE_DECAL" {
		t.Fatalf("expected decal asset type, got %q", payload.AssetType)
	}
	if payload.DisplayName != "Vehicle - Left" {
		t.Fatalf("expected display name, got %q", payload.DisplayName)
	}
	if payload.CreationContext.Creator.GroupID == nil || *payload.CreationContext.Creator.GroupID != 123456 {
		t.Fatalf("expected group creator, got %#v", payload.CreationContext.Creator)
	}
	if parts["fileContent"] != "image" {
		t.Fatalf("expected file content, got %q", parts["fileContent"])
	}
}

func TestNewCreateAssetRequestRejectsUnsupportedUploadImageType(t *testing.T) {
	filePath := writeUploadFile(t, "Left.webp", []byte("image"))
	client := Client{}

	_, err := client.NewCreateAssetRequest(context.Background(), "token", AssetUploadRequest{
		FilePath:    filePath,
		DisplayName: "Vehicle - Left",
		AssetType:   "Decal",
		Creator:     Creator{Type: "user", ID: "123"},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported Roblox upload image type") {
		t.Fatalf("expected unsupported image error, got %v", err)
	}
}

func TestCreateAssetParsesOperation(t *testing.T) {
	filePath := writeUploadFile(t, "Left.png", []byte("image"))
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets/v1/assets" {
			t.Fatalf("expected asset create path, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("expected bearer token, got %q", r.Header.Get("Authorization"))
		}
		writeJSON(t, w, Operation{Path: "operations/op-1", OperationID: "op-1"})
	}))
	defer server.Close()

	operation, err := (Client{BaseURL: server.URL}).CreateAsset(context.Background(), "token", AssetUploadRequest{
		FilePath:    filePath,
		DisplayName: "Vehicle - Left",
		AssetType:   "Decal",
		Creator:     Creator{Type: "user", ID: "123"},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if operation.Path != "operations/op-1" {
		t.Fatalf("expected operation path, got %q", operation.Path)
	}
}

func readMultipartParts(t *testing.T, reader *multipart.Reader) map[string]string {
	t.Helper()

	parts := map[string]string{}
	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}
		data, err := io.ReadAll(part)
		if err != nil {
			t.Fatalf("read multipart part: %v", err)
		}
		parts[part.FormName()] = string(data)
	}
	return parts
}

func writeUploadFile(t *testing.T, name string, content []byte) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, content, 0o600); err != nil {
		t.Fatalf("write upload file: %v", err)
	}
	return path
}
