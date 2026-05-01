package auth

import (
	"encoding/json"
	"errors"
	"fmt"
)

const credentialTarget = "ERLCX/RobloxOAuth"

var ErrNotLoggedIn = errors.New("not logged in")

type StoredCredential struct {
	ClientID     string   `json:"clientId"`
	ClientSecret string   `json:"clientSecret,omitempty"`
	RefreshToken string   `json:"refreshToken"`
	UserID       string   `json:"userId"`
	Username     string   `json:"username"`
	DisplayName  string   `json:"displayName"`
	Scopes       string   `json:"scopes"`
	Resources    []string `json:"resources,omitempty"`
}

type CredentialStore interface {
	Save(StoredCredential) error
	Load() (StoredCredential, error)
	Delete() error
}

type WindowsCredentialStore struct {
	Target string
}

func NewCredentialStore() CredentialStore {
	return WindowsCredentialStore{Target: credentialTarget}
}

func (store WindowsCredentialStore) target() string {
	if store.Target != "" {
		return store.Target
	}
	return credentialTarget
}

func encodeCredential(credential StoredCredential) ([]byte, error) {
	if credential.ClientID == "" {
		return nil, fmt.Errorf("client ID must not be empty")
	}
	if credential.RefreshToken == "" {
		return nil, fmt.Errorf("refresh token must not be empty")
	}
	return json.Marshal(credential)
}

func decodeCredential(data []byte) (StoredCredential, error) {
	var credential StoredCredential
	if err := json.Unmarshal(data, &credential); err != nil {
		return StoredCredential{}, fmt.Errorf("decode stored credential: %w", err)
	}
	if credential.ClientID == "" || credential.RefreshToken == "" {
		return StoredCredential{}, fmt.Errorf("stored credential is incomplete")
	}
	return credential, nil
}
