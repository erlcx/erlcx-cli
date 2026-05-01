package auth

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAuthorizationURLUsesRobloxOAuthParameters(t *testing.T) {
	got := (OAuthClient{BaseURL: "https://example.test/oauth"}).AuthorizationURL(AuthorizationParams{
		ClientID:      "client",
		RedirectURI:   "http://127.0.0.1/callback",
		Scopes:        "openid profile",
		State:         "state",
		CodeChallenge: "challenge",
	})

	for _, part := range []string{
		"https://example.test/oauth/v1/authorize?",
		"client_id=client",
		"redirect_uri=http%3A%2F%2F127.0.0.1%2Fcallback",
		"scope=openid+profile",
		"response_type=code",
		"state=state",
		"code_challenge=challenge",
		"code_challenge_method=S256",
	} {
		if !strings.Contains(got, part) {
			t.Fatalf("expected auth URL to contain %q, got %q", part, got)
		}
	}
}

func TestExchangeCodeAndRefreshUseTokenEndpoint(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/token" {
			t.Fatalf("expected token path, got %s", r.URL.Path)
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("parse form: %v", err)
		}
		if r.Form.Get("client_id") != "client" {
			t.Fatalf("expected client ID, got %q", r.Form.Get("client_id"))
		}
		if r.Form.Get("client_secret") != "secret" {
			t.Fatalf("expected client secret, got %q", r.Form.Get("client_secret"))
		}
		switch r.Form.Get("grant_type") {
		case "authorization_code":
			if r.Form.Get("code") != "code" || r.Form.Get("code_verifier") != "verifier" {
				t.Fatalf("expected code exchange form, got %#v", r.Form)
			}
			writeJSON(w, TokenSet{AccessToken: "access", RefreshToken: "refresh"})
		case "refresh_token":
			if r.Form.Get("refresh_token") != "old-refresh" {
				t.Fatalf("expected refresh token form, got %#v", r.Form)
			}
			writeJSON(w, TokenSet{AccessToken: "new-access", RefreshToken: "new-refresh", Scope: "openid"})
		default:
			t.Fatalf("unexpected grant type %q", r.Form.Get("grant_type"))
		}
	}))
	defer server.Close()

	client := OAuthClient{BaseURL: server.URL}

	exchanged, err := client.ExchangeCode(context.Background(), "client", "secret", "http://redirect", "code", "verifier")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if exchanged.AccessToken != "access" || exchanged.RefreshToken != "refresh" {
		t.Fatalf("expected exchanged tokens, got %#v", exchanged)
	}

	refreshed, err := client.Refresh(context.Background(), "client", "secret", "old-refresh")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if refreshed.AccessToken != "new-access" || refreshed.RefreshToken != "new-refresh" {
		t.Fatalf("expected refreshed tokens, got %#v", refreshed)
	}
}

func TestUserInfoUsesBearerToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/userinfo" {
			t.Fatalf("expected userinfo path, got %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access" {
			t.Fatalf("expected bearer auth, got %q", r.Header.Get("Authorization"))
		}
		writeJSON(w, UserInfo{Subject: "123", PreferredName: "user", Name: "Display"})
	}))
	defer server.Close()

	info, err := (OAuthClient{BaseURL: server.URL}).UserInfo(context.Background(), "access")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if info.Subject != "123" || info.PreferredName != "user" || info.Name != "Display" {
		t.Fatalf("expected user info, got %#v", info)
	}
}

func writeJSON(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(value); err != nil {
		panic(err)
	}
}
