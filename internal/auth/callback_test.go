package auth

import (
	"context"
	"net/http"
	"testing"
	"time"
)

func TestCallbackServerAcceptsCodeWithExpectedState(t *testing.T) {
	callback, err := StartCallbackServerAt("state", "http://localhost:0/callback")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	go func() {
		resp, err := http.Get(callback.RedirectURI + "?code=abc&state=state")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	result, err := callback.Wait(context.Background(), time.Second)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if result.Code != "abc" {
		t.Fatalf("expected code abc, got %q", result.Code)
	}
}

func TestCallbackServerRejectsStateMismatch(t *testing.T) {
	callback, err := StartCallbackServerAt("state", "http://localhost:0/callback")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	go func() {
		resp, err := http.Get(callback.RedirectURI + "?code=abc&state=wrong")
		if err == nil {
			_ = resp.Body.Close()
		}
	}()

	_, err = callback.Wait(context.Background(), time.Second)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}
