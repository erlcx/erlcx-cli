package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const pkceVerifierBytes = 32

func GenerateVerifier() (string, error) {
	data := make([]byte, pkceVerifierBytes)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate PKCE verifier: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}

func ChallengeS256(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

func GenerateState() (string, error) {
	data := make([]byte, 32)
	if _, err := rand.Read(data); err != nil {
		return "", fmt.Errorf("generate OAuth state: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(data), nil
}
