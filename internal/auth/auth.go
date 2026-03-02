package auth

import (
	"crypto/subtle"
	"encoding/json"
	"errors"
)

type Authenticator interface {
	Validate(appID, writeKey string) bool
}

type StaticAuthenticator struct {
	// appID -> writeKey (PoC: stored in memory as plain text).
	keys map[string]string
}

func NewStaticAuthenticator(appKeysJSON string) (*StaticAuthenticator, error) {
	if appKeysJSON == "" {
		return nil, errors.New("empty app keys json")
	}

	m := map[string]string{}
	if err := json.Unmarshal([]byte(appKeysJSON), &m); err != nil {
		return nil, err
	}

	if len(m) == 0 {
		return nil, errors.New("no app keys provided")
	}

	return &StaticAuthenticator{keys: m}, nil
}

func (a *StaticAuthenticator) Validate(appID, writeKey string) bool {
	expected, ok := a.keys[appID]
	if !ok {
		return false
	}
	if len(expected) != len(writeKey) {
		return false
	}
	return subtle.ConstantTimeCompare(
		[]byte(expected),
		[]byte(writeKey),
	) == 1
}
