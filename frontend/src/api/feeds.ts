import type { Feed } from "../types";

// The app's only backend call. /feeds is fetched same-origin — the Go server
// sends no CORS headers, so nginx (or Vite's dev proxy) must front it.
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
