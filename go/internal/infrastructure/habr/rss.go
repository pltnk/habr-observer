// Package habr provides strongly typed references to Habr “top articles” RSS feeds.
package habr

//go:generate go run golang.org/x/tools/cmd/stringer -type=RSSFeed

type RSSFeed int

const (
	FeedDaily RSSFeed = iota
	FeedWeekly
	FeedMonthly
	FeedYearly
	FeedAllTime
	_feedCount // always keep this last
)

type feedMeta struct {
	name string
	url  string
}

var feeds = [_feedCount]feedMeta{
	FeedDaily:   {"Сутки", "https://habr.com/ru/rss/articles/top/daily/?fl=ru"},
	FeedWeekly:  {"Неделя", "https://habr.com/ru/rss/articles/top/weekly/?fl=ru"},
	FeedMonthly: {"Месяц", "https://habr.com/ru/rss/articles/top/monthly/?fl=ru"},
	FeedYearly:  {"Год", "https://habr.com/ru/rss/articles/top/yearly/?fl=ru"},
	FeedAllTime: {"Всё время", "https://habr.com/ru/rss/articles/top/alltime/?fl=ru"},
}

// Name returns the display label. Panics if f is invalid.
func (f RSSFeed) Name() string {
	return feeds[f].name
}

// URL returns the RSS URL. Panics if f is invalid.
func (f RSSFeed) URL() string {
	return feeds[f].url
}

// allFeeds is the canonical iteration order.
var allFeeds = [_feedCount]RSSFeed{
	FeedDaily,
	FeedWeekly,
	FeedMonthly,
	FeedYearly,
	FeedAllTime,
}

// AllFeeds returns a copy of the canonical ordered list of feeds for safe iteration.
func AllFeeds() []RSSFeed {
	cp := make([]RSSFeed, len(allFeeds))
	copy(cp, allFeeds[:])
	return cp
}
