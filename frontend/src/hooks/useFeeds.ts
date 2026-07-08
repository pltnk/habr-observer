import { useEffect, useState } from "react";

import { fetchFeeds } from "../api/feeds";
import type { Feed } from "../types";

/** Loading, ready with feeds, or empty (no data or a fetch failure). */
export type FeedsState =
  | { status: "loading" }
  | { status: "ready"; feeds: Feed[] }
  | { status: "empty" };

/**
 * Fetches `/feeds` once on mount. No polling or retries — the data changes only
 * every ~10 minutes server-side. Any failure (network, non-2xx, malformed body)
 * folds into "empty", the same state as an empty feed list.
 */
export function useFeeds(): FeedsState {
  const [state, setState] = useState<FeedsState>({ status: "loading" });

  useEffect(() => {
    const controller = new AbortController();
    fetchFeeds(controller.signal)
      .then((feeds) => {
        setState(
          feeds.length > 0 ? { status: "ready", feeds } : { status: "empty" },
        );
      })
      .catch((error: unknown) => {
        // StrictMode runs the effect twice; the aborted duplicate is not a failure.
        if (controller.signal.aborted) return;
        console.error(error);
        setState({ status: "empty" });
      });
    return () => controller.abort();
  }, []);

  return state;
}
