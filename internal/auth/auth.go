package auth

import (
	"context"
	"crypto/subtle"
)

type Authenticator interface {
	Validate(ctx context.Context, appID, writeKey string) (bool, error)
}

type CredentialsStore interface {
	GetWriteKey(ctx context.Context, appID string) (string, error)
}

type StoreAuthenticator struct {
	store CredentialsStore
}

func NewStoreAuthenticator(
	store CredentialsStore,
) *StoreAuthenticator {
	return &StoreAuthenticator{store: store}
}

func (a *StoreAuthenticator) Validate(
	ctx context.Context,
	appID, writeKey string,
) (bool, error) {
	expected, err := a.store.GetWriteKey(ctx, appID)
	if err != nil {
		return false, err
	}
	if expected == "" {
		return false, nil
	}
	if len(expected) != len(writeKey) {
		return false, nil
	}
	return subtle.ConstantTimeCompare(
		[]byte(expected),
		[]byte(writeKey),
	) == 1, nil
}
