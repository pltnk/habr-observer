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
        <span>{pubDateFormat.format(new Date(article.pub_date))} (UTC)</span>
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
      {/* flex-start + the CSS offset on .article-info anchor the button to
          the title's FIRST line; centering against the whole block leaves it
          floating between the lines of wrapped titles on phones. */}
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
        {/* HelpMark's icon is hardcoded, so this is its Popover-plus-button
            shape with a CircleInfo glyph; safePolygon keeps the popover open
            while the pointer travels to the links inside. */}
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
        <SummaryTheses
          content={article.summary.content}
          collapsed={collapsed}
        />
      )}
      <Divider className="article-divider" />
    </article>
  );
}
