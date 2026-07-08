import { useEffect, useState } from "react";
import { Tab, TabList, TabPanel, TabProvider } from "@gravity-ui/uikit";

import type { Feed } from "../types";
import { ArticleEntry } from "./ArticleEntry";

interface FeedTabsProps {
  feeds: Feed[];
  collapseSummaries: boolean;
}

/**
 * Feed tab bar with one panel per feed. All panels stay mounted (TabPanel
 * hides the inactive ones), so each article's expansion state survives tab
 * switches.
 */
export function FeedTabs({ feeds, collapseSummaries }: FeedTabsProps) {
  // Active tab is client-only state, not in the URL; a reload resets it.
  const [activeFeedId, setActiveFeedId] = useState(() => feeds[0]?.id ?? "");

  // Play the article fade-in only on the initial feed load, then drop the class
  // so switching tabs (which unhides a display:none panel) doesn't replay it.
  // The delay outlasts the longest article stagger (0.33s) plus its 0.6s fade.
  const [entering, setEntering] = useState(true);
  useEffect(() => {
    const timer = setTimeout(() => setEntering(false), 1000);
    return () => clearTimeout(timer);
  }, []);

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
              entering={entering}
            />
          ))}
        </TabPanel>
      ))}
    </TabProvider>
  );
}
