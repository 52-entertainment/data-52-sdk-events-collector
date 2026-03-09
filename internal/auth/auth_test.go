package auth

import (
	"context"
	"errors"
	"testing"
)

type stubCredentialsStore struct {
	writeKey string
	err      error
}

func (s stubCredentialsStore) GetWriteKey(
	_ context.Context,
	_ string,
) (string, error) {
	return s.writeKey, s.err
}

func TestStoreAuthenticatorValidate(t *testing.T) {
	t.Parallel()

	storeErr := errors.New("boom")

	tests := []struct {
		name     string
		store    stubCredentialsStore
		writeKey string
		wantOK   bool
		wantErr  error
	}{
		{
			name:     "valid credentials",
			store:    stubCredentialsStore{writeKey: "secret"},
			writeKey: "secret",
			wantOK:   true,
		},
		{
			name:     "unknown app",
			store:    stubCredentialsStore{},
			writeKey: "secret",
			wantOK:   false,
		},
		{
			name:     "wrong write key",
			store:    stubCredentialsStore{writeKey: "secret"},
			writeKey: "wrong1",
			wantOK:   false,
		},
		{
			name:     "length mismatch",
			store:    stubCredentialsStore{writeKey: "secret"},
			writeKey: "nope",
			wantOK:   false,
		},
		{
			name:     "store error",
			store:    stubCredentialsStore{err: storeErr},
			writeKey: "secret",
			wantOK:   false,
			wantErr:  storeErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			authenticator := NewStoreAuthenticator(tc.store)

			gotOK, err := authenticator.Validate(
				context.Background(),
				"app_1",
				tc.writeKey,
			)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("Validate() error = %v, want %v", err, tc.wantErr)
			}
			if gotOK != tc.wantOK {
				t.Fatalf("Validate() ok = %v, want %v", gotOK, tc.wantOK)
			}
		})
	}
}
