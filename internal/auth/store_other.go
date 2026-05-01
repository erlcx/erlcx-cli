//go:build !windows

package auth

import "fmt"

func (store WindowsCredentialStore) Save(credential StoredCredential) error {
	return fmt.Errorf("secure credential storage is only implemented on Windows")
}

func (store WindowsCredentialStore) Load() (StoredCredential, error) {
	return StoredCredential{}, fmt.Errorf("secure credential storage is only implemented on Windows: %w", ErrNotLoggedIn)
}

func (store WindowsCredentialStore) Delete() error {
	return fmt.Errorf("secure credential storage is only implemented on Windows: %w", ErrNotLoggedIn)
}
