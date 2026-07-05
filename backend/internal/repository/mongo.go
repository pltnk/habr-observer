// Package repository provides MongoDB-backed persistence for articles and
// feeds. It is a pure data-access adapter: it stores and retrieves domain
// values and adds no caching, batching, or business logic of its own.
package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/v2/mongo"
	"go.mongodb.org/mongo-driver/v2/mongo/options"
	"go.mongodb.org/mongo-driver/v2/mongo/readpref"

	"habr-observer/internal/config"
)

const (
	// serverSelectionTimeout bounds how long the driver waits to find a
	// suitable server before failing an operation.
	serverSelectionTimeout = 10 * time.Second

	// defaultOpTimeout is the per-operation deadline applied when the caller's
	// context carries none.
	defaultOpTimeout = 30 * time.Second
)

// ErrMongoStorageCreation wraps any failure to create a MongoStorage — an
// invalid client config or a failed connectivity Ping. The cause is wrapped, so
// errors.Is(err, ErrMongoStorageCreation) matches while the cause stays inspectable.
var ErrMongoStorageCreation = errors.New("repository: creating mongo storage")

// MongoStorage is a MongoDB-backed store for articles and feeds. It is safe
// for concurrent use by multiple goroutines.
//
// A MongoStorage must be created with [NewMongoStorage]; the zero value is not
// usable. The caller owns the returned value and must call [MongoStorage.Close]
// when done.
type MongoStorage struct {
	client   *mongo.Client
	articles *mongo.Collection
	feeds    *mongo.Collection
}

// NewMongoStorage connects to MongoDB using cfg, verifies the connection with a
// Ping against the primary, and returns a store bound to its database and
// collection names. Credentials are passed to the driver's auth options rather
// than embedded in a connection string, so the password never lands in a URI.
// The context bounds the initial Ping only; it does not bound the lifetime of
// the returned client. All failures wrap [ErrMongoStorageCreation].
func NewMongoStorage(ctx context.Context, cfg config.MongoConfig) (*MongoStorage, error) {
	// In driver v2, Connect performs no I/O and takes no context; the Ping
	// below is what actually verifies connectivity. AuthSource "admin" matches
	// the root user created by the db container.
	client, err := mongo.Connect(options.Client().
		ApplyURI("mongodb://" + cfg.Host).
		SetAuth(options.Credential{AuthSource: "admin", Username: cfg.User, Password: cfg.Password}).
		SetServerSelectionTimeout(serverSelectionTimeout).
		SetTimeout(defaultOpTimeout))
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrMongoStorageCreation, err)
	}

	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		_ = client.Disconnect(context.Background())
		return nil, fmt.Errorf("%w: %w", ErrMongoStorageCreation, err)
	}

	database := client.Database(cfg.DB)
	return &MongoStorage{
		client:   client,
		articles: database.Collection(cfg.ArticlesColl),
		feeds:    database.Collection(cfg.FeedsColl),
	}, nil
}

// Close disconnects from MongoDB, releasing pooled connections. The context
// bounds how long Close waits for in-progress operations to finish.
func (m *MongoStorage) Close(ctx context.Context) error {
	if err := m.client.Disconnect(ctx); err != nil {
		return fmt.Errorf("repository: disconnecting from mongo: %w", err)
	}
	return nil
}
