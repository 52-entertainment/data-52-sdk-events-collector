package config

import "testing"

func setRequiredEnv(t *testing.T) {
	t.Helper()
	t.Setenv("GOOGLE_CLOUD_PROJECT", "fft-tmp-raw")
	t.Setenv("PUBSUB_TOPIC", "sdk-events-ingest")
	t.Setenv("FIRESTORE_DATABASE", "collector-smoke")
}

func TestFromEnvRequiresFirestoreDatabase(t *testing.T) {
	t.Setenv("GOOGLE_CLOUD_PROJECT", "fft-tmp-raw")
	t.Setenv("PUBSUB_TOPIC", "sdk-events-ingest")

	_, err := FromEnv()
	if err == nil || err.Error() != "missing FIRESTORE_DATABASE" {
		t.Fatalf("FromEnv() error = %v, want missing FIRESTORE_DATABASE", err)
	}
}

func TestFromEnvDefaultsAppsCollection(t *testing.T) {
	setRequiredEnv(t)

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}

	if cfg.FirestoreDatabaseID != "collector-smoke" {
		t.Fatalf(
			"FromEnv() FirestoreDatabaseID = %q, want %q",
			cfg.FirestoreDatabaseID,
			"collector-smoke",
		)
	}

	if cfg.FirestoreAppsCollection != "apps" {
		t.Fatalf(
			"FromEnv() FirestoreAppsCollection = %q, want %q",
			cfg.FirestoreAppsCollection,
			"apps",
		)
	}
}

func TestFromEnvOverridesAppsCollection(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("FIRESTORE_DATABASE", "another-db")
	t.Setenv("FIRESTORE_APPS_COLLECTION", "sdk-apps")

	cfg, err := FromEnv()
	if err != nil {
		t.Fatalf("FromEnv() error = %v", err)
	}

	if cfg.FirestoreDatabaseID != "another-db" {
		t.Fatalf(
			"FromEnv() FirestoreDatabaseID = %q, want %q",
			cfg.FirestoreDatabaseID,
			"another-db",
		)
	}

	if cfg.FirestoreAppsCollection != "sdk-apps" {
		t.Fatalf(
			"FromEnv() FirestoreAppsCollection = %q, want %q",
			cfg.FirestoreAppsCollection,
			"sdk-apps",
		)
	}
}

func TestFromEnvDoesNotRequireAppKeysJSON(t *testing.T) {
	setRequiredEnv(t)
	t.Setenv("APP_KEYS_JSON", "")

	if _, err := FromEnv(); err != nil {
		t.Fatalf("FromEnv() error = %v, want nil", err)
	}
}
