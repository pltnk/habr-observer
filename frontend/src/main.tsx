// Self-hosted Source Sans Pro (weights 400/600/700), imported before uikit's
// styles. uikit's bundled fonts.css is a render-blocking Google Fonts import
// and is intentionally not used.
import "@fontsource/source-sans-pro/400.css";
import "@fontsource/source-sans-pro/600.css";
import "@fontsource/source-sans-pro/700.css";
import "@gravity-ui/uikit/styles/styles.css";
import "./styles/global.css";

import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { configure } from "@gravity-ui/uikit";

import App from "./App";

configure({ lang: "ru" });

const root = document.getElementById("root");
if (root === null) {
  throw new Error("missing #root element");
}

createRoot(root).render(
  <StrictMode>
    <App />
  </StrictMode>,
);
