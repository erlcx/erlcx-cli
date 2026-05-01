package auth

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	DefaultOAuthBaseURL = "https://apis.roblox.com/oauth"
	DefaultScopes       = "openid profile asset:read asset:write"
	DefaultRedirectURI  = "http://localhost:53682/callback"
)

type OAuthClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

type TokenSet struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	IDToken      string `json:"id_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	Scope        string `json:"scope"`
}

type UserInfo struct {
	Subject       string `json:"sub"`
	Name          string `json:"name"`
	PreferredName string `json:"preferred_username"`
	Profile       string `json:"profile"`
}

func (client OAuthClient) AuthorizationURL(params AuthorizationParams) string {
	base := strings.TrimRight(client.baseURL(), "/") + "/v1/authorize"
	values := url.Values{}
	values.Set("client_id", params.ClientID)
	values.Set("redirect_uri", params.RedirectURI)
	values.Set("scope", params.Scopes)
	values.Set("response_type", "code")
	values.Set("state", params.State)
	values.Set("code_challenge", params.CodeChallenge)
	values.Set("code_challenge_method", "S256")
	return base + "?" + values.Encode()
}

type AuthorizationParams struct {
	ClientID      string
	RedirectURI   string
	Scopes        string
	State         string
	CodeChallenge string
}

func (client OAuthClient) ExchangeCode(ctx context.Context, clientID string, clientSecret string, redirectURI string, code string, verifier string) (TokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", clientID)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	form.Set("redirect_uri", redirectURI)
	form.Set("code", code)
	form.Set("code_verifier", verifier)
	return client.postToken(ctx, form)
}

func (client OAuthClient) Refresh(ctx context.Context, clientID string, clientSecret string, refreshToken string) (TokenSet, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", clientID)
	if clientSecret != "" {
		form.Set("client_secret", clientSecret)
	}
	form.Set("refresh_token", refreshToken)
	return client.postToken(ctx, form)
}

func (client OAuthClient) UserInfo(ctx context.Context, accessToken string) (UserInfo, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(client.baseURL(), "/")+"/v1/userinfo", nil)
	if err != nil {
		return UserInfo{}, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return UserInfo{}, fmt.Errorf("get Roblox user info: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return UserInfo{}, fmt.Errorf("read Roblox user info: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UserInfo{}, fmt.Errorf("get Roblox user info: %s: %s", resp.Status, string(bytes.TrimSpace(body)))
	}

	var info UserInfo
	if err := json.Unmarshal(body, &info); err != nil {
		return UserInfo{}, fmt.Errorf("decode Roblox user info: %w", err)
	}
	return info, nil
}

func (client OAuthClient) postToken(ctx context.Context, form url.Values) (TokenSet, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(client.baseURL(), "/")+"/v1/token", strings.NewReader(form.Encode()))
	if err != nil {
		return TokenSet{}, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.httpClient().Do(req)
	if err != nil {
		return TokenSet{}, fmt.Errorf("call Roblox token endpoint: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return TokenSet{}, fmt.Errorf("read Roblox token response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return TokenSet{}, fmt.Errorf("Roblox token endpoint returned %s: %s", resp.Status, string(bytes.TrimSpace(body)))
	}

	var tokens TokenSet
	if err := json.Unmarshal(body, &tokens); err != nil {
		return TokenSet{}, fmt.Errorf("decode Roblox token response: %w", err)
	}
	if tokens.AccessToken == "" {
		return TokenSet{}, fmt.Errorf("Roblox token response did not include an access token")
	}
	if tokens.RefreshToken == "" {
		return TokenSet{}, fmt.Errorf("Roblox token response did not include a refresh token")
	}
	return tokens, nil
}

func (client OAuthClient) baseURL() string {
	if client.BaseURL != "" {
		return client.BaseURL
	}
	return DefaultOAuthBaseURL
}

func (client OAuthClient) httpClient() *http.Client {
	if client.HTTPClient != nil {
		return client.HTTPClient
	}
	return &http.Client{Timeout: 30 * time.Second}
}
