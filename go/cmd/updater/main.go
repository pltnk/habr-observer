// Command updater periodically refreshes Habr feeds and their AI summaries,
// writing articles and per-feed snapshots to MongoDB. It is a standalone,
// long-lived worker, separate from the read-side HTTP API.
//
// If a refresh cycle wedges past its deadline, an in-process watchdog exits the
// process so the container's restart policy revives it.
package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"habr-observer/internal/application/services/updater"
	"habr-observer/internal/application/usecases"
	"habr-observer/internal/config"
	"habr-observer/internal/infrastructure/habr"
	"habr-observer/internal/infrastructure/yagpt"
	"habr-observer/internal/repository"
)

// closeTimeout bounds how long we wait to disconnect from MongoDB on shutdown.
const closeTimeout = 10 * time.Second

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	slog.SetDefault(log)

	if err := run(log); err != nil {
		log.Error("updater exited with error", "err", err)
		os.Exit(1)
	}
	log.Info("updater stopped cleanly")
}

func run(log *slog.Logger) error {
	// Cancel the root context on SIGINT/SIGTERM for graceful shutdown.
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	feedClient := habr.NewClient(nil)

	summaryClient, err := yagpt.NewClient(cfg.AuthToken, nil)
	if err != nil {
		return fmt.Errorf("creating summary client: %w", err)
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

	updateFeed := usecases.NewUpdateFeedUsecase(feedClient, summaryClient, repo, log)
	svc := updater.NewService(updateFeed, log)

	interval := cfg.FeedRuntime.UpdateInterval
	deadline := cfg.FeedRuntime.UpdateTimeout
	// A healthy cycle finishes within deadline and the next starts within
	// interval, so exceeding twice that gap means a cycle wedged past its
	// deadline. The nil onStall makes the watchdog exit the process on a stall.
	stallTimeout := 2 * (interval + deadline)
	runner := updater.NewRunner(svc, interval, deadline, stallTimeout, log, nil)

	log.Info("updater started",
		"interval", interval.String(),
		"deadline", deadline.String(),
		"stall_timeout", stallTimeout.String(),
		"mongo_db", cfg.Mongo.DB,
	)
	runner.Run(ctx)
	return nil
}
