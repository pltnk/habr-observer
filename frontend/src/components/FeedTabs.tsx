import { useState } from "react";
import { Tab, TabList, TabPanel, TabProvider } from "@gravity-ui/uikit";

import type { Feed } from "../types";
import { ArticleEntry } from "./ArticleEntry";

interface FeedTabsProps {
  feeds: Feed[];
  collapseSummaries: boolean;
}

export function FeedTabs({ feeds, collapseSummaries }: FeedTabsProps) {
  // Tab state is deliberately not in the URL — the original kept it purely
  // client-side and a reload reset it.
  const [activeFeedId, setActiveFeedId] = useState(() => feeds[0]?.id ?? "");

  // Every panel is rendered and stays mounted (TabPanel hides inactive ones),
  // so per-article disclosure state survives tab switches — as the original's
  // pre-rendered panels did.
  return (
    <TabProvider value={activeFeedId} onUpdate={setActiveFeedId}>
      <div className="tab-bar">
        <TabList size="xl" contentOverflow="scroll">
          {feeds.map((feed) => (
            <Tab key={feed.id} value={feed.id}>
              {feed.name}
            </Tab>
          ))}
        </TabList>
      </div>
      {feeds.map((feed) => (
        <TabPanel key={feed.id} value={feed.id}>
          {(feed.articles ?? []).map((article) => (
            <ArticleEntry
              key={article.id}
              article={article}
              collapsed={collapseSummaries}
            />
          ))}
        </TabPanel>
      ))}
    </TabProvider>
  );
}
