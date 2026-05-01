package uploader

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestUploadAssetCreatesPollsAndReturnsAsset(t *testing.T) {
	filePath := writeUploadFile(t, "Left.png", []byte("image"))
	server := newUploadTestServer(t, nil)
	defer server.Close()

	_, asset, err := (Client{BaseURL: server.URL}).UploadAsset(context.Background(), "token", AssetUploadRequest{
		FilePath:    filePath,
		DisplayName: "Vehicle - Left",
		AssetType:   "Image",
		Creator:     Creator{Type: "user", ID: "123"},
	}, PollOptions{Interval: time.Millisecond, Timeout: time.Second})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if asset.AssetID == "" {
		t.Fatal("expected asset ID")
	}
}

func TestUploadManyUsesBoundedConcurrency(t *testing.T) {
	var active int32
	var maxActive int32
	server := newUploadTestServer(t, func() {
		current := atomic.AddInt32(&active, 1)
		defer atomic.AddInt32(&active, -1)
		for {
			previous := atomic.LoadInt32(&maxActive)
			if current <= previous || atomic.CompareAndSwapInt32(&maxActive, previous, current) {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	})
	defer server.Close()

	jobs := uploadJobs(t, 6)
	results, err := (Client{BaseURL: server.URL}).UploadMany(context.Background(), "token", jobs, UploadOptions{
		Concurrency: 2,
		Poll:        PollOptions{Interval: time.Millisecond, Timeout: time.Second},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != len(jobs) {
		t.Fatalf("expected %d results, got %d", len(jobs), len(results))
	}
	if maxActive > 2 {
		t.Fatalf("expected max concurrency <= 2, got %d", maxActive)
	}
}

func TestUploadManyKeepsResultsInJobOrderWhenJobIndexesAreSparse(t *testing.T) {
	server := newUploadTestServer(t, nil)
	defer server.Close()

	jobs := uploadJobs(t, 3)
	jobs[0].Index = 10
	jobs[1].Index = 20
	jobs[2].Index = 30

	results, err := (Client{BaseURL: server.URL}).UploadMany(context.Background(), "token", jobs, UploadOptions{
		Concurrency: 2,
		Poll:        PollOptions{Interval: time.Millisecond, Timeout: time.Second},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(results) != len(jobs) {
		t.Fatalf("expected %d results, got %d", len(jobs), len(results))
	}
	for i, result := range results {
		if result.Job.Index != jobs[i].Index {
			t.Fatalf("expected result %d to keep job index %d, got %d", i, jobs[i].Index, result.Job.Index)
		}
	}
}

func TestUploadManyCallsOnResultAsJobsFinish(t *testing.T) {
	server := newUploadTestServer(t, nil)
	defer server.Close()

	var mu sync.Mutex
	completed := 0
	_, err := (Client{BaseURL: server.URL}).UploadMany(context.Background(), "token", uploadJobs(t, 3), UploadOptions{
		Concurrency: 2,
		Poll:        PollOptions{Interval: time.Millisecond, Timeout: time.Second},
		OnResult: func(Result) {
			mu.Lock()
			defer mu.Unlock()
			completed++
		},
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if completed != 3 {
		t.Fatalf("expected 3 completed callbacks, got %d", completed)
	}
}

func TestUploadManyContinuesWhenFailFastDisabled(t *testing.T) {
	server := uploadServerWithFailure(t, "1")
	defer server.Close()

	results, err := (Client{BaseURL: server.URL}).UploadMany(context.Background(), "token", uploadJobs(t, 3), UploadOptions{
		Concurrency: 1,
		FailFast:    false,
		Poll:        PollOptions{Interval: time.Millisecond, Timeout: time.Second},
	})

	if err == nil {
		t.Fatal("expected first upload error, got nil")
	}
	successes := 0
	failures := 0
	for _, result := range results {
		if result.Err != nil {
			failures++
		} else if result.Asset.AssetID != "" {
			successes++
		}
	}
	if successes != 2 || failures != 1 {
		t.Fatalf("expected 2 successes and 1 failure, got %d successes and %d failures", successes, failures)
	}
}

func TestUploadManyStopsStartingWorkWhenFailFastEnabled(t *testing.T) {
	server := uploadServerWithFailure(t, "0")
	defer server.Close()

	results, err := (Client{BaseURL: server.URL}).UploadMany(context.Background(), "token", uploadJobs(t, 4), UploadOptions{
		Concurrency: 1,
		FailFast:    true,
		Poll:        PollOptions{Interval: time.Millisecond, Timeout: time.Second},
	})

	if err == nil {
		t.Fatal("expected upload error, got nil")
	}
	completed := 0
	for _, result := range results {
		if result.Job.Request.DisplayName != "" || result.Err != nil || result.Asset.AssetID != "" {
			completed++
		}
	}
	if completed != 1 {
		t.Fatalf("expected only first job to complete, got %d", completed)
	}
}

func newUploadTestServer(t *testing.T, onCreate func()) *httptest.Server {
	t.Helper()

	var mu sync.Mutex
	nextOperation := 0
	operationAssets := map[string]string{}

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/v1/assets":
			if onCreate != nil {
				onCreate()
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			displayName := displayNameFromMultipartRequest(t, r)
			index := strings.TrimPrefix(displayName, "Vehicle - Side")

			mu.Lock()
			nextOperation++
			operationID := "op-" + strconv.Itoa(nextOperation)
			operationAssets[operationID] = "asset-" + index
			mu.Unlock()

			writeJSON(t, w, Operation{Path: "operations/" + operationID, OperationID: operationID})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets/v1/operations/"):
			operationID := strings.TrimPrefix(r.URL.Path, "/assets/v1/operations/")
			mu.Lock()
			assetID := operationAssets[operationID]
			mu.Unlock()
			writeJSON(t, w, Operation{
				Path: "operations/" + operationID,
				Done: true,
				Response: &Asset{
					AssetID: assetID,
				},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func uploadServerWithFailure(t *testing.T, failingIndex string) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/v1/assets":
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse multipart form: %v", err)
			}
			displayName := displayNameFromMultipartRequest(t, r)
			index := strings.TrimPrefix(displayName, "Vehicle - Side")
			writeJSON(t, w, Operation{Path: "operations/op-" + index, OperationID: "op-" + index})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/assets/v1/operations/op-"):
			index := strings.TrimPrefix(r.URL.Path, "/assets/v1/operations/op-")
			if index == failingIndex {
				writeJSON(t, w, Operation{
					Path:   "operations/op-" + index,
					Done:   true,
					Status: &OperationStatus{Message: "failed " + index},
				})
				return
			}
			writeJSON(t, w, Operation{
				Path:     "operations/op-" + index,
				Done:     true,
				Response: &Asset{AssetID: "asset-" + index},
			})
		default:
			t.Fatalf("unexpected request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func uploadJobs(t *testing.T, count int) []Job {
	t.Helper()

	jobs := make([]Job, 0, count)
	for i := range count {
		jobs = append(jobs, Job{
			Index: i,
			Request: AssetUploadRequest{
				FilePath:    writeUploadFile(t, "Side"+strconv.Itoa(i)+".png", []byte("image")),
				DisplayName: "Vehicle - Side" + strconv.Itoa(i),
				AssetType:   "Image",
				Creator:     Creator{Type: "user", ID: "123"},
			},
		})
	}
	return jobs
}

func displayNameFromMultipartRequest(t *testing.T, r *http.Request) string {
	t.Helper()

	var payload createAssetPayload
	if err := json.Unmarshal([]byte(r.MultipartForm.Value["request"][0]), &payload); err != nil {
		t.Fatalf("decode request payload: %v", err)
	}
	return payload.DisplayName
}
