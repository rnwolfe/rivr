// Generate the GitHub repo social-preview card (1280×640) — a targeted, proof-forward card
// showing a real rivr invocation (structured JSON + untrusted fence + tagged deep link).
// Output: .github/social-preview.png (uploaded manually in repo Settings → Social preview).
// Run: node scripts/gen-social.mjs
import { execSync } from "node:child_process";
import { mkdirSync, writeFileSync, rmSync, existsSync } from "node:fs";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";

const __dir = dirname(fileURLToPath(import.meta.url));
const repo = resolve(__dir, "..", ".."); // site/scripts -> site -> repo root
const out = resolve(repo, ".github", "social-preview.png");
const tmp = resolve(repo, "site", ".social-tmp");
mkdirSync(dirname(out), { recursive: true });
mkdirSync(tmp, { recursive: true });

const CHROME =
  process.env.CHROME_BIN ||
  ["google-chrome", "google-chrome-stable", "chromium", "chromium-browser"].find((b) => {
    try { execSync(`command -v ${b}`, { stdio: "ignore" }); return true; } catch { return false; }
  });
if (!CHROME) { console.error("No Chrome/Chromium found (set CHROME_BIN)."); process.exit(1); }

const html = `<!doctype html><html><head><meta charset="utf-8"/>
<style>
  @import url("https://fonts.googleapis.com/css2?family=Bricolage+Grotesque:opsz,wght@12..96,800&family=JetBrains+Mono:wght@400;600&display=swap");
  *{margin:0;box-sizing:border-box}
  html,body{width:1280px;height:640px}
  body{font-family:"Bricolage Grotesque","Segoe UI",sans-serif;color:#eef4ff;background:#070b12;
    position:relative;overflow:hidden;padding:48px 64px 46px;display:flex;flex-direction:column}
  .glow{position:absolute;inset:0;background:
    radial-gradient(48rem 30rem at 92% -10%, rgba(47,230,196,.20), transparent 60%),
    radial-gradient(40rem 30rem at -8% 16%, rgba(31,143,255,.18), transparent 55%)}
  .top{display:flex;align-items:center;justify-content:space-between;position:relative}
  .brand{display:flex;align-items:center;gap:14px}
  .brand svg{width:46px;height:46px}
  .brand b{font-size:34px;font-weight:800;letter-spacing:-.02em}
  .url{font-family:"JetBrains Mono",monospace;color:#2fe6c4;font-size:22px}
  h1{position:relative;font-size:48px;font-weight:800;line-height:1.04;letter-spacing:-.03em;margin-top:20px;max-width:1140px}
  .grad{background:linear-gradient(100deg,#2fe6c4,#29c0e8 55%,#1f8fff);-webkit-background-clip:text;background-clip:text;color:transparent}
  .term{position:relative;margin-top:20px;border:1px solid #243049;border-radius:14px;background:#0b1322;
    box-shadow:0 24px 60px -28px rgba(0,0,0,.8);overflow:hidden}
  .bar{display:flex;gap:8px;padding:11px 15px;border-bottom:1px solid #1b2438;background:#0d1626}
  .bar i{width:11px;height:11px;border-radius:50%}
  .r{background:#ff6b6b}.y{background:#f6b352}.g{background:#2fe6c4}
  .code{font-family:"JetBrains Mono",monospace;font-size:18px;line-height:1.5;padding:15px 20px;white-space:pre}
  .p{color:#2fe6c4}.k{color:#8aa0c0}.s{color:#b9c7de}.n{color:#f6b352}.u{color:#7c8db0}
  .row{display:flex;align-items:center;justify-content:space-between;margin-top:24px;position:relative}
  .pills{display:flex;gap:12px;font-family:"JetBrains Mono",monospace;font-size:18px}
  .pill{border:1px solid #243049;border-radius:999px;padding:7px 16px;color:#b9c7de;background:#0a0f1a}
  .install{font-family:"JetBrains Mono",monospace;font-size:20px;color:#eef4ff;border:1px solid #243049;border-radius:10px;padding:10px 16px;background:#0d1626}
  .install .d{color:#2fe6c4}
</style></head><body>
  <div class="glow"></div>
  <div class="top">
    <div class="brand">
      <svg viewBox="0 0 32 32" fill="none"><linearGradient id="g" x1="2" y1="6" x2="30" y2="26" gradientUnits="userSpaceOnUse"><stop stop-color="#2fe6c4"/><stop offset=".55" stop-color="#29c0e8"/><stop offset="1" stop-color="#1f8fff"/></linearGradient>
      <path d="M3 9c5 0 5 4 10 4s5-4 10-4 5 4 6 4" stroke="url(#g)" stroke-width="3" stroke-linecap="round"/>
      <path d="M3 16c5 0 5 4 10 4s5-4 10-4 5 4 6 4" stroke="url(#g)" stroke-width="3" stroke-linecap="round" opacity=".7"/>
      <path d="M3 23c5 0 5 4 10 4s5-4 10-4 5 4 6 4" stroke="url(#g)" stroke-width="3" stroke-linecap="round" opacity=".4"/></svg>
      <b>rivr</b>
    </div>
    <div class="url">rivr.sh</div>
  </div>

  <h1>A read-only Amazon Shopping CLI<br/>for <span class="grad">AI agents</span>.</h1>

  <div class="term">
    <div class="bar"><i class="r"></i><i class="y"></i><i class="g"></i></div>
<div class="code"><span class="p">$</span> rivr search <span class="s">"usb-c cable"</span> --json <span class="k">| jq '.items[0]'</span>
{ <span class="k">"asin"</span>: <span class="s">"B0CXYZ123"</span>, <span class="k">"price"</span>: <span class="n">12.99</span>, <span class="k">"rating"</span>: <span class="n">4.6</span>, <span class="k">"prime"</span>: <span class="n">true</span>,
  <span class="k">"title"</span>: <span class="s">"<span class="u">‹untrusted›</span>Anker USB-C Cable<span class="u">‹/untrusted›</span>"</span>,
  <span class="k">"url"</span>: <span class="s">"https://amazon.com/dp/B0CXYZ123?tag=rivr-20"</span> }</div>
  </div>

  <div class="row">
    <div class="pills">
      <span class="pill">read-only</span><span class="pill">injection-fenced</span><span class="pill">4 backends</span><span class="pill">MIT</span>
    </div>
    <div class="install"><span class="d">$</span> brew install rnwolfe/tap/rivr</div>
  </div>
</body></html>`;

const f = resolve(tmp, "social.html");
writeFileSync(f, html);
execSync(
  `"${CHROME}" --headless=new --no-sandbox --disable-gpu --hide-scrollbars ` +
    `--force-device-scale-factor=1 --window-size=1280,640 --screenshot="${out}" "file://${f}"`,
  { stdio: "ignore" }
);
rmSync(tmp, { recursive: true, force: true });
if (!existsSync(out)) { console.error("failed to render"); process.exit(1); }
console.log("social card →", out);
