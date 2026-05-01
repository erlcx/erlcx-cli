package auth

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type memoryStore struct {
	credential StoredCredential
	hasValue   bool
}

func (store *memoryStore) Save(credential StoredCredential) error {
	store.credential = credential
	store.hasValue = true
	return nil
}

func (store *memoryStore) Load() (StoredCredential, error) {
	if !store.hasValue {
		return StoredCredential{}, ErrNotLoggedIn
	}
	return store.credential, nil
}

func (store *memoryStore) Delete() error {
	if !store.hasValue {
		return ErrNotLoggedIn
	}
	store.hasValue = false
	return nil
}

func TestStatusReturnsNotLoggedInWhenCredentialMissing(t *testing.T) {
	status, err := (Service{Store: &memoryStore{}}).Status(context.Background(), StatusOptions{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if status.LoggedIn {
		t.Fatal("expected not logged in")
	}
}

func TestStatusRefreshesAndRotatesRefreshToken(t *testing.T) {
	store := &memoryStore{
		credential: StoredCredential{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "old-refresh",
			UserID:       "123",
			Username:     "user",
			DisplayName:  "Display",
			Scopes:       "openid profile",
		},
		hasValue: true,
	}
	server := fakeOAuthServer(t)
	defer server.Close()

	status, err := (Service{
		Store: store,
		OAuth: OAuthClient{BaseURL: server.URL},
	}).Status(context.Background(), StatusOptions{
		Refresh:      true,
		ClientSecret: "secret",
	})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !status.LoggedIn || status.Username != "user" {
		t.Fatalf("expected logged-in status, got %#v", status)
	}
	if store.credential.RefreshToken != "new-refresh" {
		t.Fatalf("expected rotated refresh token, got %q", store.credential.RefreshToken)
	}
}

func TestAccessTokenRefreshesCredentialAndReturnsAccessToken(t *testing.T) {
	store := &memoryStore{
		credential: StoredCredential{
			ClientID:     "client",
			ClientSecret: "secret",
			RefreshToken: "old-refresh",
			UserID:       "123",
			Username:     "user",
		},
		hasValue: true,
	}
	server := fakeOAuthServer(t)
	defer server.Close()

	token, err := (Service{
		Store: store,
		OAuth: OAuthClient{BaseURL: server.URL},
	}).AccessToken(context.Background(), AccessTokenOptions{})

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if token.Token != "new-access" {
		t.Fatalf("expected access token, got %q", token.Token)
	}
	if token.Credential.RefreshToken != "new-refresh" {
		t.Fatalf("expected rotated refresh token, got %q", token.Credential.RefreshToken)
	}
}

func TestLogoutDeletesCredentialAndTreatsMissingAsSuccess(t *testing.T) {
	store := &memoryStore{
		credential: StoredCredential{ClientID: "client", RefreshToken: "refresh"},
		hasValue:   true,
	}
	service := Service{Store: store}

	if err := service.Logout(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if store.hasValue {
		t.Fatal("expected credential to be deleted")
	}
	if err := service.Logout(); err != nil {
		t.Fatalf("expected missing credential to be success, got %v", err)
	}
}

func TestLoginRequiresClientID(t *testing.T) {
	_, err := (Service{}).Login(context.Background(), LoginOptions{})

	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestEncodeCredentialRejectsIncompleteCredential(t *testing.T) {
	if _, err := encodeCredential(StoredCredential{}); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestDecodeCredentialRejectsIncompleteCredential(t *testing.T) {
	if _, err := decodeCredential([]byte(`{}`)); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestErrNotLoggedInCanBeWrapped(t *testing.T) {
	if !errors.Is(errors.Join(ErrNotLoggedIn), ErrNotLoggedIn) {
		t.Fatal("expected ErrNotLoggedIn to be comparable")
	}
}

func fakeOAuthServer(t *testing.T) *httptest.Server {
	t.Helper()

	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/token" {
			t.Fatalf("expected token path, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("grant_type") != "refresh_token" {
			t.Fatalf("expected refresh grant, got %q", r.Form.Get("grant_type"))
		}
		if r.Form.Get("refresh_token") != "old-refresh" {
			t.Fatalf("expected old refresh token, got %q", r.Form.Get("refresh_token"))
		}
		if r.Form.Get("client_secret") != "secret" {
			t.Fatalf("expected client secret, got %q", r.Form.Get("client_secret"))
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(TokenSet{
			AccessToken:  "new-access",
			RefreshToken: "new-refresh",
			Scope:        "openid profile asset:read asset:write",
		}); err != nil {
			t.Fatalf("encode response: %v", err)
		}
	}))
}
