import { useCallback, useState } from "react";

// The pinned theme choice; absence means "follow the OS theme".
const STORAGE_KEY = "habr-observer:theme";

export type ThemePreference = "light" | "dark";

// Strict read: only the two exact stored values pin the theme; anything else —
// no value, garbage, or storage being unavailable — keeps following the system.
function readStored(): ThemePreference | null {
  try {
    const raw = window.localStorage.getItem(STORAGE_KEY);
    return raw === "light" || raw === "dark" ? raw : null;
  } catch {
    return null;
  }
}

export function useThemePreference(): [
  ThemePreference | null,
  (value: ThemePreference) => void,
] {
  // Lazy initializer: a synchronous, pure read — no flash of the wrong theme
  // on load, and safe under StrictMode's double invocation.
  const [preference, setPreference] = useState(readStored);

  const update = useCallback((value: ThemePreference) => {
    setPreference(value);
    try {
      window.localStorage.setItem(STORAGE_KEY, value);
    } catch {
      // Private mode or quota: the choice lives for the session only.
    }
  }, []);

  return [preference, update];
}
