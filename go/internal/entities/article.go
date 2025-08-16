package entities

import "time"

type Article struct {
	ID      string    `json:"id" bson:"_id"`
	Title   string    `json:"title" bson:"title"`
	PubDate time.Time `json:"pub_date" bson:"pub_date"`
	Author  string    `json:"author" bson:"author"`
	Summary *Summary  `json:"summary" bson:"summary"`
}

func (a *Article) URL() string {
	return a.ID
}
