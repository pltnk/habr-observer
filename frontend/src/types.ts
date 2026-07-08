// Mirrors the canonical /feeds response — backend/internal/domain/entities.go.

/** An article's AI-generated summary. */
export interface Summary {
  /** Link to the summary on 300.ya.ru. */
  url: string;
  /** Thesis bullet points. */
  content: string[];
}

/** A habr.com article and its optional summary. */
export interface Article {
  /** The article's habr.com URL (the RSS guid); also the link target. */
  id: string;
  title: string;
  /** RFC 3339 UTC timestamp, parsed at render time. */
  pub_date: string;
  /** Habr username; linked in the info popover when non-empty. */
  author: string;
  /** `null` until the article has been summarized. */
  summary: Summary | null;
}

/** A habr "top articles" feed with its articles. */
export interface Feed {
  /** The feed's RSS URL — unique, never displayed. */
  id: string;
  name: string;
  /** `null` when the feed has no stored articles yet. */
  articles: Article[] | null;
}
