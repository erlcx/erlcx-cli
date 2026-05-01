package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDotEnvSetsMissingVariables(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	writeFile(t, path, `
# comment
ERLCX_TEST_ALPHA=one
ERLCX_TEST_BETA="two words"
export ERLCX_TEST_GAMMA='three words'
`)
	t.Setenv("ERLCX_TEST_ALPHA", "")
	_ = os.Unsetenv("ERLCX_TEST_ALPHA")
	_ = os.Unsetenv("ERLCX_TEST_BETA")
	_ = os.Unsetenv("ERLCX_TEST_GAMMA")

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := os.Getenv("ERLCX_TEST_ALPHA"); got != "one" {
		t.Fatalf("expected alpha from .env, got %q", got)
	}
	if got := os.Getenv("ERLCX_TEST_BETA"); got != "two words" {
		t.Fatalf("expected quoted beta from .env, got %q", got)
	}
	if got := os.Getenv("ERLCX_TEST_GAMMA"); got != "three words" {
		t.Fatalf("expected exported gamma from .env, got %q", got)
	}
}

func TestLoadDotEnvDoesNotOverrideExistingEnvironment(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	writeFile(t, path, "ERLCX_TEST_OVERRIDE=from-file\n")
	t.Setenv("ERLCX_TEST_OVERRIDE", "from-env")

	if err := loadDotEnv(path); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if got := os.Getenv("ERLCX_TEST_OVERRIDE"); got != "from-env" {
		t.Fatalf("expected existing env to win, got %q", got)
	}
}

func TestLoadDotEnvRejectsMalformedLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".env")
	writeFile(t, path, "ERLCX_TEST_BAD\n")

	if err := loadDotEnv(path); err == nil {
		t.Fatal("expected error, got nil")
	}
}
