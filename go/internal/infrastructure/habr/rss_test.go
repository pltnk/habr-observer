package habr

import (
	"reflect"
	"testing"
)

func assertEqual(t *testing.T, got, want any, msg string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("%s: got %v, want %v", msg, got, want)
	}
}

var canonicalOrder = [...]RSSFeed{
	FeedDaily,
	FeedWeekly,
	FeedMonthly,
	FeedYearly,
	FeedAllTime,
}

var canonicalName = map[RSSFeed]string{
	FeedDaily:   "Сутки",
	FeedWeekly:  "Неделя",
	FeedMonthly: "Месяц",
	FeedYearly:  "Год",
	FeedAllTime: "Всё время",
}

var canonicalURL = map[RSSFeed]string{
	FeedDaily:   "https://habr.com/ru/rss/articles/top/daily/?fl=ru",
	FeedWeekly:  "https://habr.com/ru/rss/articles/top/weekly/?fl=ru",
	FeedMonthly: "https://habr.com/ru/rss/articles/top/monthly/?fl=ru",
	FeedYearly:  "https://habr.com/ru/rss/articles/top/yearly/?fl=ru",
	FeedAllTime: "https://habr.com/ru/rss/articles/top/alltime/?fl=ru",
}

func TestAllFeeds(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func(*testing.T)
	}{
		{
			name: "order",
			fn: func(t *testing.T) {
				got := AllFeeds()
				if !reflect.DeepEqual(got, canonicalOrder[:]) {
					t.Fatalf("order mismatch: got %v, want %v", got, canonicalOrder)
				}
			},
		},
		{
			name: "immutability",
			fn: func(t *testing.T) {
				s := AllFeeds()
				s[0] = 999
				if AllFeeds()[0] != canonicalOrder[0] {
					t.Fatal("slice is not a copy (caller mutates package state)")
				}
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			tc.fn(t)
		})
	}
}

func TestFeedMetaLookup(t *testing.T) {
	t.Parallel()

	for _, f := range canonicalOrder {
		t.Run(f.String(), func(t *testing.T) {
			t.Parallel()
			assertEqual(t, f.Name(), canonicalName[f], "Name()")
			assertEqual(t, f.URL(), canonicalURL[f], "URL()")
		})
	}
}

func TestPanicOnInvalidValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		fn   func()
	}{
		{
			name: "Name panic",
			fn:   func() { _ = RSSFeed(-1).Name() },
		},
		{
			name: "URL panic",
			fn:   func() { _ = RSSFeed(_feedCount).URL() },
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			defer func() {
				if r := recover(); r == nil {
					t.Fatalf("%s: expected panic, got none", tc.name)
				}
			}()
			tc.fn()
		})
	}
}
