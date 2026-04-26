package cli

import (
	"bytes"
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

		code := Run([]string{command}, &stdout, &stderr)

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

	code := Run([]string{"lock", "clean"}, &stdout, &stderr)

	if code != 1 {
		t.Fatalf("expected exit code 1, got %d", code)
	}
	if !strings.Contains(stderr.String(), "lock clean is not implemented yet.") {
		t.Fatalf("expected lock clean routing, got %q", stderr.String())
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
