import { useEffect, useState } from "react";
import { Alert, Flex, Spin, Text, ThemeProvider } from "@gravity-ui/uikit";

import { BackToTop } from "./components/BackToTop";
import { FeedTabs } from "./components/FeedTabs";
import { Footer } from "./components/Footer";
import { Header } from "./components/Header";
import { SummaryViewSelector } from "./components/SummaryViewSelector";
import { ThemeSelector } from "./components/ThemeSelector";
import { useCollapsePreference } from "./hooks/useCollapsePreference";
import { useFeeds } from "./hooks/useFeeds";
import { useThemePreference } from "./hooks/useThemePreference";

// Delay before showing the spinner, so a fast response never flashes one.
const SPINNER_DELAY_MS = 100;

/** Spinner shown while feeds load, deferred by {@link SPINNER_DELAY_MS}. */
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

/**
 * Root component: fetches feeds and renders the header, controls, feed tabs,
 * footer, and back-to-top button, switching between the loading, empty, and
 * ready states.
 */
export default function App() {
  const feedsState = useFeeds();
  const [collapseSummaries, setCollapseSummaries] = useCollapsePreference();
  const [themePreference, setThemePreference] = useThemePreference();

  return (
    // ThemeProvider lives here, not in main.tsx, because the pinned theme is
    // app state; "system" follows the OS until the user pins a choice.
    <ThemeProvider theme={themePreference ?? "system"}>
      <div className="page">
        <Header />
        <main>
          {/* The preference controls need no feed data, so they render right
              away instead of waiting for the fetch — only the tabs and
              articles below them depend on it. */}
          <Flex
            justifyContent="center"
            alignItems="center"
            gap={3}
            className="controls-row"
          >
            <SummaryViewSelector
              collapsed={collapseSummaries}
              onUpdate={setCollapseSummaries}
            />
            <ThemeSelector
              preference={themePreference}
              onUpdate={setThemePreference}
            />
          </Flex>
          {feedsState.status === "loading" && <LoadingIndicator />}
          {feedsState.status === "empty" && (
            <Alert
              theme="info"
              message="Лента пересобирается, загляните позже 😉"
              className="empty-banner"
            />
          )}
          {feedsState.status === "ready" && (
            <FeedTabs
              feeds={feedsState.feeds}
              collapseSummaries={collapseSummaries}
            />
          )}
        </main>
        {feedsState.status !== "loading" && (
          <>
            <Footer />
            <BackToTop />
          </>
        )}
      </div>
    </ThemeProvider>
  );
}
