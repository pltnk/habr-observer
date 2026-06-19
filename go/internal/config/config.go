// Package config loads the application's configuration from OBSERVER_*
// environment variables, applying defaults and validating the result. Call
// [Load] once at startup: it reads secrets from the environment and then
// unsets them, so it is not meant to be called more than once.
package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"time"
)

// Environment variable names.
const (
	envMongoUser     = "OBSERVER_MONGO_USER"
	envMongoPass     = "OBSERVER_MONGO_PASS"
	envMongoHost     = "OBSERVER_MONGO_HOST"
	envMongoDB       = "OBSERVER_MONGO_DB"
	envMongoArticles = "OBSERVER_MONGO_ARTICLES"
	envMongoFeeds    = "OBSERVER_MONGO_FEEDS"
	envAuthToken     = "OBSERVER_AUTH_TOKEN"
	envUpdateTimeout = "OBSERVER_FEED_UPDATE_TIMEOUT"
	envCacheTTL      = "OBSERVER_FEED_CACHE_TTL"
	envServerAddr    = "OBSERVER_SERVER_ADDR"
)

// Defaults applied when the corresponding variable is unset.
const (
	defMongoUser         = "default"
	defMongoPass         = "default"
	defMongoHost         = "db"
	defMongoDB           = "observer"
	defMongoArticles     = "articles"
	defMongoFeeds        = "feeds"
	defUpdateTimeoutSecs = 600
	defCacheTTLSecs      = 60
	defServerAddr        = ":8080"
)

// MongoConfig holds the connection parts for MongoDB. Credentials are kept
// separate rather than pre-assembled into a URI; the repository hands them to
// the driver's auth options at the connection boundary, so the password never
// lands in a connection string.
type MongoConfig struct {
	Host         string
	User         string
	Password     string
	DB           string
	ArticlesColl string
	FeedsColl    string
}

// FeedRuntimeConfig holds the feed updater's timing parameters.
type FeedRuntimeConfig struct {
	UpdateTimeout time.Duration
	CacheTTL      time.Duration
}

// Config is the fully resolved application configuration.
type Config struct {
	AuthToken   string
	FeedRuntime FeedRuntimeConfig
	Mongo       MongoConfig
}

// validate reports every invariant violation at once, joined into a single
// error, so a misconfigured deployment sees all problems in one startup log.
func (c *Config) validate() error {
	var errs []error
	if c.AuthToken == "" {
		errs = append(errs, fmt.Errorf("%s must be set", envAuthToken))
	}
	if c.FeedRuntime.UpdateTimeout <= 0 {
		errs = append(errs, fmt.Errorf("%s must be positive", envUpdateTimeout))
	}
	if c.FeedRuntime.CacheTTL <= 0 {
		errs = append(errs, fmt.Errorf("%s must be positive", envCacheTTL))
	}
	return errors.Join(errs...)
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// getEnvInt returns key as an int, or fallback if unset. If the variable is
// set but not a valid integer it returns an error, so a typo fails at startup
// instead of silently using the default.
func getEnvInt(key string, fallback int) (int, error) {
	value := os.Getenv(key)
	if value == "" {
		return fallback, nil
	}
	n, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("%s: invalid integer %q: %w", key, value, err)
	}
	return n, nil
}

// getEnvOnce reads key, then removes it from the process environment so the
// secret cannot be re-read via os.Getenv or leaked through os.Environ. As a
// result Load is not idempotent for secrets: a second call sees the fallback.
func getEnvOnce(key, fallback string) string {
	value := getEnv(key, fallback)
	os.Unsetenv(key)
	return value
}

// loadMongo reads the MongoDB connection settings from the environment,
// applying defaults. It scrubs the user and password after reading (see
// [getEnvOnce]), so it must be called at most once per process.
func loadMongo() MongoConfig {
	return MongoConfig{
		Host:         getEnv(envMongoHost, defMongoHost),
		User:         getEnvOnce(envMongoUser, defMongoUser),
		Password:     getEnvOnce(envMongoPass, defMongoPass),
		DB:           getEnv(envMongoDB, defMongoDB),
		ArticlesColl: getEnv(envMongoArticles, defMongoArticles),
		FeedsColl:    getEnv(envMongoFeeds, defMongoFeeds),
	}
}

// Load reads configuration from the environment, applies defaults, and
// validates it, returning an error if a value is malformed or a required value
// is missing.
func Load() (*Config, error) {
	updateSecs, err := getEnvInt(envUpdateTimeout, defUpdateTimeoutSecs)
	if err != nil {
		return nil, err
	}
	cacheSecs, err := getEnvInt(envCacheTTL, defCacheTTLSecs)
	if err != nil {
		return nil, err
	}

	cfg := &Config{
		AuthToken: getEnvOnce(envAuthToken, ""), // no default: must be provided
		FeedRuntime: FeedRuntimeConfig{
			UpdateTimeout: time.Duration(updateSecs) * time.Second,
			CacheTTL:      time.Duration(cacheSecs) * time.Second,
		},
		Mongo: loadMongo(),
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ServerConfig is the read-side HTTP server's resolved configuration. The server
// reads feeds from MongoDB and serves them cached, so unlike the updater it
// needs no summarization token — [LoadServer] does not require OBSERVER_AUTH_TOKEN.
type ServerConfig struct {
	Addr     string
	CacheTTL time.Duration
	Mongo    MongoConfig
}

// LoadServer reads the read-side server's configuration from the environment,
// applies defaults, and validates it. Like [Load] it scrubs Mongo secrets after
// reading, so it is not meant to be called more than once.
func LoadServer() (*ServerConfig, error) {
	cacheSecs, err := getEnvInt(envCacheTTL, defCacheTTLSecs)
	if err != nil {
		return nil, err
	}

	cfg := &ServerConfig{
		Addr:     getEnv(envServerAddr, defServerAddr),
		CacheTTL: time.Duration(cacheSecs) * time.Second,
		Mongo:    loadMongo(),
	}

	if cfg.CacheTTL <= 0 {
		return nil, fmt.Errorf("%s must be positive", envCacheTTL)
	}
	return cfg, nil
}
