import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";
import type { Plugin } from "vite";

// The Yandex.Metrika counter is production-only configuration: its id lives
// in the deployment's .env, never in the repo. When OBSERVER_METRIKA_ID is
// set at build time the placeholder comment in index.html is replaced with
// the exact snippet the live site has always served; unset builds (dev, CI)
// carry no analytics at all.
const METRIKA_PLACEHOLDER =
  /[ \t]*<!-- yandex-metrika:[\s\S]*?removed otherwise -->\n?/;

function metrikaSnippet(id: string): string {
  return `<!-- Yandex.Metrika counter -->
<script type="text/javascript" >
(function(m,e,t,r,i,k,a){m[i]=m[i]||function(){(m[i].a=m[i].a||[]).push(arguments)};
m[i].l=1*new Date();
for (var j = 0; j < document.scripts.length; j++) {if (document.scripts[j].src === r) { return; }}
k=e.createElement(t),a=e.getElementsByTagName(t)[0],k.async=1,k.src=r,a.parentNode.insertBefore(k,a)})
(window, document, "script", "https://mc.yandex.ru/metrika/tag.js", "ym");

ym(${id}, "init", {
        clickmap:true,
        trackLinks:true,
        accurateTrackBounce:true,
        webvisor:true
});
</script>
<noscript><div><img src="https://mc.yandex.ru/watch/${id}" style="position:absolute; left:-9999px;" alt="" /></div></noscript>
<!-- /Yandex.Metrika counter -->
`;
}

function injectMetrika(): Plugin {
  const id = process.env.OBSERVER_METRIKA_ID;
  return {
    name: "inject-metrika",
    transformIndexHtml(html) {
      return html.replace(METRIKA_PLACEHOLDER, id ? metrikaSnippet(id) : "");
    },
  };
}

export default defineConfig({
  plugins: [react(), injectMetrika()],
  server: {
    // The Go server sends no CORS headers, so /feeds must stay same-origin;
    // nginx does this in production, this proxy does it in dev.
    proxy: {
      "/feeds": "http://localhost:8080",
    },
  },
});
