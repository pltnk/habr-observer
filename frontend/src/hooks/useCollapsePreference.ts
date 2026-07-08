import { useCallback, useState } from "react";

/** localStorage key for the summary-collapse preference. */
const STORAGE_KEY = "habr-observer:collapse-theses";

/**
 * Strict read: only an exact "false" turns collapsing off; any other value, a
 * missing key, or unavailable storage falls back to the default (collapsed).
 */
function readStored(): boolean {
  try {
    return window.localStorage.getItem(STORAGE_KEY) !== "false";
  } catch {
    return true;
  }
}

/**
 * Whether article summaries collapse to the first three theses, persisted to
 * localStorage. Returns the current value and a setter; defaults to collapsed.
 */
export function useCollapsePreference(): [boolean, (value: boolean) => void] {
  // Lazy, synchronous read: no flash of the wrong view, safe under StrictMode.
  const [collapsed, setCollapsed] = useState(readStored);

  const update = useCallback((value: boolean) => {
    setCollapsed(value);
    try {
      window.localStorage.setItem(STORAGE_KEY, String(value));
    } catch {
      // Private mode or quota: the preference lasts for the session only.
    }
  }, []);

  return [collapsed, update];
}
