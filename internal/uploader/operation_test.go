package uploader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestAssetFromOperationParsesAssetIDFromResponse(t *testing.T) {
	asset, done, err := AssetFromOperation(Operation{
		Done: true,
		Response: &Asset{
			Path: "assets/2205400862",
		},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !done {
		t.Fatal("expected done")
	}
	if asset.AssetID != "2205400862" {
		t.Fatalf("expected asset ID from path, got %q", asset.AssetID)
	}
}

func TestAssetFromOperationReturnsFailureStatus(t *testing.T) {
	_, _, err := AssetFromOperation(Operation{
		Done:   true,
		Status: &OperationStatus{Message: "moderation failed"},
	})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestPollOperationWaitsUntilDone(t *testing.T) {
	calls := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/assets/v1/operations/op-1" {
			t.Fatalf("expected operation path, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer token" {
			t.Fatalf("expected bearer token, got %q", r.Header.Get("Authorization"))
		}
		calls++
		if calls == 1 {
			writeJSON(t, w, Operation{Path: "operations/op-1", Done: false})
			return
		}
		writeJSON(t, w, Operation{
			Path: "operations/op-1",
			Done: true,
			Response: &Asset{
				AssetID: "123",
			},
		})
	}))
	defer server.Close()

	asset, err := (Client{BaseURL: server.URL}).PollOperation(context.Background(), "token", Operation{Path: "operations/op-1"}, PollOptions{
		Interval: time.Millisecond,
		Timeout:  time.Second,
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if asset.AssetID != "123" {
		t.Fatalf("expected asset ID 123, got %q", asset.AssetID)
	}
	if calls != 2 {
		t.Fatalf("expected 2 poll calls, got %d", calls)
	}
}

func writeJSON(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json: %v", err)
	}
}
