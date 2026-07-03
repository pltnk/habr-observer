package yagpt

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
)

func TestNewSummaryURL_Success(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name  string
		raw   string
		token string
	}{
		{
			name:  "simple",
			raw:   "https://300.ya.ru/BoudKEx0",
			token: "BoudKEx0",
		},
		{
			name:  "trailing_slash",
			raw:   "https://300.ya.ru/Tv0lFS6a/",
			token: "Tv0lFS6a",
		},
		{
			name:  "nested_path",
			raw:   "https://300.ya.ru/foo/bar/X1pBnQ2h",
			token: "X1pBnQ2h",
		},
		{
			name:  "query_and_fragment",
			raw:   "https://300.ya.ru/PY8NyDXD?x=1#frag",
			token: "PY8NyDXD",
		},
		{
			name:  "mixed_case_host_and_port",
			raw:   "HTTPS://300.YA.ru:443/ZdADIfJG",
			token: "ZdADIfJG",
		},
		{
			name:  "extra_whitespace",
			raw:   "  \n\t https://300.ya.ru/5SuCnKaU \t ",
			token: "5SuCnKaU",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			su, err := NewSummaryURL(tc.raw)
			if err != nil {
				t.Fatalf("NewSummaryURL(%q): unexpected error: %v", tc.raw, err)
			}

			wantURL, err := url.Parse(strings.TrimSpace(tc.raw))
			if err != nil {
				t.Fatalf("url.Parse(%q) failed in test: %v", tc.raw, err)
			}

			if got := su.URL(); !reflect.DeepEqual(got, *wantURL) {
				t.Fatalf("URL() mismatch:\n got:  %#v\n want: %#v", got, *wantURL)
			}

			if got, want := su.String(), wantURL.String(); got != want {
				t.Fatalf("String(): got %q, want %q", got, want)
			}

			if got := su.Token(); got != tc.token {
				t.Fatalf("Token(): got %q, want %q", got, tc.token)
			}

		})
	}
}

func TestNewSummaryURL_Errors(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		raw  string
		err  string
	}{
		{
			name: "malformed_url",
			raw:  "https://300.ya.ru/%zz",
			err:  "invalid URL escape",
		},
		{
			name: "wrong_scheme",
			raw:  "http://300.ya.ru/BoudKEx0",
			err:  "want Scheme=\"https\"",
		},
		{
			name: "wrong_host",
			raw:  "https://example.com/foo",
			err:  "want Hostname=\"300.ya.ru\"",
		},
		{
			name: "no_path",
			raw:  "https://300.ya.ru",
			err:  "no token",
		},
		{
			name: "root_path",
			raw:  "https://300.ya.ru/",
			err:  "no token",
		},
		{
			name: "dot_path",
			raw:  "https://300.ya.ru/.",
			err:  "no token",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			su, err := NewSummaryURL(tc.raw)
			if err == nil {
				t.Fatalf("NewSummaryURL(%q): expected error, got nil (su=%+v)", tc.raw, su)
			}

			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("error = %q, want substring %q", err, tc.err)
			}
		})
	}
}

func TestSummaryURL_URL_ReturnsCopy(t *testing.T) {
	t.Parallel()

	raw := "https://300.ya.ru/PY8NyDXD?x=1#frag"
	su, err := NewSummaryURL(raw)
	if err != nil {
		t.Fatalf("NewSummaryURL(%q): %v", raw, err)
	}

	origURL := su.URL()
	origToken := su.Token()
	origStr := su.String()

	u := su.URL()
	u.Scheme = "http"
	u.Host = "evil.example"
	u.Path = "/MUTATED"
	u.RawPath = "/MUTATED"
	u.RawQuery = "hacked=1"
	u.ForceQuery = true
	u.Fragment = "owned"

	if reflect.DeepEqual(u, su.URL()) {
		t.Fatalf("mutated copy equals internal URL: URL() likely returned an alias, not a copy")
	}

	if got := su.URL(); !reflect.DeepEqual(got, origURL) {
		t.Fatalf("internal URL mutated via returned copy: got %#v, want %#v", got, origURL)
	}

	if got := su.Token(); got != origToken {
		t.Fatalf("Token changed unexpectedly: got %q, want %q", got, origToken)
	}

	if got := su.String(); got != origStr {
		t.Fatalf("String() changed unexpectedly: got %q, want %q", got, origStr)
	}
}
