import { useEffect, useState } from "react";
import { Alert, Flex, Spin, Text } from "@gravity-ui/uikit";

import { BackToTop } from "./components/BackToTop";
import { FeedTabs } from "./components/FeedTabs";
import { Footer } from "./components/Footer";
import { Header } from "./components/Header";
import { SummaryViewSelector } from "./components/SummaryViewSelector";
import { useCollapsePreference } from "./hooks/useCollapsePreference";
import { useFeeds } from "./hooks/useFeeds";

// Streamlit showed its spinner only after 0.1 s in flight; the same
// anti-flicker delay keeps a fast response from flashing a spinner.
const SPINNER_DELAY_MS = 100;

function LoadingIndicator() {
  const [visible, setVisible] = useState(false);

  useEffect(() => {
    const timer = setTimeout(() => setVisible(true), SPINNER_DELAY_MS);
    return () => clearTimeout(timer);
  }, []);

  if (!visible) {
    return null;
  }
  return (
    <Flex
      justifyContent="center"
      alignItems="center"
      gap={3}
      className="loading-row"
    >
      <Spin size="l" />
      <Text variant="body-2">Читаю статьи...</Text>
    </Flex>
  );
}

export default function App() {
  const feedsState = useFeeds();
  const [collapseSummaries, setCollapseSummaries] = useCollapsePreference();

  return (
    <div className="page">
      <Header />
      <main>
        {feedsState.status === "loading" && <LoadingIndicator />}
        {feedsState.status === "empty" && (
          <Alert
            theme="info"
            message="Лента пересобирается, загляните позже 😉"
            className="empty-banner"
          />
        )}
        {feedsState.status === "ready" && (
          <>
            <SummaryViewSelector
              collapsed={collapseSummaries}
              onUpdate={setCollapseSummaries}
            />
            <FeedTabs
              feeds={feedsState.feeds}
              collapseSummaries={collapseSummaries}
            />
          </>
        )}
      </main>
      {feedsState.status !== "loading" && (
        <>
          <Footer />
          <BackToTop />
        </>
      )}
    </div>
  );
}
