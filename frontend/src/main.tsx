// Inter is self-hosted (cyrillic subsets included) and must load before
// uikit's styles; uikit's own fonts.css is a render-blocking Google Fonts
// @import and is deliberately not used.
import "@fontsource/inter/400.css";
import "@fontsource/inter/600.css";
import "@gravity-ui/uikit/styles/styles.css";
import "./styles/global.css";

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ThemeProvider, configure } from "@gravity-ui/uikit";

import App from "./App";

configure({ lang: "ru" });

const root = document.getElementById("root");
if (root === null) {
  throw new Error("missing #root element");
}

// ThemeProvider's default theme is "system": it live-tracks the OS
// light/dark preference, matching the original's automatic theming.
createRoot(root).render(
  <StrictMode>
    <ThemeProvider>
      <App />
    </ThemeProvider>
  </StrictMode>,
);
