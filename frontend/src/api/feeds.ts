import type { Feed } from "../types";

/**
 * Fetches the feed list from the same-origin `/feeds` endpoint, aborting on
 * `signal`. Throws on a non-2xx status or a non-array body.
 *
 * The Go backend sends no CORS headers, so a reverse proxy (nginx in
 * production, Vite's dev proxy locally) must front `/feeds` same-origin.
 */
export async function fetchFeeds(signal: AbortSignal): Promise<Feed[]> {
  const response = await fetch("/feeds", { signal });
  if (!response.ok) {
    throw new Error(`GET /feeds responded with ${response.status}`);
  }
  const data: unknown = await response.json();
  if (!Array.isArray(data)) {
    throw new Error("GET /feeds returned a non-array body");
  }
  return data as Feed[];
}
