package auth

import (
	"regexp"
	"testing"
)

func TestGenerateVerifierAndChallenge(t *testing.T) {
	verifier, err := GenerateVerifier()
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(verifier) < 43 {
		t.Fatalf("expected verifier to have at least 43 characters, got %d", len(verifier))
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(verifier) {
		t.Fatalf("expected URL-safe verifier, got %q", verifier)
	}

	challenge := ChallengeS256(verifier)
	if len(challenge) < 43 {
		t.Fatalf("expected challenge to have at least 43 characters, got %d", len(challenge))
	}
	if !regexp.MustCompile(`^[A-Za-z0-9_-]+$`).MatchString(challenge) {
		t.Fatalf("expected URL-safe challenge, got %q", challenge)
	}
}

func TestChallengeS256KnownValue(t *testing.T) {
	got := ChallengeS256("dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk")
	want := "E9Melhoa2OwvFrEMTJguCHaoeK1t8URWbuGJSstw-cM"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
