// Package delivery contains the HTTP handlers that expose the read-side API to
// the frontend.
package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"habr-observer/internal/domain"
)

// FeedsGetter supplies the feeds to serve. *usecases.GetFeedsUsecase satisfies
// it; the handler depends on the interface so its tests can drive it with a fake.
type FeedsGetter interface {
	Execute(ctx context.Context) ([]*domain.Feed, error)
}

// GetFeedsHandler serves every feed — in canonical order, with embedded articles
// and their summaries — as a single JSON array.
//
// It implements [http.Handler]; register it under a GET route (e.g.
// "GET /feeds") so the router rejects other methods.
type GetFeedsHandler struct {
	getter FeedsGetter
	log    *slog.Logger
}

// NewGetFeedsHandler returns a handler backed by getter. If log is nil,
// [slog.Default] is used.
func NewGetFeedsHandler(getter FeedsGetter, log *slog.Logger) *GetFeedsHandler {
	if log == nil {
		log = slog.Default()
	}
	return &GetFeedsHandler{getter: getter, log: log}
}

// ServeHTTP writes the feeds as a JSON array. A failure yields a clean 500,
// never a half-written 200; a nil feed set is encoded as [] (not null) so
// clients always receive an array.
func (h *GetFeedsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	feeds, err := h.getter.Execute(r.Context())
	if err != nil {
		if !errors.Is(err, context.Canceled) { // a canceled request is the client's doing, not a failure
			h.log.Error("loading feeds for response", "err", err)
		}
		http.Error(w, "failed to load feeds", http.StatusInternalServerError)
		return
	}

	if feeds == nil {
		feeds = []*domain.Feed{}
	}

	// Marshal up front so an encoding failure can still yield a clean 500.
	body, err := json.Marshal(feeds)
	if err != nil {
		h.log.Error("encoding feeds response", "err", err)
		http.Error(w, "failed to encode feeds", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if _, err := w.Write(body); err != nil {
		// The client most likely disconnected; the status is already sent, so
		// just record it.
		h.log.Warn("writing feeds response", "err", err)
	}
}
