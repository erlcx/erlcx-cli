package cli

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/erlcx/cli/internal/auth"
	"github.com/erlcx/cli/internal/lockfile"
	"github.com/erlcx/cli/internal/uploader"
)

func TestRunPrintsHelpWithoutArgs(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run(nil, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Usage:") {
		t.Fatalf("expected help output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
}

func TestRunReturnsErrorForUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"missing"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Unknown command: missing") {
		t.Fatalf("expected unknown command error, got %q", stderr.String())
	}
}

func TestRunPrintsAuthHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "erlcx auth <command>") {
		t.Fatalf("expected auth help output, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
}

func TestRunAuthStatusReportsMissingLogin(t *testing.T) {
	withAuthService(t, auth.Service{Store: &cliMemoryStore{}})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "status"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stdout.String(), "Not logged in.") {
		t.Fatalf("expected missing login status, got %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr, got %q", stderr.String())
	}
}

func TestRunAuthLogoutDeletesStoredCredential(t *testing.T) {
	store := &cliMemoryStore{
		credential: auth.StoredCredential{ClientID: "client", RefreshToken: "refresh"},
		hasValue:   true,
	}
	withAuthService(t, auth.Service{Store: store})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "logout"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if store.hasValue {
		t.Fatal("expected logout to delete credential")
	}
	if !strings.Contains(stdout.String(), "Logged out.") {
		t.Fatalf("expected logout output, got %q", stdout.String())
	}
}

func TestRunAuthClearOAuthAppDeletesStoredCredential(t *testing.T) {
	store := &cliMemoryStore{
		credential: auth.StoredCredential{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "refresh",
		},
		hasValue: true,
	}
	withAuthService(t, auth.Service{Store: store})

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "clear-oauth-app"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if store.hasValue {
		t.Fatal("expected stored OAuth app to be deleted")
	}
	if !strings.Contains(stdout.String(), "Cleared saved OAuth app") {
		t.Fatalf("expected clear output, got %q", stdout.String())
	}
}

func TestRunAuthLoginRequiresClientID(t *testing.T) {
	withAuthService(t, auth.Service{Store: &cliMemoryStore{}})
	withAuthPromptInput(t, "\n\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "login"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "client ID") {
		t.Fatalf("expected client ID error, got %q", stderr.String())
	}
}

func TestRunAuthLoginPromptsForCustomOAuthApp(t *testing.T) {
	service := auth.Service{
		Store: &cliMemoryStore{},
		OpenBrowser: func(string) error {
			return nil
		},
		CallbackTimeout: time.Millisecond,
	}
	withAuthService(t, service)
	withAuthPromptInput(t, "1\nclient\nsecret\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "login"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected login to progress to OAuth callback and fail later, got code %d", code)
	}
	if strings.Contains(stderr.String(), "client ID") {
		t.Fatalf("expected prompted client ID to be used, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "Custom Roblox OAuth app") {
		t.Fatalf("expected login choices, got %q", stdout.String())
	}
}

func TestRunAuthLoginReportsERLCXLoginUnavailable(t *testing.T) {
	withAuthService(t, auth.Service{Store: &cliMemoryStore{}})
	withAuthPromptInput(t, "2\n")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "login"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "not available yet") {
		t.Fatalf("expected unavailable message, got %q", stderr.String())
	}
}

func TestRunScanPrintsCountsAndReasons(t *testing.T) {
	withAuthService(t, auth.Service{Store: &cliMemoryStore{}})
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "Vehicle", "Left.png"), "image")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"scan", packDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d and output %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "Scanned 1 images") {
		t.Fatalf("expected scan count, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "upload          Vehicle/Left.png") {
		t.Fatalf("expected upload item, got %q", stderr.String())
	}
}

func TestRunScanHidesTemplateDetailsUnlessVerbose(t *testing.T) {
	withAuthService(t, auth.Service{Store: &cliMemoryStore{}})
	packDir := t.TempDir()
	templatesDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "Vehicle", "Left.png"), "same")
	writeFile(t, filepath.Join(templatesDir, "Left.png"), "same")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"scan", packDir, "--templates", templatesDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d and output %q", code, stderr.String())
	}
	if strings.Contains(stderr.String(), "matched template") {
		t.Fatalf("expected template details to be hidden by default, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "1 unchanged/template/skip entries hidden") {
		t.Fatalf("expected hidden summary, got %q", stderr.String())
	}

	stdout.Reset()
	stderr.Reset()
	code = Run([]string{"scan", packDir, "--templates", templatesDir, "--verbose"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d and output %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "matched template") {
		t.Fatalf("expected verbose template details, got %q", stderr.String())
	}
}

func TestRunUploadDryRunUsesScanPlannerWithoutWritingFiles(t *testing.T) {
	withAuthService(t, auth.Service{Store: &cliMemoryStore{}})
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "Vehicle", "Left.png"), "image")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upload", packDir, "--dry-run"}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d and output %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "dry-run:") {
		t.Fatalf("expected dry-run output, got %q", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(packDir, ".erlcx-upload.lock.json")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run not to write lock file, stat err was %v", err)
	}
	if _, err := os.Stat(filepath.Join(packDir, "IDs.txt")); !os.IsNotExist(err) {
		t.Fatalf("expected dry-run not to write IDs file, stat err was %v", err)
	}
}

func TestRunUploadUploadsAndWritesLockAndIDs(t *testing.T) {
	store := &cliMemoryStore{
		credential: auth.StoredCredential{
			ClientID:     "client",
			RefreshToken: "refresh",
			UserID:       "123",
			Username:     "tester",
		},
		hasValue: true,
	}
	oauthServer := cliOAuthServer(t)
	defer oauthServer.Close()
	uploadServer := cliUploadServer(t)
	defer uploadServer.Close()

	withAuthService(t, auth.Service{
		Store: store,
		OAuth: auth.OAuthClient{BaseURL: oauthServer.URL},
	})
	withUploaderClient(t, uploader.Client{BaseURL: uploadServer.URL})

	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "Vehicle", "Left.png"), "image")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upload", packDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d and output %q", code, stderr.String())
	}

	lock := readLockFile(t, filepath.Join(packDir, ".erlcx-upload.lock.json"))
	entry, ok := lock.Files["Vehicle/Left.png"]
	if !ok {
		t.Fatalf("expected lock entry, got %#v", lock.Files)
	}
	if entry.AssetID != "2205400862" {
		t.Fatalf("expected asset ID, got %q", entry.AssetID)
	}
	if entry.AssetType != lockfile.AssetTypeImage {
		t.Fatalf("expected image asset type, got %q", entry.AssetType)
	}

	idsData, err := os.ReadFile(filepath.Join(packDir, "IDs.txt"))
	if err != nil {
		t.Fatalf("read IDs file: %v", err)
	}
	if !strings.Contains(string(idsData), "Left: 2205400862") {
		t.Fatalf("expected IDs file to contain asset ID, got %q", string(idsData))
	}
}

func TestRunUploadReportsFakeServerPollingFailure(t *testing.T) {
	store := &cliMemoryStore{
		credential: auth.StoredCredential{
			ClientID:     "client",
			RefreshToken: "refresh",
			UserID:       "123",
			Username:     "tester",
		},
		hasValue: true,
	}
	oauthServer := cliOAuthServer(t)
	defer oauthServer.Close()
	uploadServer := cliFailingUploadServer(t)
	defer uploadServer.Close()

	withAuthService(t, auth.Service{
		Store: store,
		OAuth: auth.OAuthClient{BaseURL: oauthServer.URL},
	})
	withUploaderClient(t, uploader.Client{BaseURL: uploadServer.URL})

	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "Vehicle", "Left.png"), "image")

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"upload", packDir, "--fail-fast=false"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d and output %q", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "moderation failed") {
		t.Fatalf("expected polling failure message, got %q", stderr.String())
	}
	if !strings.Contains(stderr.String(), "upload completed with failures") {
		t.Fatalf("expected non-fail-fast summary, got %q", stderr.String())
	}

	lock := readLockFile(t, filepath.Join(packDir, ".erlcx-upload.lock.json"))
	if len(lock.Files) != 0 {
		t.Fatalf("expected failed upload not to be recorded, got %#v", lock.Files)
	}
}

func TestRunIDsRegeneratesIDsFromLockFile(t *testing.T) {
	packDir := t.TempDir()
	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "123"})
	lock.Files["Vehicle/Left.png"] = testLockEntry("2205400862")
	if err := lockfile.Save(filepath.Join(packDir, ".erlcx-upload.lock.json"), lock); err != nil {
		t.Fatalf("save lock file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"ids", packDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d and output %q", code, stderr.String())
	}
	data, err := os.ReadFile(filepath.Join(packDir, "IDs.txt"))
	if err != nil {
		t.Fatalf("read IDs file: %v", err)
	}
	if !strings.Contains(string(data), "Left: 2205400862") {
		t.Fatalf("expected regenerated IDs file, got %q", string(data))
	}
}

func TestRunRoutesLockClean(t *testing.T) {
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, "Vehicle", "Left.png"), "image")

	lock := lockfile.New(lockfile.Creator{Type: lockfile.CreatorTypeUser, ID: "123"})
	lock.Files["Vehicle/Left.png"] = testLockEntry("111")
	lock.Files["Vehicle/Missing.png"] = testLockEntry("222")
	if err := lockfile.Save(filepath.Join(packDir, ".erlcx-upload.lock.json"), lock); err != nil {
		t.Fatalf("save lock file: %v", err)
	}

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"lock", "clean", packDir}, &stdout, &stderr)

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}
	if !strings.Contains(stderr.String(), "removed: 1 stale lock entries") {
		t.Fatalf("expected clean summary, got %q", stderr.String())
	}

	cleaned := readLockFile(t, filepath.Join(packDir, ".erlcx-upload.lock.json"))
	if _, ok := cleaned.Files["Vehicle/Left.png"]; !ok {
		t.Fatalf("expected existing file to remain, got %#v", cleaned.Files)
	}
	if _, ok := cleaned.Files["Vehicle/Missing.png"]; ok {
		t.Fatalf("expected missing file to be removed, got %#v", cleaned.Files)
	}
}

func TestRunFileCommandRequiresPackDir(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"scan"}, &stdout, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "Usage: erlcx scan <pack-dir>") {
		t.Fatalf("expected usage error, got %q", stderr.String())
	}
}

func TestParseFileCommandOptionsAppliesCLIOverConfig(t *testing.T) {
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, ".erlcx-uploader.json"), `{
		"templatesDir": "from-config",
		"outputFile": "config-ids.txt",
		"lockFile": "config-lock.json",
		"concurrency": 2,
		"creator": {
			"type": "group",
			"groupId": 111
		}
	}`)

	var stderr bytes.Buffer
	opts, code := parseFileCommandOptions("upload", []string{
		packDir,
		"--templates", "from-cli",
		"--output", "cli-ids.txt",
		"--lock-file", "cli-lock.json",
		"--creator", "group",
		"--group-id", "222",
		"--concurrency", "7",
		"--dry-run",
	}, &stderr)

	if code != -1 {
		t.Fatalf("expected successful parse marker, got code %d and stderr %q", code, stderr.String())
	}
	if opts.PackDir != packDir {
		t.Fatalf("expected pack dir %q, got %q", packDir, opts.PackDir)
	}
	if opts.Config.TemplatesDir != mustAbs(t, "from-cli") {
		t.Fatalf("expected CLI templates dir, got %q", opts.Config.TemplatesDir)
	}
	if opts.Config.OutputFile != mustAbs(t, "cli-ids.txt") {
		t.Fatalf("expected CLI output file, got %q", opts.Config.OutputFile)
	}
	if opts.Config.LockFile != mustAbs(t, "cli-lock.json") {
		t.Fatalf("expected CLI lock file, got %q", opts.Config.LockFile)
	}
	if opts.Config.Creator.GroupID == nil || *opts.Config.Creator.GroupID != 222 {
		t.Fatalf("expected CLI group ID 222, got %#v", opts.Config.Creator.GroupID)
	}
	if opts.Config.Concurrency != 7 {
		t.Fatalf("expected CLI concurrency 7, got %d", opts.Config.Concurrency)
	}
	if !opts.DryRun {
		t.Fatal("expected dry run to be true")
	}
}

func TestParseFileCommandOptionsResolvesCLITemplatesRelativeToWorkingDirectory(t *testing.T) {
	packDir := t.TempDir()

	var stderr bytes.Buffer
	opts, code := parseFileCommandOptions("scan", []string{
		packDir,
		"--templates", "templates",
	}, &stderr)

	if code != -1 {
		t.Fatalf("expected successful parse marker, got code %d and stderr %q", code, stderr.String())
	}
	if opts.Config.TemplatesDir != mustAbs(t, "templates") {
		t.Fatalf("expected CLI templates path to resolve from working directory, got %q", opts.Config.TemplatesDir)
	}
}

func TestParseFileCommandOptionsCanOverrideGroupConfigToUser(t *testing.T) {
	packDir := t.TempDir()
	writeFile(t, filepath.Join(packDir, ".erlcx-uploader.json"), `{
		"creator": {
			"type": "group",
			"groupId": 111
		}
	}`)

	var stderr bytes.Buffer
	opts, code := parseFileCommandOptions("scan", []string{packDir, "--creator", "user"}, &stderr)

	if code != -1 {
		t.Fatalf("expected successful parse marker, got code %d and stderr %q", code, stderr.String())
	}
	if opts.Config.Creator.Type != "user" {
		t.Fatalf("expected user creator, got %q", opts.Config.Creator.Type)
	}
	if opts.Config.Creator.GroupID != nil {
		t.Fatalf("expected group ID to be cleared, got %#v", opts.Config.Creator.GroupID)
	}
}

func TestParseFileCommandOptionsRejectsInvalidFlagPrecedenceResult(t *testing.T) {
	packDir := t.TempDir()

	var stderr bytes.Buffer
	_, code := parseFileCommandOptions("scan", []string{packDir, "--creator", "group"}, &stderr)

	if code != 2 {
		t.Fatalf("expected exit code 2, got %d", code)
	}
	if !strings.Contains(stderr.String(), "groupId") {
		t.Fatalf("expected group ID validation error, got %q", stderr.String())
	}
}

func writeFile(t *testing.T, path string, content string) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("create parent dir for %s: %v", path, err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
}

func mustAbs(t *testing.T, path string) string {
	t.Helper()

	absPath, err := filepath.Abs(path)
	if err != nil {
		t.Fatalf("resolve abs path %s: %v", path, err)
	}
	return absPath
}

func TestRunReturnsErrorForUnknownSubcommands(t *testing.T) {
	for _, args := range [][]string{
		{"auth", "missing"},
		{"lock", "missing"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := Run(args, &stdout, &stderr)

		if code != 2 {
			t.Fatalf("expected exit code 2 for %v, got %d", args, code)
		}
		if !strings.Contains(stderr.String(), "Unknown") {
			t.Fatalf("expected unknown subcommand error for %v, got %q", args, stderr.String())
		}
	}
}

type cliMemoryStore struct {
	credential auth.StoredCredential
	hasValue   bool
}

func (store *cliMemoryStore) Save(credential auth.StoredCredential) error {
	store.credential = credential
	store.hasValue = true
	return nil
}

func (store *cliMemoryStore) Load() (auth.StoredCredential, error) {
	if !store.hasValue {
		return auth.StoredCredential{}, auth.ErrNotLoggedIn
	}
	return store.credential, nil
}

func (store *cliMemoryStore) Delete() error {
	if !store.hasValue {
		return auth.ErrNotLoggedIn
	}
	store.hasValue = false
	return nil
}

func withAuthService(t *testing.T, service auth.Service) {
	t.Helper()

	previous := newAuthService
	newAuthService = func() auth.Service {
		return service
	}
	t.Cleanup(func() {
		newAuthService = previous
	})
}

func withAuthPromptInput(t *testing.T, input string) {
	t.Helper()

	previous := authPromptInput
	authPromptInput = strings.NewReader(input)
	t.Cleanup(func() {
		authPromptInput = previous
	})
}

func withUploaderClient(t *testing.T, client uploader.Client) {
	t.Helper()

	previous := newUploaderClient
	newUploaderClient = func() uploader.Client {
		return client
	}
	t.Cleanup(func() {
		newUploaderClient = previous
	})
}

func cliOAuthServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/token" {
			t.Fatalf("expected token endpoint, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse token form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Fatalf("expected refresh grant, got %q", r.Form.Get("grant_type"))
		}
		writeJSONResponse(t, w, auth.TokenSet{
			AccessToken:  "access",
			RefreshToken: "rotated",
			Scope:        "openid profile asset:read asset:write",
		})
	}))
}

func cliUploadServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/v1/assets":
			if r.Header.Get("Authorization") != "Bearer access" {
				t.Fatalf("expected bearer access token, got %q", r.Header.Get("Authorization"))
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse upload multipart: %v", err)
			}
			writeJSONResponse(t, w, uploader.Operation{Path: "operations/op-1", OperationID: "op-1"})
		case r.Method == http.MethodGet && r.URL.Path == "/assets/v1/operations/op-1":
			writeJSONResponse(t, w, uploader.Operation{
				Path: "operations/op-1",
				Done: true,
				Response: &uploader.Asset{
					AssetID: "2205400862",
				},
			})
		default:
			t.Fatalf("unexpected upload request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func cliFailingUploadServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/assets/v1/assets":
			if r.Header.Get("Authorization") != "Bearer access" {
				t.Fatalf("expected bearer access token, got %q", r.Header.Get("Authorization"))
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				t.Fatalf("parse upload multipart: %v", err)
			}
			writeJSONResponse(t, w, uploader.Operation{Path: "operations/op-1", OperationID: "op-1"})
		case r.Method == http.MethodGet && r.URL.Path == "/assets/v1/operations/op-1":
			writeJSONResponse(t, w, uploader.Operation{
				Path:   "operations/op-1",
				Done:   true,
				Status: &uploader.OperationStatus{Message: "moderation failed"},
			})
		default:
			t.Fatalf("unexpected upload request %s %s", r.Method, r.URL.Path)
		}
	}))
}

func writeJSONResponse(t *testing.T, w http.ResponseWriter, value any) {
	t.Helper()

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		t.Fatalf("write json response: %v", err)
	}
}

func readLockFile(t *testing.T, path string) lockfile.LockFile {
	t.Helper()

	lock, err := lockfile.Load(path)
	if err != nil {
		t.Fatalf("load lock file: %v", err)
	}
	return lock
}

func testLockEntry(assetID string) lockfile.Entry {
	return lockfile.Entry{
		SHA256:      "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
		AssetType:   lockfile.AssetTypeImage,
		AssetID:     assetID,
		DisplayName: "Vehicle - Left",
		UploadedAt:  time.Date(2026, 4, 26, 18, 0, 0, 0, time.UTC),
	}
}
