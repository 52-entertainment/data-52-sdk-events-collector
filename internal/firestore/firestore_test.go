package firestore

import (
	"context"
	"errors"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type stubAppsDocumentGetter struct {
	writeKey string
	err      error
}

func (g stubAppsDocumentGetter) Get(
	_ context.Context,
	_ string,
	dest any,
) error {
	if g.err != nil {
		return g.err
	}

	doc := dest.(*appDocument)
	doc.WriteKey = g.writeKey
	return nil
}

func TestConfigWithDefaults(t *testing.T) {
	t.Parallel()

	if got := (Config{}).withDefaults(); got.AppsCollection != "apps" {
		t.Fatalf("withDefaults() AppsCollection = %q, want %q", got.AppsCollection, "apps")
	}

	got := (Config{
		DatabaseID:     "collector-smoke",
		AppsCollection: "sdk-apps",
	}).withDefaults()
	if got.DatabaseID != "collector-smoke" {
		t.Fatalf("withDefaults() DatabaseID = %q, want %q", got.DatabaseID, "collector-smoke")
	}
	if got.AppsCollection != "sdk-apps" {
		t.Fatalf("withDefaults() AppsCollection = %q, want %q", got.AppsCollection, "sdk-apps")
	}
}

func TestNewStoreRejectsEmptyProjectID(t *testing.T) {
	t.Parallel()

	_, err := NewStore(context.Background(), "", Config{DatabaseID: "collector-smoke"})
	if err == nil || err.Error() != "empty projectID" {
		t.Fatalf("NewStore() error = %v, want empty projectID", err)
	}
}

func TestNewStoreRejectsEmptyDatabaseID(t *testing.T) {
	t.Parallel()

	_, err := NewStore(context.Background(), "fft-tmp-raw", Config{})
	if err == nil || err.Error() != "empty databaseID" {
		t.Fatalf("NewStore() error = %v, want empty databaseID", err)
	}
}

func TestFirestoreAppsRepositoryGetWriteKey(t *testing.T) {
	t.Parallel()

	backendErr := errors.New("backend down")

	tests := []struct {
		name    string
		getter  stubAppsDocumentGetter
		wantKey string
		wantErr error
	}{
		{
			name:    "returns write key",
			getter:  stubAppsDocumentGetter{writeKey: "secret"},
			wantKey: "secret",
		},
		{
			name:    "trims write key",
			getter:  stubAppsDocumentGetter{writeKey: "  secret  "},
			wantKey: "secret",
		},
		{
			name:    "not found becomes invalid credentials",
			getter:  stubAppsDocumentGetter{err: status.Error(codes.NotFound, "missing")},
			wantKey: "",
		},
		{
			name:    "backend error propagates",
			getter:  stubAppsDocumentGetter{err: backendErr},
			wantKey: "",
			wantErr: backendErr,
		},
		{
			name:    "blank app id is invalid credentials",
			getter:  stubAppsDocumentGetter{writeKey: "secret"},
			wantKey: "",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			repo := &FirestoreAppsRepository{getter: tc.getter}

			appID := "app_1"
			if tc.name == "blank app id is invalid credentials" {
				appID = " "
			}

			gotKey, err := repo.GetWriteKey(context.Background(), appID)

			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("GetWriteKey() error = %v, want %v", err, tc.wantErr)
			}
			if gotKey != tc.wantKey {
				t.Fatalf("GetWriteKey() = %q, want %q", gotKey, tc.wantKey)
			}
		})
	}
}
