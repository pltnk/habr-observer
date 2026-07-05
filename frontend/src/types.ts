// Mirrors the canonical /feeds schema — backend/internal/domain/entities.go.

export interface Summary {
  url: string;
  content: string[];
}

export interface Article {
  // The article's habr.com URL (the RSS guid) — doubles as the link target.
  id: string;
  title: string;
  // RFC 3339 UTC timestamp; parsed only at render time.
  pub_date: string;
  // Habr username; linked in the title help popover when non-empty.
  author: string;
  summary: Summary | null;
}

export interface Feed {
  // The feed's RSS URL — unique, never displayed.
  id: string;
  name: string;
  articles: Article[] | null;
}
