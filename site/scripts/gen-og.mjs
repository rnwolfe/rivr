// Generate 1200×630 OG/social cards in the rivr brand style, one per page + a default.
// Renders a small HTML template with headless Chrome. Run: `node scripts/gen-og.mjs`.
import { execSync } from "node:child_process";
import { mkdirSync, writeFileSync, rmSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const __dir = dirname(fileURLToPath(import.meta.url));
const root = resolve(__dir, "..");
const outDir = resolve(root, "public/og");
const tmpDir = resolve(root, ".og-tmp");
mkdirSync(outDir, { recursive: true });
mkdirSync(tmpDir, { recursive: true });

const CHROME =
  process.env.CHROME_BIN ||
  ["google-chrome", "google-chrome-stable", "chromium", "chromium-browser"].find((b) => {
    try { execSync(`command -v ${b}`, { stdio: "ignore" }); return true; } catch { return false; }
  });
if (!CHROME) { console.error("No Chrome/Chromium found (set CHROME_BIN)."); process.exit(1); }

const pages = [
  { slug: "default", title: "Amazon Shopping, for AI agents", sub: "Read-only · one normalized schema over four backends" },
  { slug: "index", title: "A read-only Amazon Shopping CLI for AI agents", sub: "SerpApi · Rainforest · official Creators API · keyless scraping" },
  { slug: "getting-started", title: "Getting started", sub: "First result in under a minute — offline, then real data" },
  { slug: "backends", title: "Backends — capability, cost & risk", sub: "Pick by what it can do, what it costs, what it risks" },
  { slug: "auth", title: "Authentication & security", sub: "stdin → OS keyring · doctor · secret threat model" },
  { slug: "commands", title: "Command reference", sub: "Every command, flag, and exit code — all read-only" },
  { slug: "agents", title: "For agents", sub: "Self-describing · fenced · token-disciplined" },
];

const card = (title, sub) => `<!doctype html><html><head><meta charset="utf-8"/>
<style>
  @import url("https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,800&family=JetBrains+Mono:wght@600&display=swap");
  *{margin:0;box-sizing:border-box}
  html,body{width:1200px;height:630px}
  body{font-family:"Bricolage Grotesque","Segoe UI",sans-serif;color:#eef4ff;
    background:#070b12;position:relative;overflow:hidden;padding:72px 84px 92px;display:flex;flex-direction:column}
  .glow{position:absolute;inset:0;background:
    radial-gradient(50rem 30rem at 88% -8%, rgba(47,230,196,.20), transparent 60%),
    radial-gradient(40rem 30rem at -6% 18%, rgba(31,143,255,.18), transparent 55%)}
  .wave{position:absolute;left:0;right:0;bottom:-110px;height:240px;opacity:.45}
  .brand{display:flex;align-items:center;gap:16px;position:relative}
  .brand svg{width:54px;height:54px}
  .brand b{font-size:40px;font-weight:800;letter-spacing:-.02em}
  h1{position:relative;font-size:68px;font-weight:800;line-height:1.03;letter-spacing:-.03em;margin-top:auto;max-width:1020px}
  .sub{position:relative;font-family:"JetBrains Mono",monospace;color:#8aa0c0;font-size:25px;margin-top:22px}
  .pills{position:relative;display:flex;gap:14px;margin-top:28px;font-family:"JetBrains Mono",monospace;font-size:19px}
  .pill{border:1px solid #243049;border-radius:999px;padding:8px 18px;color:#b9c7de;background:#0a0f1a}
  .url{position:absolute;right:84px;bottom:40px;font-family:"JetBrains Mono",monospace;color:#2fe6c4;font-size:23px}
  .grad{background:linear-gradient(100deg,#2fe6c4,#29c0e8 55%,#1f8fff);-webkit-background-clip:text;background-clip:text;color:transparent}
</style></head><body>
  <div class="glow"></div>
  <svg class="wave" viewBox="0 0 1200 240" fill="none" preserveAspectRatio="none">
    <linearGradient id="w" x1="0" y1="0" x2="1200" y2="0" gradientUnits="userSpaceOnUse"><stop stop-color="#2fe6c4"/><stop offset=".55" stop-color="#29c0e8"/><stop offset="1" stop-color="#1f8fff"/></linearGradient>
    <path d="M0 120 C 200 60, 360 180, 600 120 S 1000 60, 1200 120" stroke="url(#w)" stroke-width="4" opacity=".5"/>
    <path d="M0 160 C 220 100, 380 220, 600 160 S 1000 100, 1200 160" stroke="url(#w)" stroke-width="4" opacity=".3"/>
  </svg>
  <div class="brand">
    <svg viewBox="0 0 32 32" fill="none"><linearGradient id="g" x1="2" y1="6" x2="30" y2="26" gradientUnits="userSpaceOnUse"><stop stop-color="#2fe6c4"/><stop offset=".55" stop-color="#29c0e8"/><stop offset="1" stop-color="#1f8fff"/></linearGradient>
    <path d="M3 9c5 0 5 4 10 4s5-4 10-4 5 4 6 4" stroke="url(#g)" stroke-width="3" stroke-linecap="round"/>
    <path d="M3 16c5 0 5 4 10 4s5-4 10-4 5 4 6 4" stroke="url(#g)" stroke-width="3" stroke-linecap="round" opacity=".7"/>
    <path d="M3 23c5 0 5 4 10 4s5-4 10-4 5 4 6 4" stroke="url(#g)" stroke-width="3" stroke-linecap="round" opacity=".4"/></svg>
    <b>rivr</b>
  </div>
  <h1>${title.replace(/agents$/, '<span class="grad">agents</span>')}</h1>
  <div class="sub">${sub}</div>
  <div class="pills"><span class="pill">read-only</span><span class="pill">structured JSON</span><span class="pill">agent-safe</span></div>
  <div class="url">rivr.sh</div>
</body></html>`;

for (const p of pages) {
  const html = resolve(tmpDir, `${p.slug}.html`);
  const png = resolve(outDir, `${p.slug}.png`);
  writeFileSync(html, card(p.title, p.sub));
  execSync(
    `"${CHROME}" --headless=new --no-sandbox --disable-gpu --hide-scrollbars ` +
      `--force-device-scale-factor=1 --window-size=1200,630 --screenshot="${png}" "file://${html}"`,
    { stdio: "ignore" }
  );
  if (!existsSync(png)) { console.error("failed:", p.slug); process.exit(1); }
  console.log("og:", p.slug + ".png");
}
rmSync(tmpDir, { recursive: true, force: true });
console.log("done →", outDir);
