package uploader

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path"
	"strings"
	"time"
)

type Operation struct {
	Path        string           `json:"path"`
	OperationID string           `json:"operationId"`
	Done        bool             `json:"done"`
	Response    *Asset           `json:"response"`
	Status      *OperationStatus `json:"status"`
}

type Asset struct {
	Path        string `json:"path"`
	AssetID     string `json:"assetId"`
	DisplayName string `json:"displayName"`
	AssetType   string `json:"assetType"`
}

type OperationStatus struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type PollOptions struct {
	Interval time.Duration
	Timeout  time.Duration
}

func (client Client) PollOperation(ctx context.Context, accessToken string, operation Operation, options PollOptions) (Asset, error) {
	if strings.TrimSpace(accessToken) == "" {
		return Asset{}, fmt.Errorf("access token must not be empty")
	}

	ctx, cancel := context.WithTimeout(ctx, pollTimeout(options))
	defer cancel()

	timer := time.NewTimer(0)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			return Asset{}, fmt.Errorf("poll Roblox asset operation timed out: %w", ctx.Err())
		case <-timer.C:
			current, err := client.GetOperation(ctx, accessToken, operationPath(operation))
			if err != nil {
				return Asset{}, err
			}
			asset, done, err := AssetFromOperation(current)
			if err != nil {
				return Asset{}, err
			}
			if done {
				return asset, nil
			}
			timer.Reset(pollInterval(options))
		}
	}
}

func (client Client) GetOperation(ctx context.Context, accessToken string, operationPath string) (Operation, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(client.baseURL(), "/")+"/assets/v1/"+strings.TrimLeft(operationPath, "/"), nil)
	if err != nil {
		return Operation{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return Operation{}, fmt.Errorf("get Roblox asset operation: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return Operation{}, fmt.Errorf("read Roblox asset operation response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return Operation{}, fmt.Errorf("get Roblox asset operation: %s: %s", resp.Status, string(bytes.TrimSpace(body)))
	}

	var operation Operation
	if err := json.Unmarshal(body, &operation); err != nil {
		return Operation{}, fmt.Errorf("decode Roblox asset operation response: %w", err)
	}
	return operation, nil
}

func AssetFromOperation(operation Operation) (Asset, bool, error) {
	if !operation.Done {
		return Asset{}, false, nil
	}
	if operation.Status != nil {
		return Asset{}, true, fmt.Errorf("Roblox asset operation failed: %s", operation.Status.Message)
	}
	if operation.Response == nil {
		return Asset{}, true, fmt.Errorf("Roblox asset operation finished without an asset response")
	}
	if operation.Response.AssetID == "" && operation.Response.Path != "" {
		operation.Response.AssetID = path.Base(operation.Response.Path)
	}
	if operation.Response.AssetID == "" {
		return Asset{}, true, fmt.Errorf("Roblox asset operation response did not include an asset ID")
	}
	return *operation.Response, true, nil
}

func operationPath(operation Operation) string {
	if operation.Path != "" {
		return operation.Path
	}
	if operation.OperationID != "" {
		return "operations/" + operation.OperationID
	}
	return ""
}

func pollInterval(options PollOptions) time.Duration {
	if options.Interval > 0 {
		return options.Interval
	}
	return 2 * time.Second
}

func pollTimeout(options PollOptions) time.Duration {
	if options.Timeout > 0 {
		return options.Timeout
	}
	return 2 * time.Minute
}
