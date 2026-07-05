import { useCallback, useState } from "react";

// The one persisted user preference: collapse summaries to the first three
// theses («Кратко») or show them in full («Целиком»).
const STORAGE_KEY = "habr-observer:collapse-theses";

// Strict read: only an exact stored "false" turns collapsing off; anything
// else — no value, garbage, or storage being unavailable — falls back to the
// default (collapsed, matching the original app's on-by-default switch).
function readStored(): boolean {
  try {
    return window.localStorage.getItem(STORAGE_KEY) !== "false";
  } catch {
    return true;
  }
}

export function useCollapsePreference(): [boolean, (value: boolean) => void] {
  // Lazy initializer: a synchronous, pure read — no flash of the wrong view
  // on load, and safe under StrictMode's double invocation.
  const [collapsed, setCollapsed] = useState(readStored);

  const update = useCallback((value: boolean) => {
    setCollapsed(value);
    try {
      window.localStorage.setItem(STORAGE_KEY, String(value));
    } catch {
      // Private mode or quota: the preference lives for the session only.
    }
  }, []);

  return [collapsed, update];
}
