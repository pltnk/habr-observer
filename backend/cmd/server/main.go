// Command server exposes the read-side HTTP API: a single GET /feeds endpoint
// returning every Habr feed — in canonical order, with articles and their
// summaries — as one JSON array for the frontend. It reads from MongoDB
// through an in-memory, TTL-cached use case and runs separately from the updater
// worker that writes the data.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"habr-observer/internal/application/usecases"
	"habr-observer/internal/config"
	"habr-observer/internal/delivery"
	"habr-observer/internal/infrastructure/habr"
	"habr-observer/internal/repository"
)

const (
	// addr is the fixed container-internal listen address; the Dockerfile's
	// EXPOSE, the compose port mapping, and the frontend's default backend URL
	// all assume it.
	addr = ":8080"

	// closeTimeout bounds how long we wait to disconnect from MongoDB on shutdown.
	closeTimeout = 10 * time.Second
	// shutdownTimeout bounds draining in-flight HTTP requests on shutdown.
	shutdownTimeout = 15 * time.Second

	// HTTP server timeouts guard against slow or stuck clients.
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("server exited with error", "err", err)
		os.Exit(1)
	}
	log.Info("server stopped cleanly")
}

func run(log *slog.Logger) error {
	// Cancel the root context on SIGINT/SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadServer()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	repo, err := repository.NewMongoStorage(ctx, cfg.Mongo)
	if err != nil {
		return fmt.Errorf("connecting to repository: %w", err)
	}
	defer func() {
		// The root ctx is already canceled during shutdown, so disconnect on a
		// fresh, bounded context.
		closeCtx, cancel := context.WithTimeout(context.Background(), closeTimeout)
		defer cancel()
		if cerr := repo.Close(closeCtx); cerr != nil {
			log.Error("closing repository", "err", cerr)
		}
	}()

	getFeeds := usecases.NewGetFeedsUsecase(repo, feedURLs(), cfg.CacheTTL, log)
	handler := delivery.NewGetFeedsHandler(getFeeds, log)

	srv := &http.Server{
		Addr:              addr,
		Handler:           routes(handler),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	log.Info("server starting", "addr", addr, "cache_ttl", cfg.CacheTTL.String(), "mongo_db", cfg.Mongo.DB)
	return serve(ctx, srv, log)
}

// routes builds the read API's request multiplexer. GET /feeds is the only
// endpoint; the method-scoped pattern makes the mux answer other methods with
// 405 and unknown paths with 404.
func routes(feeds http.Handler) *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("GET /feeds", feeds)
	return mux
}

// feedURLs returns every Habr feed URL in canonical (display) order — the _ids
// the updater writes and the read use case queries.
func feedURLs() []string {
	feeds := habr.AllFeeds()
	ids := make([]string, len(feeds))
	for i, f := range feeds {
		ids[i] = f.URL()
	}
	return ids
}

// serve runs srv until ctx is canceled, then drains in-flight requests within
// shutdownTimeout. It returns nil on a clean shutdown and an error only if
// serving or the graceful shutdown fails.
func serve(ctx context.Context, srv *http.Server, log *slog.Logger) error {
	errCh := make(chan error, 1)
	go func() { errCh <- srv.ListenAndServe() }()

	select {
	case err := <-errCh:
		if err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("http server: %w", err)
		}
		return nil
	case <-ctx.Done():
		log.Info("shutdown signal received; draining requests")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("graceful shutdown: %w", err)
		}
		return nil
	}
}
