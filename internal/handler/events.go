package handler

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/52-entertainment/52-sdk-event-collector/internal/auth"
	"github.com/52-entertainment/52-sdk-event-collector/internal/pubsub"
)

type EventsDeps struct {
	Authenticator     auth.Authenticator
	Publisher         pubsub.Publisher
	MaxBodyBytes      int64
	MaxUnzippedBytes  int64
	MaxEventsPerBatch int
}

type eventsHandler struct {
	deps EventsDeps
}

type eventsRequest struct {
	Events []event `json:"events"`
}

type event struct {
	EventID    string                 `json:"event_id"`
	EventName  string                 `json:"event_name"`
	ClientTime string                 `json:"client_time,omitempty"`
	DeviceID   string                 `json:"device_id,omitempty"`
	UserID     string                 `json:"user_id,omitempty"`
	Properties map[string]any         `json:"properties,omitempty"`
	Meta       map[string]any         `json:"meta,omitempty"`
}

type ingestEnvelope struct {
	AppID      string   `json:"app_id"`
	ReceivedAt string   `json:"received_at"`
	RequestID  string   `json:"request_id"`
	Events     []event  `json:"events"`
}

type eventsResponse struct {
	RequestID string `json:"request_id"`
	Accepted  int    `json:"accepted"`
}

func NewEventsHandler(deps EventsDeps) http.Handler {
	return &eventsHandler{deps: deps}
}

func (h *eventsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{
			"error": "method_not_allowed",
		})
		return
	}

	appID := r.Header.Get("X-App-Id")
	writeKey := r.Header.Get("X-Write-Key")
	if appID == "" || writeKey == "" {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "missing_credentials",
		})
		return
	}
	if !h.deps.Authenticator.Validate(appID, writeKey) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{
			"error": "invalid_credentials",
		})
		return
	}

	reqID := requestIDFromHeaderOrNew(r)
	now := time.Now().UTC().Format(time.RFC3339Nano)

	body, err := readBody(r, h.deps.MaxBodyBytes, h.deps.MaxUnzippedBytes)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": err.Error(),
		})
		return
	}

	var req eventsRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "invalid_json",
		})
		return
	}

	if len(req.Events) == 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{
			"error": "no_events",
		})
		return
	}
	if h.deps.MaxEventsPerBatch > 0 &&
		len(req.Events) > h.deps.MaxEventsPerBatch {
		writeJSON(w, http.StatusRequestEntityTooLarge, map[string]string{
			"error": "too_many_events",
		})
		return
	}

	for i := range req.Events {
		if strings.TrimSpace(req.Events[i].EventID) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "missing_event_id",
			})
			return
		}
		if strings.TrimSpace(req.Events[i].EventName) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{
				"error": "missing_event_name",
			})
			return
		}
	}

	env := ingestEnvelope{
		AppID:      appID,
		ReceivedAt: now,
		RequestID:  reqID,
		Events:     req.Events,
	}

	data, err := json.Marshal(env)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{
			"error": "marshal_failed",
		})
		return
	}

	attrs := map[string]string{
		"app_id":      appID,
		"received_at": now,
		"request_id":  reqID,
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := h.deps.Publisher.Publish(ctx, data, attrs); err != nil {
		writeJSON(w, http.StatusServiceUnavailable, map[string]string{
			"error": "pubsub_publish_failed",
		})
		return
	}

	writeJSON(w, http.StatusAccepted, eventsResponse{
		RequestID: reqID,
		Accepted:  len(req.Events),
	})
}

func readBody(
	r *http.Request,
	maxBodyBytes int64,
	maxUnzippedBytes int64,
) ([]byte, error) {
	r.Body = http.MaxBytesReader(nil, r.Body, maxBodyBytes)
	defer r.Body.Close()

	var reader io.Reader = r.Body

	if strings.Contains(strings.ToLower(r.Header.Get("Content-Encoding")), "gzip") {
		gz, err := gzip.NewReader(r.Body)
		if err != nil {
			return nil, errInvalid("invalid_gzip")
		}
		defer gz.Close()
		reader = gz
	}

	if maxUnzippedBytes > 0 {
		reader = io.LimitReader(reader, maxUnzippedBytes)
	}

	b, err := io.ReadAll(reader)
	if err != nil {
		return nil, errInvalid("invalid_body")
	}

	if len(bytes.TrimSpace(b)) == 0 {
		return nil, errInvalid("empty_body")
	}
	return b, nil
}

type invalidErr string

func (e invalidErr) Error() string { return string(e) }

func errInvalid(msg string) error { return invalidErr(msg) }

func requestIDFromHeaderOrNew(r *http.Request) string {
	if v := strings.TrimSpace(r.Header.Get("X-Request-Id")); v != "" {
		return v
	}
	if v := strings.TrimSpace(r.Header.Get("X-Cloud-Trace-Context")); v != "" {
		// Format: TRACE_ID/SPAN_ID;o=TRACE_TRUE
		parts := strings.SplitN(v, "/", 2)
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}
	return newHexID(16)
}

func newHexID(nBytes int) string {
	b := make([]byte, nBytes)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
