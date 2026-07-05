import { useEffect, useState } from "react";

import { fetchFeeds } from "../api/feeds";
import type { Feed } from "../types";

export type FeedsState =
  | { status: "loading" }
  | { status: "ready"; feeds: Feed[] }
  | { status: "empty" };

// Fetches /feeds once on mount — no polling, no retries: the data changes
// every ~10 minutes server-side. A network error, a non-2xx status, or a
// malformed body all fold into "empty", the same state as an empty feed list.
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
