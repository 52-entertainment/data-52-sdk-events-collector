package firestore

import (
	"context"
	"errors"

	gcpfirestore "cloud.google.com/go/firestore"
)

const defaultAppsCollection = "apps"

type Config struct {
	AppsCollection string
	DatabaseID     string
}

type Store interface {
	Apps() AppsRepository
	Close() error
}

type FirestoreStore struct {
	client *gcpfirestore.Client
	apps   AppsRepository
}

func (c Config) withDefaults() Config {
	if c.AppsCollection == "" {
		c.AppsCollection = defaultAppsCollection
	}
	return c
}

func NewStore(
	ctx context.Context,
	projectID string,
	cfg Config,
) (*FirestoreStore, error) {
	if projectID == "" {
		return nil, errors.New("empty projectID")
	}
	if cfg.DatabaseID == "" {
		return nil, errors.New("empty databaseID")
	}

	cfg = cfg.withDefaults()

	client, err := gcpfirestore.NewClientWithDatabase(
		ctx,
		projectID,
		cfg.DatabaseID,
	)
	if err != nil {
		return nil, err
	}

	return &FirestoreStore{
		client: client,
		apps: &FirestoreAppsRepository{
			getter: collectionAppsDocumentGetter{
				collection: client.Collection(cfg.AppsCollection),
			},
		},
	}, nil
}

func (s *FirestoreStore) Apps() AppsRepository {
	return s.apps
}

func (s *FirestoreStore) Close() error {
	return s.client.Close()
}
