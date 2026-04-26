package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
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

func TestRunRoutesAuthSubcommands(t *testing.T) {
	for _, args := range [][]string{
		{"auth", "login"},
		{"auth", "status"},
		{"auth", "logout"},
	} {
		var stdout bytes.Buffer
		var stderr bytes.Buffer

		code := Run(args, &stdout, &stderr)

		if code != 1 {
			t.Fatalf("expected exit code 1 for %v, got %d", args, code)
		}
		if !strings.Contains(stderr.String(), strings.Join(args, " ")+" is not implemented yet.") {
			t.Fatalf("expected routed unimplemented error for %v, got %q", args, stderr.String())
		}
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
