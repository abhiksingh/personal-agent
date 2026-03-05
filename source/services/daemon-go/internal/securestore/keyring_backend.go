package securestore

import (
	"errors"
	"fmt"

	keyring "github.com/zalando/go-keyring"
)

type KeyringBackend struct{}

func NewKeyringBackend() *KeyringBackend {
	return &KeyringBackend{}
}

func (b *KeyringBackend) Set(service string, account string, value string) error {
	if err := keyring.Set(service, account, value); err != nil {
		return fmt.Errorf("keyring set: %w", err)
	}
	return nil
}

func (b *KeyringBackend) Get(service string, account string) (string, error) {
	value, err := keyring.Get(service, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return "", ErrSecretNotFound
		}
		return "", fmt.Errorf("keyring get: %w", err)
	}
	return value, nil
}

func (b *KeyringBackend) Delete(service string, account string) error {
	err := keyring.Delete(service, account)
	if err != nil {
		if errors.Is(err, keyring.ErrNotFound) {
			return ErrSecretNotFound
		}
		return fmt.Errorf("keyring delete: %w", err)
	}
	return nil
}
