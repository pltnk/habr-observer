import {
  Calendar,
  CircleInfo,
  FaceRobotSmile,
  Person,
} from "@gravity-ui/icons";
import {
  Button,
  Divider,
  Flex,
  Icon,
  Link,
  Popover,
  Text,
} from "@gravity-ui/uikit";

import type { Article } from "../types";
import { SummaryTheses } from "./SummaryTheses";

// Rendered in the viewer's timezone to match how habr shows publication times.
const pubDateFormat = new Intl.DateTimeFormat("ru-RU", {
  day: "2-digit",
  month: "2-digit",
  year: "numeric",
  hour: "2-digit",
  minute: "2-digit",
});

interface ArticleEntryProps {
  article: Article;
  collapsed: boolean;
}

/**
 * One article card: the title linking to the original habr article, an info
 * popover (author, date, and summary link), the thesis summary, and a trailing
 * divider.
 */
export function ArticleEntry({ article, collapsed }: ArticleEntryProps) {
  const info = (
    <div className="article-info-popover">
      {article.author !== "" && (
        <div>
          <Icon data={Person} size={16} />
          <Link
            href={`https://habr.com/ru/users/${encodeURIComponent(article.author)}`}
            target="_blank"
            rel="noopener noreferrer"
          >
            {article.author}
          </Link>
        </div>
      )}
      <div>
        <Icon data={Calendar} size={16} />
        <span>{pubDateFormat.format(new Date(article.pub_date))}</span>
      </div>
      {article.summary !== null && (
        <div>
          <Icon data={FaceRobotSmile} size={16} />
          <Link
            href={article.summary.url}
            target="_blank"
            rel="noopener noreferrer"
          >
            Ссылка на пересказ
          </Link>
        </div>
      )}
    </div>
  );

  return (
    <article className="article">
      {/* flex-start plus the CSS offset on .article-info pin the button to the
          card's top-right corner, level with the title's first line whatever
          way the title wraps. */}
      <Flex alignItems="flex-start" gap={2} className="article-title">
        <Text as="h3" variant="subheader-3">
          <Link
            view="primary"
            href={article.id}
            target="_blank"
            rel="noopener noreferrer"
          >
            {article.title}
          </Link>
        </Text>
        {/* Popover + icon button (HelpMark can't change its glyph); safePolygon
            keeps it open while the pointer travels to the links inside. */}
        <Popover content={info} hasArrow enableSafePolygon>
          <Button
            view="flat-secondary"
            size="s"
            pin="circle-circle"
            aria-label="Сведения о статье"
            className="article-info"
          >
            <Icon data={CircleInfo} size={16} />
          </Button>
        </Popover>
      </Flex>
      {article.summary !== null && (
        /* Keyed by view mode so switching «Кратко»/«Целиком» remounts and
           resets the per-article expansion. */
        <SummaryTheses
          key={collapsed ? "collapsed" : "flat"}
          content={article.summary.content}
          collapsed={collapsed}
        />
      )}
      <Divider className="article-divider" />
    </article>
  );
}
