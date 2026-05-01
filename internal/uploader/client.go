package uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	DefaultBaseURL     = "https://apis.roblox.com"
	DefaultDescription = "Uploaded by ERLCX"
)

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

type AssetUploadRequest struct {
	FilePath    string
	DisplayName string
	Description string
	AssetType   string
	Creator     Creator
}

type Creator struct {
	Type string
	ID   string
}

type createAssetPayload struct {
	AssetType       string          `json:"assetType"`
	DisplayName     string          `json:"displayName"`
	Description     string          `json:"description,omitempty"`
	CreationContext creationContext `json:"creationContext"`
}

type creationContext struct {
	Creator       creatorPayload `json:"creator"`
	ExpectedPrice int            `json:"expectedPrice"`
}

type creatorPayload struct {
	UserID  *int64 `json:"userId,omitempty"`
	GroupID *int64 `json:"groupId,omitempty"`
}

func (client Client) CreateAsset(ctx context.Context, accessToken string, upload AssetUploadRequest) (Operation, error) {
	req, err := client.NewCreateAssetRequest(ctx, accessToken, upload)
	if err != nil {
		return Operation{}, err
	}

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return Operation{}, fmt.Errorf("create Roblox asset: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Operation{}, fmt.Errorf("read Roblox asset create response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Operation{}, fmt.Errorf("create Roblox asset: %s: %s", resp.Status, string(bytes.TrimSpace(body)))
	}

	var operation Operation
	if err := json.Unmarshal(body, &operation); err != nil {
		return Operation{}, fmt.Errorf("decode Roblox asset create response: %w", err)
	}
	if operation.Path == "" && operation.OperationID == "" {
		return Operation{}, fmt.Errorf("Roblox asset create response did not include an operation path")
	}
	return operation, nil
}

func (client Client) NewCreateAssetRequest(ctx context.Context, accessToken string, upload AssetUploadRequest) (*http.Request, error) {
	if strings.TrimSpace(accessToken) == "" {
		return nil, fmt.Errorf("access token must not be empty")
	}

	payload, err := buildCreateAssetPayload(upload)
	if err != nil {
		return nil, err
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("encode Roblox asset create request: %w", err)
	}

	file, err := os.Open(upload.FilePath)
	if err != nil {
		return nil, fmt.Errorf("open upload file %s: %w", upload.FilePath, err)
	}
	defer file.Close()

	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	if err := writer.WriteField("request", string(payloadJSON)); err != nil {
		return nil, fmt.Errorf("write Roblox asset request metadata: %w", err)
	}

	contentType, err := imageContentType(upload.FilePath)
	if err != nil {
		return nil, err
	}
	header := textproto.MIMEHeader{}
	header.Set("Content-Disposition", fmt.Sprintf(`form-data; name="fileContent"; filename="%s"`, escapeQuotes(filepath.Base(upload.FilePath))))
	header.Set("Content-Type", contentType)
	part, err := writer.CreatePart(header)
	if err != nil {
		return nil, fmt.Errorf("write Roblox asset file part: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return nil, fmt.Errorf("write Roblox asset file content: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("finish Roblox asset multipart request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(client.baseURL(), "/")+"/assets/v1/assets", &body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

func buildCreateAssetPayload(upload AssetUploadRequest) (createAssetPayload, error) {
	displayName := strings.TrimSpace(upload.DisplayName)
	if displayName == "" {
		return createAssetPayload{}, fmt.Errorf("display name must not be empty")
	}

	assetType, err := robloxAssetType(upload.AssetType)
	if err != nil {
		return createAssetPayload{}, err
	}

	creatorID, err := strconv.ParseInt(strings.TrimSpace(upload.Creator.ID), 10, 64)
	if err != nil || creatorID <= 0 {
		return createAssetPayload{}, fmt.Errorf("creator ID must be a positive integer")
	}

	creator := creatorPayload{}
	switch upload.Creator.Type {
	case "user":
		creator.UserID = &creatorID
	case "group":
		creator.GroupID = &creatorID
	default:
		return createAssetPayload{}, fmt.Errorf("creator type must be user or group")
	}

	description := upload.Description
	if description == "" {
		description = DefaultDescription
	}

	return createAssetPayload{
		AssetType:   assetType,
		DisplayName: displayName,
		Description: description,
		CreationContext: creationContext{
			Creator:       creator,
			ExpectedPrice: 0,
		},
	}, nil
}

func robloxAssetType(assetType string) (string, error) {
	switch assetType {
	case "Decal", "ASSET_TYPE_DECAL":
		return "ASSET_TYPE_DECAL", nil
	default:
		return "", fmt.Errorf("unsupported upload asset type %q", assetType)
	}
}

func imageContentType(path string) (string, error) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".png":
		return "image/png", nil
	case ".jpg", ".jpeg":
		return "image/jpeg", nil
	case ".bmp":
		return "image/bmp", nil
	case ".tga":
		return "image/tga", nil
	default:
		return "", fmt.Errorf("unsupported Roblox upload image type %q", filepath.Ext(path))
	}
}

func (client Client) baseURL() string {
	if client.BaseURL != "" {
		return client.BaseURL
	}
	return DefaultBaseURL
}

func (client Client) httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}

func escapeQuotes(value string) string {
	return strings.ReplaceAll(value, `"`, `\"`)
}
