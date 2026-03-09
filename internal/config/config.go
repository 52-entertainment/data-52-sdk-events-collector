package config

import (
	"errors"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Port                    string
	GCPProjectID            string
	PubSubTopic             string
	FirestoreDatabaseID     string
	FirestoreAppsCollection string
	MaxBodyBytes            int64
	MaxUnzippedBytes        int64
	MaxEventsPerBatch       int
	RequestTimeout          time.Duration
}

func FromEnv() (Config, error) {
	port := getenv("PORT", "8080")

	projectID := os.Getenv("GOOGLE_CLOUD_PROJECT")
	if projectID == "" {
		projectID = os.Getenv("GCP_PROJECT")
	}
	if projectID == "" {
		return Config{}, errors.New(
			"missing GOOGLE_CLOUD_PROJECT (or GCP_PROJECT)",
		)
	}

	topic := os.Getenv("PUBSUB_TOPIC")
	if topic == "" {
		return Config{}, errors.New("missing PUBSUB_TOPIC")
	}

	databaseID := os.Getenv("FIRESTORE_DATABASE")
	if databaseID == "" {
		return Config{}, errors.New("missing FIRESTORE_DATABASE")
	}
	appsCollection := getenv("FIRESTORE_APPS_COLLECTION", "apps")

	maxBodyBytes := getenvInt64("MAX_BODY_BYTES", 1_048_576)         // 1 MiB
	maxUnzippedBytes := getenvInt64("MAX_UNZIPPED_BYTES", 4_194_304) // 4 MiB
	maxEvents := getenvInt("MAX_EVENTS_PER_BATCH", 200)

	timeout := getenvDuration("REQUEST_TIMEOUT", 10*time.Second)

	return Config{
		Port:                    port,
		GCPProjectID:            projectID,
		PubSubTopic:             topic,
		FirestoreDatabaseID:     databaseID,
		FirestoreAppsCollection: appsCollection,
		MaxBodyBytes:            maxBodyBytes,
		MaxUnzippedBytes:        maxUnzippedBytes,
		MaxEventsPerBatch:       maxEvents,
		RequestTimeout:          timeout,
	}, nil
}

func getenv(k, def string) string {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	return v
}

func getenvInt(k string, def int) int {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

func getenvInt64(k string, def int64) int64 {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	n, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return def
	}
	return n
}

func getenvDuration(k string, def time.Duration) time.Duration {
	v := os.Getenv(k)
	if v == "" {
		return def
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return def
	}
	return d
}
