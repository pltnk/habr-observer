package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestRoutes verifies the read API's routing: GET /feeds reaches the handler,
// other methods on that path get 405, and unknown paths get 404. The feeds
// handler is stubbed, so this needs no MongoDB.
func TestRoutes(t *testing.T) {
	t.Parallel()

	stub := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	srv := httptest.NewServer(routes(stub))
	t.Cleanup(srv.Close)

	cases := []struct {
		name   string
		method string
		path   string
		want   int
	}{
		{"get_feeds_ok", http.MethodGet, "/feeds", http.StatusOK},
		{"post_feeds_method_not_allowed", http.MethodPost, "/feeds", http.StatusMethodNotAllowed},
		{"unknown_path_not_found", http.MethodGet, "/nope", http.StatusNotFound},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			req, err := http.NewRequest(tc.method, srv.URL+tc.path, nil)
			if err != nil {
				t.Fatalf("NewRequest: %v", err)
			}
			resp, err := srv.Client().Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != tc.want {
				t.Fatalf("%s %s = %d, want %d", tc.method, tc.path, resp.StatusCode, tc.want)
			}
		})
	}
}
