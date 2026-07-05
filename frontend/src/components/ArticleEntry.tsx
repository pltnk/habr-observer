import { FaceRobotSmile, SquareArticle } from "@gravity-ui/icons";
import { Divider, Flex, HelpMark, Icon, Link, Text } from "@gravity-ui/uikit";

import type { Article } from "../types";
import { SummaryTheses } from "./SummaryTheses";

// UTC on purpose: the original displayed the raw UTC datetime; viewer-local
// time would silently show different values to different users.
const pubDateFormat = new Intl.DateTimeFormat("ru-RU", {
  timeZone: "UTC",
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

export function ArticleEntry({ article, collapsed }: ArticleEntryProps) {
  return (
    <article className="article">
      <Flex alignItems="center" gap={2} className="article-title">
        <Text as="h3" variant="subheader-3">
          {article.title}
        </Text>
        {/* safePolygon keeps the popover open while the pointer travels to
            the author link inside it. */}
        <HelpMark popoverProps={{ enableSafePolygon: true }}>
          {article.author !== "" && (
            <div>
              Автор:{" "}
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
            Дата публикации: {pubDateFormat.format(new Date(article.pub_date))}{" "}
            (UTC)
          </div>
        </HelpMark>
      </Flex>
      {article.summary !== null && (
        <SummaryTheses
          content={article.summary.content}
          collapsed={collapsed}
        />
      )}
      <Text
        as="div"
        variant="caption-2"
        color="secondary"
        className="links-row"
      >
        {article.summary !== null && (
          <Link
            view="secondary"
            href={article.summary.url}
            target="_blank"
            rel="noopener noreferrer"
          >
            <Icon data={FaceRobotSmile} size={16} />
            Ссылка на пересказ
          </Link>
        )}
        <Link
          view="secondary"
          href={article.id}
          target="_blank"
          rel="noopener noreferrer"
        >
          <Icon data={SquareArticle} size={16} />
          Открыть оригинал
        </Link>
      </Text>
      <Divider className="article-divider" />
    </article>
  );
}
