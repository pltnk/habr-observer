import { useCallback, useState } from "react";

/** localStorage key for the pinned theme; absence means "follow the OS". */
const STORAGE_KEY = "habr-observer:theme";

/** A pinned theme choice. */
export type ThemePreference = "light" | "dark";

/**
 * Strict read: only "light" or "dark" pin the theme; any other value, a missing
 * key, or unavailable storage keeps following the system theme.
 */
function readStored(): ThemePreference | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    return raw === "light" || raw === "dark" ? raw : null;
  } catch {
    return null;
  }
}

/**
 * The pinned theme, persisted to localStorage, or `null` to follow the OS.
 * Returns the current value and a setter.
 */
export function useThemePreference(): [
  ThemePreference | null,
  (value: ThemePreference) => void,
] {
  // Lazy, synchronous read: no flash of the wrong theme, safe under StrictMode.
  const [preference, setPreference] = useState(readStored);

  const update = useCallback((value: ThemePreference) => {
    setPreference(value);
    try {
      window.localStorage.setItem(STORAGE_KEY, value);
    } catch {
      // Private mode or quota: the choice lasts for the session only.
    }
  }, []);

  return [preference, update];
}
