package habr

import (
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want time.Time
		ok   bool
	}{
		{
			name: "RFC1123_GMT",
			in:   "Sun, 27 Nov 2011 19:21:13 GMT",
			want: time.Date(2011, time.November, 27, 19, 21, 13, 0, time.UTC),
			ok:   true,
		},
		{
			name: "RFC1123Z_Minus0700",
			in:   "Mon, 02 Jan 2006 15:04:05 -0700",
			// 15:04:05 -0700 == 22:04:05 UTC
			want: time.Date(2006, time.January, 2, 22, 4, 5, 0, time.UTC),
			ok:   true,
		},
		{
			name: "RFC1123Z_Plus0300",
			in:   "Tue, 05 Aug 2025 13:11:34 +0300",
			// 13:11:34 +0300 == 10:11:34 UTC
			want: time.Date(2025, time.August, 5, 10, 11, 34, 0, time.UTC),
			ok:   true,
		},
		{
			name: "TrimmedWhitespace",
			in:   "  \tSun, 27 Nov 2011 19:21:13 GMT\n",
			want: time.Date(2011, time.November, 27, 19, 21, 13, 0, time.UTC),
			ok:   true,
		},
		{
			name: "Invalid_MissingZone",
			in:   "Fri, 08 Aug 2025 17:17:38", // no zone
			ok:   false,
		},
		{
			name: "Invalid_ISO8601",
			in:   "2011-11-27T19:21:13Z",
			ok:   false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			got, err := parseDate(tc.in)
			if tc.ok {
				if err != nil {
					t.Fatalf("parseDate(%q) unexpected error: %v", tc.in, err)
				}
				if !got.Equal(tc.want) {
					t.Fatalf("parseDate(%q) mismatch: got %s, want %s", tc.in, got.UTC(), tc.want.UTC())
				}
			} else {
				if err == nil {
					t.Fatalf("parseDate(%q) expected error, got nil (time=%v)", tc.in, got)
				}
				if !got.IsZero() {
					t.Fatalf("parseDate(%q) on error should return zero time, got %v", tc.in, got)
				}
			}
		})
	}
}

func TestParseAuthor(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "bare_username",
			in:   "some_user",
			want: "some_user",
		},
		{
			name: "company_blog",
			in:   "some_user (SomeCompany.com)",
			want: "some_user",
		},
		{
			name: "company_name_with_spaces",
			in:   "some_user (Рога и копыта)",
			want: "some_user",
		},
		{
			name: "no_space_before_bracket",
			in:   "some_user(SomeCompany)",
			want: "some_user",
		},
		{
			name: "trimmed_whitespace",
			in:   "  \tsome_user\n",
			want: "some_user",
		},
		{
			name: "company_only",
			in:   "(SomeCompany.com)",
			want: "",
		},
		{
			name: "empty",
			in:   "",
			want: "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := parseAuthor(tc.in); got != tc.want {
				t.Fatalf("parseAuthor(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseXML_ValidRSS(t *testing.T) {
	t.Parallel()

	testData := readTestData(t, testDataValidRSSFilename)

	got, err := parseXML(testData)
	if err != nil {
		t.Fatalf("parseXML: %v", err)
	}

	if len(got) != testDataExpectedItemsNum {
		t.Fatalf("got %d articles, want %d", len(got), testDataExpectedItemsNum)
	}

	want := newExpectedThreeArticles(t)

	for i := range testDataExpectedItemsNum {
		if !reflect.DeepEqual(got[i], want[i]) {
			t.Errorf("got %+v, want %+v", got[i], want[i])
		}
	}

}

func TestParseXML_InvalidRSS(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name     string
		filename string
		err      string
	}{
		{
			name:     "bad_RSS",
			filename: testDataBadRSSFilename,
			err:      "parsing XML: XML syntax error on line 1: unexpected EOF",
		},
		{
			name:     "empty_RSS",
			filename: testDataEmptyRSSFilename,
			err:      "parsing XML: no items found",
		},
		{
			name:     "bad_date_RSS",
			filename: testDataBadDateRSSFilename,
			err:      "parsing XML: parsing date",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			testData := readTestData(t, tc.filename)
			_, err := parseXML(testData)
			if err == nil {
				t.Fatalf("want err %q, got no err", tc.err)
			}
			if !strings.Contains(err.Error(), tc.err) {
				t.Fatalf("want err to contain %q, got %q", tc.err, err.Error())
			}

		})
	}
}
