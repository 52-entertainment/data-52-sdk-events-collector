package firestore

import (
	"context"
	"strings"

	gcpfirestore "cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type AppsRepository interface {
	GetWriteKey(ctx context.Context, appID string) (string, error)
}

type FirestoreAppsRepository struct {
	getter appsDocumentGetter
}

type appDocument struct {
	WriteKey string `firestore:"write_key"`
}

type appsDocumentGetter interface {
	Get(ctx context.Context, appID string, dest any) error
}

type collectionAppsDocumentGetter struct {
	collection *gcpfirestore.CollectionRef
}

func (g collectionAppsDocumentGetter) Get(
	ctx context.Context,
	appID string,
	dest any,
) error {
	snap, err := g.collection.Doc(appID).Get(ctx)
	if err != nil {
		return err
	}
	return snap.DataTo(dest)
}

func (r *FirestoreAppsRepository) GetWriteKey(
	ctx context.Context,
	appID string,
) (string, error) {
	if strings.TrimSpace(appID) == "" {
		return "", nil
	}

	var doc appDocument
	if err := r.getter.Get(ctx, appID, &doc); err != nil {
		if status.Code(err) == codes.NotFound {
			return "", nil
		}
		return "", err
	}

	return strings.TrimSpace(doc.WriteKey), nil
}
