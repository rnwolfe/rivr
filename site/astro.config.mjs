// @ts-check
import { defineConfig } from "astro/config";
import starlight from "@astrojs/starlight";
import starlightLlmsTxt from "starlight-llms-txt";

const SITE = "https://rivr.sh";

export default defineConfig({
  site: SITE,
  integrations: [
    starlight({
      title: "rivr",
      description:
        "A read-only Amazon Shopping CLI built for AI agents — one normalized schema over SerpApi, Rainforest, the official Creators API, or keyless scraping.",
      logo: { src: "./src/assets/mark.svg", alt: "rivr" },
      customCss: ["./src/styles/tokens.css", "./src/styles/docs.css"],
      social: [
        { icon: "github", label: "GitHub", href: "https://github.com/rnwolfe/rivr" },
      ],
      plugins: [starlightLlmsTxt()],
      // Head override sets per-page og:image/twitter:image (see src/components/Head.astro).
      components: { Head: "./src/components/Head.astro" },
      head: [
        { tag: "meta", attrs: { property: "og:type", content: "website" } },
        { tag: "meta", attrs: { name: "twitter:card", content: "summary_large_image" } },
        {
          tag: "link",
          attrs: {
            rel: "stylesheet",
            href: "https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,600;12..96,800&family=Inter+Tight:wght@400;500;600&family=JetBrains+Mono:wght@400;600&display=swap",
          },
        },
      ],
      sidebar: [
        { label: "Start", items: [{ slug: "getting-started" }] },
        { label: "Guides", items: [{ slug: "backends" }, { slug: "auth" }] },
        { label: "Reference", items: [{ slug: "commands" }, { slug: "agents" }] },
      ],
      editLink: { baseUrl: "https://github.com/rnwolfe/rivr/edit/main/site/" },
    }),
  ],
});
