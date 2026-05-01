package auth

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

type Service struct {
	OAuth           OAuthClient
	Store           CredentialStore
	OpenBrowser     BrowserLauncher
	CallbackTimeout time.Duration
}

type LoginOptions struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	Scopes       string
}

type StatusOptions struct {
	Refresh      bool
	ClientSecret string
}

type AccessTokenOptions struct {
	ClientSecret string
}

type AccessToken struct {
	Token      string
	Credential StoredCredential
}

type Status struct {
	LoggedIn    bool
	UserID      string
	Username    string
	DisplayName string
	Scopes      string
	ClientID    string
}

func (service Service) Login(ctx context.Context, options LoginOptions) (Status, error) {
	clientID := strings.TrimSpace(options.ClientID)
	if clientID == "" {
		return Status{}, fmt.Errorf("Roblox OAuth client ID is required")
	}

	scopes := strings.TrimSpace(options.Scopes)
	if scopes == "" {
		scopes = DefaultScopes
	}

	verifier, err := GenerateVerifier()
	if err != nil {
		return Status{}, err
	}
	state, err := GenerateState()
	if err != nil {
		return Status{}, err
	}

	redirectURI := strings.TrimSpace(options.RedirectURI)
	if redirectURI == "" {
		redirectURI = DefaultRedirectURI
	}

	callback, err := StartCallbackServerAt(state, redirectURI)
	if err != nil {
		return Status{}, err
	}
	defer callback.Close(context.Background())

	authURL := service.oauth().AuthorizationURL(AuthorizationParams{
		ClientID:      clientID,
		RedirectURI:   callback.RedirectURI,
		Scopes:        scopes,
		State:         state,
		CodeChallenge: ChallengeS256(verifier),
	})

	launcher := service.OpenBrowser
	if launcher == nil {
		launcher = OpenBrowser
	}
	if err := launcher(authURL); err != nil {
		return Status{}, err
	}

	result, err := callback.Wait(ctx, service.callbackTimeout())
	if err != nil {
		return Status{}, err
	}

	tokens, err := service.oauth().ExchangeCode(ctx, clientID, strings.TrimSpace(options.ClientSecret), callback.RedirectURI, result.Code, verifier)
	if err != nil {
		return Status{}, err
	}

	info, err := service.oauth().UserInfo(ctx, tokens.AccessToken)
	if err != nil {
		return Status{}, err
	}

	credential := StoredCredential{
		ClientID:     clientID,
		ClientSecret: strings.TrimSpace(options.ClientSecret),
		RefreshToken: tokens.RefreshToken,
		UserID:       info.Subject,
		Username:     info.PreferredName,
		DisplayName:  info.Name,
		Scopes:       scopes,
	}
	if err := service.store().Save(credential); err != nil {
		return Status{}, err
	}

	return statusFromCredential(credential), nil
}

func (service Service) Status(ctx context.Context, options StatusOptions) (Status, error) {
	credential, err := service.store().Load()
	if err != nil {
		if errors.Is(err, ErrNotLoggedIn) {
			return Status{LoggedIn: false}, nil
		}
		return Status{}, err
	}

	if options.Refresh {
		tokens, err := service.oauth().Refresh(ctx, credential.ClientID, credential.clientSecret(options.ClientSecret), credential.RefreshToken)
		if err != nil {
			return Status{}, err
		}
		credential.RefreshToken = tokens.RefreshToken
		if tokens.Scope != "" {
			credential.Scopes = tokens.Scope
		}
		if err := service.store().Save(credential); err != nil {
			return Status{}, err
		}
	}

	return statusFromCredential(credential), nil
}

func (service Service) AccessToken(ctx context.Context, options AccessTokenOptions) (AccessToken, error) {
	credential, err := service.store().Load()
	if err != nil {
		if errors.Is(err, ErrNotLoggedIn) {
			return AccessToken{}, ErrNotLoggedIn
		}
		return AccessToken{}, err
	}

	tokens, err := service.oauth().Refresh(ctx, credential.ClientID, credential.clientSecret(options.ClientSecret), credential.RefreshToken)
	if err != nil {
		return AccessToken{}, err
	}
	credential.RefreshToken = tokens.RefreshToken
	if tokens.Scope != "" {
		credential.Scopes = tokens.Scope
	}
	if err := service.store().Save(credential); err != nil {
		return AccessToken{}, err
	}

	return AccessToken{
		Token:      tokens.AccessToken,
		Credential: credential,
	}, nil
}

func (service Service) Logout() error {
	err := service.store().Delete()
	if errors.Is(err, ErrNotLoggedIn) {
		return nil
	}
	return err
}

func (service Service) oauth() OAuthClient {
	if service.OAuth.BaseURL != "" || service.OAuth.HTTPClient != nil {
		return service.OAuth
	}
	return OAuthClient{}
}

func (service Service) store() CredentialStore {
	if service.Store != nil {
		return service.Store
	}
	return NewCredentialStore()
}

func (service Service) callbackTimeout() time.Duration {
	if service.CallbackTimeout > 0 {
		return service.CallbackTimeout
	}
	return 5 * time.Minute
}

func (credential StoredCredential) clientSecret(override string) string {
	if strings.TrimSpace(override) != "" {
		return strings.TrimSpace(override)
	}
	return strings.TrimSpace(credential.ClientSecret)
}

func statusFromCredential(credential StoredCredential) Status {
	return Status{
		LoggedIn:    true,
		UserID:      credential.UserID,
		Username:    credential.Username,
		DisplayName: credential.DisplayName,
		Scopes:      credential.Scopes,
		ClientID:    credential.ClientID,
	}
}
