package habr

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"habr-observer/internal/entities"
	"strings"
	"time"
)

type rss struct {
	Channel rssChannel `xml:"channel"`
}

type rssChannel struct {
	Items []*item `xml:"item"`
}

type item struct {
	GUID    string `xml:"guid"`
	Title   string `xml:"title"`
	PubDate string `xml:"pubDate"`
	Creator string `xml:"http://purl.org/dc/elements/1.1/ creator"`
}

func parseDate(date string) (time.Time, error) {
	date = strings.TrimSpace(date)

	t, err := time.Parse(time.RFC1123, date)
	if err != nil {
		t, err = time.Parse(time.RFC1123Z, date)
	}
	if err != nil {
		return time.Time{}, fmt.Errorf("parsing date: %w", err)
	}

	return t, nil
}

func parseXML(data []byte) ([]*entities.Article, error) {
	data = bytes.TrimSpace(data)

	var parsed rss
	err := xml.Unmarshal(data, &parsed)
	if err != nil {
		return nil, fmt.Errorf("parsing XML: %w", err)
	}

	nItems := len(parsed.Channel.Items)
	if nItems == 0 {
		return nil, errors.New("parsing XML: no items found")
	}

	articles := make([]*entities.Article, 0, nItems)
	for _, it := range parsed.Channel.Items {
		t, err := parseDate(it.PubDate)
		if err != nil {
			return nil, fmt.Errorf("parsing XML: %w", err)
		}

		articles = append(articles, &entities.Article{
			ID:      strings.TrimSpace(it.GUID),
			Title:   strings.TrimSpace(it.Title),
			PubDate: t,
			Author:  strings.TrimSpace(it.Creator),
		})
	}

	return articles, nil
}
