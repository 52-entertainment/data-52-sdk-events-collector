package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type stubAuthenticator struct {
	ok  bool
	err error
}

func (a stubAuthenticator) Validate(
	_ context.Context,
	_ string,
	_ string,
) (bool, error) {
	return a.ok, a.err
}

type stubPublisher struct {
	err         error
	publishCall int
}

func (p *stubPublisher) Publish(
	_ context.Context,
	_ []byte,
	_ map[string]string,
) error {
	p.publishCall++
	return p.err
}

func (p *stubPublisher) Close() error {
	return nil
}

func TestEventsHandlerMissingCredentials(t *testing.T) {
	t.Parallel()

	handler := NewEventsHandler(EventsDeps{
		Authenticator: stubAuthenticator{},
		Publisher:     &stubPublisher{},
		MaxBodyBytes:  1024,
	})

	req := httptest.NewRequest(http.MethodPost, "/v1/events", bytes.NewBufferString(`{}`))
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assertJSONError(t, res, http.StatusUnauthorized, "missing_credentials")
}

func TestEventsHandlerInvalidCredentials(t *testing.T) {
	t.Parallel()

	handler := NewEventsHandler(EventsDeps{
		Authenticator: stubAuthenticator{ok: false},
		Publisher:     &stubPublisher{},
		MaxBodyBytes:  1024,
	})

	req := newEventsRequest(t)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assertJSONError(t, res, http.StatusUnauthorized, "invalid_credentials")
}

func TestEventsHandlerAuthBackendUnavailable(t *testing.T) {
	t.Parallel()

	handler := NewEventsHandler(EventsDeps{
		Authenticator: stubAuthenticator{err: errors.New("firestore down")},
		Publisher:     &stubPublisher{},
		MaxBodyBytes:  1024,
	})

	req := newEventsRequest(t)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assertJSONError(t, res, http.StatusServiceUnavailable, "auth_backend_unavailable")
}

func TestEventsHandlerPublishesWhenAuthorized(t *testing.T) {
	t.Parallel()

	publisher := &stubPublisher{}
	handler := NewEventsHandler(EventsDeps{
		Authenticator:     stubAuthenticator{ok: true},
		Publisher:         publisher,
		MaxBodyBytes:      1024,
		MaxUnzippedBytes:  1024,
		MaxEventsPerBatch: 10,
	})

	req := newEventsRequest(t)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("ServeHTTP() status = %d, want %d", res.Code, http.StatusAccepted)
	}
	if publisher.publishCall != 1 {
		t.Fatalf("Publish() calls = %d, want %d", publisher.publishCall, 1)
	}
}

func TestEventsHandlerPublisherFailure(t *testing.T) {
	t.Parallel()

	handler := NewEventsHandler(EventsDeps{
		Authenticator:     stubAuthenticator{ok: true},
		Publisher:         &stubPublisher{err: errors.New("pubsub down")},
		MaxBodyBytes:      1024,
		MaxUnzippedBytes:  1024,
		MaxEventsPerBatch: 10,
	})

	req := newEventsRequest(t)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	assertJSONError(t, res, http.StatusServiceUnavailable, "pubsub_publish_failed")
}

func newEventsRequest(t *testing.T) *http.Request {
	t.Helper()

	body := bytes.NewBufferString(
		`{"events":[{"event_id":"evt_1","event_name":"session_started"}]}`,
	)
	req := httptest.NewRequest(http.MethodPost, "/v1/events", body)
	req.Header.Set("X-App-Id", "test_app")
	req.Header.Set("X-Write-Key", "test_token")
	return req
}

func assertJSONError(
	t *testing.T,
	res *httptest.ResponseRecorder,
	wantStatus int,
	wantError string,
) {
	t.Helper()

	if res.Code != wantStatus {
		t.Fatalf("ServeHTTP() status = %d, want %d", res.Code, wantStatus)
	}

	var payload map[string]string
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("response json error = %v", err)
	}

	if payload["error"] != wantError {
		t.Fatalf("response error = %q, want %q", payload["error"], wantError)
	}
}
