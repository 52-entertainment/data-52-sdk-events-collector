package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/52-entertainment/52-sdk-event-collector/internal/auth"
	"github.com/52-entertainment/52-sdk-event-collector/internal/config"
	firestorestore "github.com/52-entertainment/52-sdk-event-collector/internal/firestore"
	"github.com/52-entertainment/52-sdk-event-collector/internal/handler"
	"github.com/52-entertainment/52-sdk-event-collector/internal/pubsub"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	ctx := context.Background()

	store, err := firestorestore.NewStore(ctx, cfg.GCPProjectID, firestorestore.Config{
		AppsCollection: cfg.FirestoreAppsCollection,
		DatabaseID:     cfg.FirestoreDatabaseID,
	})
	if err != nil {
		log.Fatalf("firestore init error: %v", err)
	}
	defer store.Close()

	authenticator := auth.NewStoreAuthenticator(store.Apps())

	publisher, err := pubsub.NewPublisher(ctx, cfg.GCPProjectID, cfg.PubSubTopic)
	if err != nil {
		log.Fatalf("pubsub init error: %v", err)
	}
	defer publisher.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", handler.Healthz)
	mux.Handle("/v1/events", handler.NewEventsHandler(handler.EventsDeps{
		Authenticator:     authenticator,
		Publisher:         publisher,
		MaxBodyBytes:      cfg.MaxBodyBytes,
		MaxUnzippedBytes:  cfg.MaxUnzippedBytes,
		MaxEventsPerBatch: cfg.MaxEventsPerBatch,
	}))

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           withRequestTimeout(mux, cfg.RequestTimeout),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", srv.Addr)
		if err := srv.ListenAndServe(); err != nil &&
			!errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Cloud Run sends SIGTERM; honor it for graceful shutdown.
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(
		context.Background(),
		10*time.Second,
	)
	defer cancel()

	_ = srv.Shutdown(shutdownCtx)
	log.Printf("shutdown complete")
}

func withRequestTimeout(next http.Handler, d time.Duration) http.Handler {
	if d <= 0 {
		return next
	}
	return http.TimeoutHandler(next, d, `{"error":"timeout"}`)
}
