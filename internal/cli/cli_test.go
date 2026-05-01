package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/erlcx/cli/internal/auth"
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

func TestRunAuthLoginRequiresClientID(t *testing.T) {
	withAuthService(t, auth.Service{Store: &cliMemoryStore{}})

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

func TestRunAuthLoginLoadsClientIDFromDotEnv(t *testing.T) {
	workingDir := t.TempDir()
	previousDir, err := os.Getwd()
	if err != nil {
		t.Fatalf("get working dir: %v", err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatalf("change working dir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(previousDir)
	})

	writeFile(t, filepath.Join(workingDir, ".env"), "ERLCX_ROBLOX_CLIENT_ID=from-dotenv\n")
	t.Setenv(authClientIDEnv, "")
	_ = os.Unsetenv(authClientIDEnv)

	service := auth.Service{
		Store: &cliMemoryStore{},
		OpenBrowser: func(string) error {
			return nil
		},
	}
	withAuthService(t, service)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := Run([]string{"auth", "login"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected login to progress past missing client ID and fail later, got code %d", code)
	}
	if strings.Contains(stderr.String(), "client ID") {
		t.Fatalf("expected .env client ID to be loaded, got %q", stderr.String())
	}
}

func TestRunRoutesTopLevelCommands(t *testing.T) {
	for _, command := range []string{"scan", "upload", "ids"} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		packDir := t.TempDir()

		code := Run([]string{command, packDir}, &stdout, &stderr)

		if code != 1 {
			t.Fatalf("expected exit code 1 for %s, got %d", command, code)
		}
		if !strings.Contains(stderr.String(), command+" is not implemented yet.") {
			t.Fatalf("expected routed unimplemented error for %s, got %q", command, stderr.String())
		}
	}
}

func TestRunRoutesLockClean(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	packDir := t.TempDir()

	code := Run([]string{"lock", "clean", packDir}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "lock clean is not implemented yet.") {
		t.Fatalf("expected lock clean routing, got %q", stderr.String())
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
	if opts.Config.TemplatesDir != "from-cli" {
		t.Fatalf("expected CLI templates dir, got %q", opts.Config.TemplatesDir)
	}
	if opts.Config.OutputFile != "cli-ids.txt" {
		t.Fatalf("expected CLI output file, got %q", opts.Config.OutputFile)
	}
	if opts.Config.LockFile != "cli-lock.json" {
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

	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("write file %s: %v", path, err)
	}
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
