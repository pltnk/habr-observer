// Package domain defines the core data structures shared across the
// application — the Habr articles, their AI-generated summaries, and the feeds
// that group them. These are plain data types (DTOs) with no behavior; they
// are serialized to JSON for the API and to BSON for MongoDB.
package domain

import "time"

// Summary is the AI-generated, thesis-style summary of an article. URL points
// to the summary on the 300.ya.ru service; Content holds its bullet points.
type Summary struct {
	URL     string   `json:"url" bson:"url"`
	Content []string `json:"content" bson:"content"`
}

// Article is a single Habr article together with its summary. ID is the
// article's URL (the RSS <guid>) and doubles as its MongoDB _id. Summary is
// nil until the article has been summarized.
type Article struct {
	ID      string    `json:"id" bson:"_id"`
	Title   string    `json:"title" bson:"title"`
	PubDate time.Time `json:"pub_date" bson:"pub_date"`
	Author  string    `json:"author" bson:"author"`
	Summary *Summary  `json:"summary" bson:"summary"`
}

// Feed is a snapshot of a Habr "top articles" RSS feed. ID is the feed's URL
// and doubles as its MongoDB _id; Articles holds the denormalized articles in
// feed order.
type Feed struct {
	ID       string     `json:"id" bson:"_id"`
	Name     string     `json:"name" bson:"name"`
	Articles []*Article `json:"articles" bson:"articles"`
}
