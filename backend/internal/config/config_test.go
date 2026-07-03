package config

import (
	"os"
	"reflect"
	"strings"
	"testing"
	"time"
)

// allEnvKeys is every variable the loaders read. setEnv neutralizes all of them
// so a test is hermetic regardless of the developer's real environment.
var allEnvKeys = []string{
	envMongoUser, envMongoPass, envMongoHost, envMongoDB, envMongoArticles,
	envMongoFeeds, envAuthToken, envUpdateInterval, envUpdateTimeout, envCacheTTL,
	envServerAddr,
}

// setEnv sets every OBSERVER_* variable for the duration of the test (restored
// automatically by t.Setenv), taking each value from env, or "" — treated as
// unset — when a key is absent.
//
// Tests using setEnv must NOT call t.Parallel: they mutate the process-global
// environment, and t.Setenv panics if t.Parallel was called.
func setEnv(t *testing.T, env map[string]string) {
	t.Helper()
	for _, key := range allEnvKeys {
		t.Setenv(key, env[key])
	}
}

func TestLoad_Defaults(t *testing.T) {
	// Only the required variable is set; everything else falls back to defaults.
	setEnv(t, map[string]string{envAuthToken: "tok"})

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := &Config{
		AuthToken: "tok",
		FeedRuntime: FeedRuntimeConfig{
			UpdateInterval: defUpdateIntervalSecs * time.Second,
			UpdateTimeout:  defUpdateTimeoutSecs * time.Second,
		},
		Mongo: MongoConfig{
			Host:         defMongoHost,
			User:         defMongoUser,
			Password:     defMongoPass,
			DB:           defMongoDB,
			ArticlesColl: defMongoArticles,
			FeedsColl:    defMongoFeeds,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() =\n %+v\nwant\n %+v", got, want)
	}
}

func TestLoad_Overrides(t *testing.T) {
	setEnv(t, map[string]string{
		envMongoUser:      "alice",
		envMongoPass:      "secret",
		envMongoHost:      "mongo.example.com:27018",
		envMongoDB:        "obsdb",
		envMongoArticles:  "arts",
		envMongoFeeds:     "fds",
		envAuthToken:      "xyz-token",
		envUpdateInterval: "300",
		envUpdateTimeout:  "120",
	})

	got, err := Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	want := &Config{
		AuthToken: "xyz-token",
		FeedRuntime: FeedRuntimeConfig{
			UpdateInterval: 300 * time.Second,
			UpdateTimeout:  120 * time.Second,
		},
		Mongo: MongoConfig{
			Host:         "mongo.example.com:27018",
			User:         "alice",
			Password:     "secret",
			DB:           "obsdb",
			ArticlesColl: "arts",
			FeedsColl:    "fds",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("Load() =\n %+v\nwant\n %+v", got, want)
	}
}

func TestLoadServer_Defaults(t *testing.T) {
	// Crucially, OBSERVER_AUTH_TOKEN is left unset: the read server must load
	// without it.
	setEnv(t, map[string]string{})

	got, err := LoadServer()
	if err != nil {
		t.Fatalf("LoadServer() error = %v", err)
	}

	want := &ServerConfig{
		Addr:     defServerAddr,
		CacheTTL: defCacheTTLSecs * time.Second,
		Mongo: MongoConfig{
			Host:         defMongoHost,
			User:         defMongoUser,
			Password:     defMongoPass,
			DB:           defMongoDB,
			ArticlesColl: defMongoArticles,
			FeedsColl:    defMongoFeeds,
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadServer() =\n %+v\nwant\n %+v", got, want)
	}
}

func TestLoadServer_Overrides(t *testing.T) {
	setEnv(t, map[string]string{
		envServerAddr:    "127.0.0.1:9000",
		envCacheTTL:      "30",
		envMongoHost:     "mongo.example.com:27018",
		envMongoUser:     "alice",
		envMongoPass:     "secret",
		envMongoDB:       "obsdb",
		envMongoArticles: "arts",
		envMongoFeeds:    "fds",
	})

	got, err := LoadServer()
	if err != nil {
		t.Fatalf("LoadServer() error = %v", err)
	}

	want := &ServerConfig{
		Addr:     "127.0.0.1:9000",
		CacheTTL: 30 * time.Second,
		Mongo: MongoConfig{
			Host:         "mongo.example.com:27018",
			User:         "alice",
			Password:     "secret",
			DB:           "obsdb",
			ArticlesColl: "arts",
			FeedsColl:    "fds",
		},
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("LoadServer() =\n %+v\nwant\n %+v", got, want)
	}
}

func TestLoadServer_InvalidCacheTTL(t *testing.T) {
	setEnv(t, map[string]string{envCacheTTL: "1m"}) // not an integer number of seconds

	cfg, err := LoadServer()
	if err == nil {
		t.Fatalf("LoadServer() error = nil, want error")
	}
	if cfg != nil {
		t.Errorf("LoadServer() cfg = %+v, want nil on error", cfg)
	}
	if !strings.Contains(err.Error(), envCacheTTL) {
		t.Errorf("LoadServer() error = %q, missing substring %q", err, envCacheTTL)
	}
}

func TestLoad_Errors(t *testing.T) {
	cases := []struct {
		name string
		env  map[string]string
		want []string // substrings expected in the error
	}{
		{
			name: "missing_auth_token",
			env:  map[string]string{}, // OBSERVER_AUTH_TOKEN left empty
			want: []string{envAuthToken},
		},
		{
			name: "invalid_update_interval",
			env:  map[string]string{envAuthToken: "tok", envUpdateInterval: "soon"},
			want: []string{envUpdateInterval},
		},
		{
			name: "zero_update_interval",
			env:  map[string]string{envAuthToken: "tok", envUpdateInterval: "0"},
			want: []string{envUpdateInterval},
		},
		{
			name: "invalid_update_timeout",
			env:  map[string]string{envAuthToken: "tok", envUpdateTimeout: "sixty"},
			want: []string{envUpdateTimeout},
		},
		{
			name: "zero_update_timeout",
			env:  map[string]string{envAuthToken: "tok", envUpdateTimeout: "0"},
			want: []string{envUpdateTimeout},
		},
		{
			// Both parse as integers, so they reach validate() and must be
			// reported together with the missing token via errors.Join.
			name: "aggregates_all_validation_errors",
			env:  map[string]string{envUpdateInterval: "0", envUpdateTimeout: "0"},
			want: []string{envAuthToken, envUpdateInterval, envUpdateTimeout},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, tc.env)

			cfg, err := Load()
			if err == nil {
				t.Fatalf("Load() error = nil, want error containing %v", tc.want)
			}
			if cfg != nil {
				t.Errorf("Load() cfg = %+v, want nil on error", cfg)
			}
			for _, want := range tc.want {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("Load() error = %q, missing substring %q", err, want)
				}
			}
		})
	}
}

func TestLoad_UnsetsSecrets(t *testing.T) {
	// getEnvOnce must scrub secrets from the environment after reading them.
	setEnv(t, map[string]string{
		envAuthToken: "tok",
		envMongoUser: "u",
		envMongoPass: "pw",
	})

	if _, err := Load(); err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	for _, key := range []string{envAuthToken, envMongoUser, envMongoPass} {
		if v := os.Getenv(key); v != "" {
			t.Errorf("%s still set after Load: %q", key, v)
		}
	}
}

func TestValidate(t *testing.T) {
	t.Parallel() // pure: touches no environment

	// newValid returns a Config that passes validation; cases mutate a copy.
	newValid := func() *Config {
		return &Config{
			AuthToken:   "tok",
			FeedRuntime: FeedRuntimeConfig{UpdateInterval: time.Second, UpdateTimeout: time.Second},
		}
	}

	cases := []struct {
		name    string
		mutate  func(*Config)
		wantErr []string // substrings; nil means expect no error
	}{
		{name: "valid", mutate: func(*Config) {}, wantErr: nil},
		{name: "missing_auth", mutate: func(c *Config) { c.AuthToken = "" }, wantErr: []string{envAuthToken}},
		{name: "zero_update_interval", mutate: func(c *Config) { c.FeedRuntime.UpdateInterval = 0 }, wantErr: []string{envUpdateInterval}},
		{name: "zero_update_timeout", mutate: func(c *Config) { c.FeedRuntime.UpdateTimeout = 0 }, wantErr: []string{envUpdateTimeout}},
		{
			name: "all_invalid",
			mutate: func(c *Config) {
				c.AuthToken = ""
				c.FeedRuntime.UpdateInterval = 0
				c.FeedRuntime.UpdateTimeout = 0
			},
			wantErr: []string{envAuthToken, envUpdateInterval, envUpdateTimeout},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			c := newValid()
			tc.mutate(c)
			err := c.validate()

			if len(tc.wantErr) == 0 {
				if err != nil {
					t.Fatalf("validate() = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validate() = nil, want error")
			}
			for _, want := range tc.wantErr {
				if !strings.Contains(err.Error(), want) {
					t.Errorf("validate() error = %q, missing substring %q", err, want)
				}
			}
		})
	}
}
