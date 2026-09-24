package main

// dashboardHTML is served unauthenticated under the plugin resource prefix, so
// it embeds no usage data. Every figure it shows is fetched at runtime from the
// management-authenticated JSON routes using a key the operator supplies.
var dashboardHTML = dashboardHead + dashboardStyle + dashboardBody +
	"<script>" + dashboardScript + dashboardTail

const dashboardHead = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>API Key Quota</title>
`

const dashboardTail = `
</script>
</body>
</html>
`

// The palette mirrors the CLIProxyAPI management center (warm gray, light,
// white and dark) so the page blends in when the host embeds it in an iframe.
// The script copies the host's data-theme onto this document when it can.
const dashboardStyle = `<style>
:root {
  color-scheme: light;
  --bg-page: #faf9f5;
  --bg-card: #f0eee8;
  --bg-muted: #e9e6df;
  --bg-hover: #e9e6df;
  --bg-field: #faf9f5;
  --floating: #fffdf9;
  --text: #2d2a26;
  --text-2: #6d6760;
  --text-3: #a29c95;
  --border: #e3e1db;
  --border-strong: #d5d2cb;
  --border-hover: #cecac4;
  --primary: #8b8680;
  --primary-hover: #7f7a74;
  --primary-contrast: #ffffff;
  --ok: #10b981;
  --ok-text: #047857;
  --warn: #d97706;
  --warn-text: #92400e;
  --bad: #c65746;
  --bad-text: #8a3a30;
  --note: #9a4d12;
  /* Chart slots: fixed order, validated for colour-vision deficiency. */
  --series-1: #2a78d6;
  --series-2: #eb6834;
  --series-3: #1baf7a;
  --series-4: #eda100;
  --series-5: #e87ba4;
  --series-6: #008300;
  --series-7: #4a3aa7;
  --series-other: #a29c95;
  --shadow: 0 1px 2px 0 rgb(0 0 0 / .08);
  --shadow-lg: 0 12px 26px rgba(0, 0, 0, .14);
  --ease: cubic-bezier(.23, 1, .32, 1);
  --mono: ui-monospace, "SF Mono", "Cascadia Mono", "JetBrains Mono", Menlo, Consolas, monospace;
}
:root[data-theme="white"] {
  --bg-page: #ffffff;
  --bg-card: #ffffff;
  --bg-muted: #f6f6f6;
  --bg-hover: #f6f6f6;
  --bg-field: #ffffff;
  --floating: #ffffff;
  --border: #e5e5e5;
  --border-strong: #d9d9d9;
  --border-hover: #cccccc;
}
:root[data-theme="dark"] {
  color-scheme: dark;
  --bg-page: #151412;
  --bg-card: #1d1b18;
  --bg-muted: #262320;
  --bg-hover: #2e2a26;
  --bg-field: #151412;
  --floating: #2a2723;
  --text: #f6f4f1;
  --text-2: #c9c3bb;
  --text-3: #9c958d;
  --border: #3a3530;
  --border-strong: #4a453f;
  --border-hover: #5a544d;
  --primary-hover: #9a948e;
  --ok: #0fae78;
  --ok-text: #6ee7b7;
  --warn: #f59e0b;
  --warn-text: #fcd34d;
  --bad: #cf4b3a;
  --bad-text: #f1b0a6;
  --note: #f2b86b;
  --series-1: #3987e5;
  --series-2: #d95926;
  --series-3: #199e70;
  --series-4: #c98500;
  --series-5: #d55181;
  --series-6: #008300;
  --series-7: #9085e9;
  --series-other: #6d675f;
  --shadow: 0 1px 3px 0 rgb(0 0 0 / .3);
  --shadow-lg: 0 14px 30px rgba(0, 0, 0, .4);
}
@media (prefers-color-scheme: dark) {
  :root:not([data-theme]) {
    color-scheme: dark;
    --bg-page: #151412;
    --bg-card: #1d1b18;
    --bg-muted: #262320;
    --bg-hover: #2e2a26;
    --bg-field: #151412;
    --floating: #2a2723;
    --text: #f6f4f1;
    --text-2: #c9c3bb;
    --text-3: #9c958d;
    --border: #3a3530;
    --border-strong: #4a453f;
    --border-hover: #5a544d;
    --primary-hover: #9a948e;
    --ok: #0fae78;
    --ok-text: #6ee7b7;
    --warn: #f59e0b;
    --warn-text: #fcd34d;
    --bad: #cf4b3a;
    --bad-text: #f1b0a6;
    --note: #f2b86b;
    --series-1: #3987e5;
    --series-2: #d95926;
    --series-3: #199e70;
    --series-4: #c98500;
    --series-5: #d55181;
    --series-6: #008300;
    --series-7: #9085e9;
    --series-other: #6d675f;
    --shadow: 0 1px 3px 0 rgb(0 0 0 / .3);
    --shadow-lg: 0 14px 30px rgba(0, 0, 0, .4);
  }
}

* { box-sizing: border-box; }
[hidden] { display: none !important; }
body {
  margin: 0;
  background: var(--bg-page);
  color: var(--text);
  font: 14px/1.5 -apple-system, BlinkMacSystemFont, "Segoe UI", "Noto Sans SC", Roboto, "Helvetica Neue", sans-serif;
  -webkit-font-smoothing: antialiased;
}
.wrap { max-width: 1280px; margin: 0 auto; padding: 24px clamp(16px, 3vw, 40px) 56px; }
svg.icon { width: 16px; height: 16px; flex-shrink: 0; display: block; }

/* Header: title on the left, page-level controls on the right. The right
   padding keeps them clear of the refresh button the management center floats
   over the top-right corner of the embedded page. */
.head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; flex-wrap: wrap; margin-bottom: 20px; padding-right: 150px; }
.head-text { min-width: 0; }
h1 { margin: 0; font-size: 22px; font-weight: 700; letter-spacing: -.01em; }
.sub { margin: 4px 0 0; color: var(--text-2); font-size: 13px; }
.head-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.statusline { display: flex; align-items: center; gap: 8px; margin-top: 10px; font-size: 12.5px; color: var(--text-2); flex-wrap: wrap; }
.statusline .dot { width: 7px; height: 7px; border-radius: 50%; background: var(--text-3); flex-shrink: 0; }
.statusline.ok .dot { background: var(--ok); }
.statusline.bad .dot { background: var(--bad); }
.statusline.bad #status { color: var(--bad); }
.statusline .sep { color: var(--text-3); }

/* Form controls */
input, select, textarea, button { font: inherit; color: var(--text); }
input, select, textarea {
  background: var(--bg-field);
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 8px 11px;
  transition: border-color .15s ease, box-shadow .15s ease;
}
input::placeholder, textarea::placeholder { color: var(--text-3); }
input:hover, select:hover, textarea:hover { border-color: var(--border-hover); }
input:focus, select:focus, textarea:focus {
  outline: none;
  border-color: var(--primary);
  box-shadow: 0 0 0 3px color-mix(in srgb, var(--primary) 20%, transparent);
}
input:disabled { opacity: .55; cursor: not-allowed; }
textarea { width: 100%; resize: vertical; font-family: var(--mono); font-size: 12.5px; line-height: 1.6; }
select { cursor: pointer; }
.mono { font-family: var(--mono); font-size: 12px; }
input.mono { font-size: 12.5px; }

.btn {
  display: inline-flex; align-items: center; justify-content: center; gap: 6px;
  border: 1px solid var(--border);
  border-radius: 8px;
  padding: 7px 13px;
  background: var(--bg-muted);
  color: var(--text);
  font-weight: 600;
  font-size: 13px;
  line-height: 1.4;
  white-space: nowrap;
  cursor: pointer;
  transition: background-color .2s var(--ease), border-color .2s var(--ease), color .2s var(--ease), transform .16s var(--ease);
}
.btn:hover { background: var(--bg-hover); border-color: var(--border-hover); }
.btn:active { transform: translateY(.5px); }
.btn:focus-visible { outline: 2px solid var(--primary); outline-offset: 1px; }
.btn:disabled { opacity: .6; cursor: not-allowed; }
.btn.primary { background: var(--primary); border-color: var(--primary); color: var(--primary-contrast); }
.btn.primary:hover { background: var(--primary-hover); border-color: var(--primary-hover); }
.btn.ghost { background: transparent; border-color: transparent; color: var(--text-2); }
.btn.ghost:hover { background: var(--bg-muted); color: var(--text); }
.btn.danger { background: transparent; border-color: color-mix(in srgb, var(--bad) 35%, transparent); color: var(--bad); }
.btn.danger:hover { background: color-mix(in srgb, var(--bad) 10%, transparent); border-color: var(--bad); }
.btn.sm { padding: 5px 10px; font-size: 12.5px; }
.btn.icon-only { padding: 7px; }
.btn.icon-only.sm { padding: 5px; }
.btn.spin svg { animation: spin .8s linear infinite; }
@keyframes spin { to { transform: rotate(360deg); } }

.seg { align-self: flex-start; display: inline-flex; padding: 3px; gap: 2px; background: var(--bg-muted); border: 1px solid var(--border); border-radius: 9px; }
.seg button { border: 0; background: transparent; padding: 5px 12px; border-radius: 6px; color: var(--text-2); font-size: 13px; font-weight: 550; cursor: pointer; }
.seg button:hover { color: var(--text); }
.seg button.on { background: var(--bg-card); color: var(--text); box-shadow: var(--shadow); }

.refresh { display: inline-flex; align-items: center; gap: 6px; color: var(--text-2); font-size: 12.5px; }
.refresh select { padding: 6px 8px; font-size: 12.5px; }

/* Switch */
.switch { display: inline-flex; align-items: center; gap: 10px; cursor: pointer; }
.switch input { position: absolute; opacity: 0; width: 0; height: 0; }
.switch .track { position: relative; width: 40px; height: 22px; border-radius: 999px; background: var(--border-strong); transition: background .15s ease; flex-shrink: 0; }
.switch .track::after { content: ""; position: absolute; top: 3px; left: 3px; width: 16px; height: 16px; border-radius: 50%; background: #fff; box-shadow: 0 1px 2px rgb(0 0 0 / .2); transition: transform .15s var(--ease); }
.switch input:checked + .track { background: var(--bad); }
.switch input:checked + .track::after { transform: translateX(18px); }
.switch input:focus-visible + .track { outline: 2px solid var(--primary); outline-offset: 2px; }

/* Summary cards */
.cards { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)); gap: 12px; margin-bottom: 22px; }
.card { background: var(--bg-card); border: 1px solid var(--border); border-radius: 12px; padding: 14px 16px; box-shadow: var(--shadow); }
.card .label { color: var(--text-2); font-size: 12px; font-weight: 550; }
.card .value { font-size: 24px; font-weight: 700; margin-top: 6px; letter-spacing: -.02em; font-variant-numeric: tabular-nums; }
.card.live .value { color: var(--ok-text); }
.card.bad .value { color: var(--bad); }

/* Tabs: underline style, matching the host's config tabs. */
.tabs { display: flex; gap: 2px; border-bottom: 1px solid var(--border); margin-bottom: 16px; overflow-x: auto; scrollbar-width: none; }
.tabs::-webkit-scrollbar { display: none; }
.tab {
  position: relative; display: inline-flex; align-items: center; gap: 7px; flex-shrink: 0;
  border: 0; background: none; cursor: pointer;
  padding: 8px 14px 11px; border-radius: 8px 8px 0 0;
  font-size: 13.5px; font-weight: 550; color: var(--text-2); white-space: nowrap;
  transition: color .2s var(--ease), background-color .2s var(--ease);
}
.tab::after { content: ""; position: absolute; left: 10px; right: 10px; bottom: -1px; height: 2px; border-radius: 999px; background: transparent; transition: background-color .2s var(--ease); }
.tab:hover { color: var(--text); background: color-mix(in srgb, var(--bg-muted) 55%, transparent); }
.tab:focus-visible { outline: 2px solid var(--primary); outline-offset: -2px; }
.tab.on { color: var(--text); font-weight: 650; }
.tab.on::after { background: var(--text); }
.tab svg { color: var(--text-3); }
.tab.on svg { color: var(--text); }
.tab .count { font-family: var(--mono); font-size: 11px; font-weight: 600; padding: 0 7px; border-radius: 999px; background: var(--bg-muted); color: var(--text-2); line-height: 1.7; }
.tab .count:empty { display: none; }
.tab .dirty { width: 6px; height: 6px; border-radius: 50%; background: var(--warn); }

/* Section cards */
.panel { background: var(--bg-card); border: 1px solid var(--border); border-radius: 12px; box-shadow: var(--shadow); padding: 20px; }
.panel + .panel { margin-top: 16px; }
.panel-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; flex-wrap: wrap; margin-bottom: 14px; }
.panel-head > div { min-width: 0; flex: 1 1 320px; }
h2 { margin: 0; font-size: 16px; font-weight: 700; }
.panel-head .hint { margin: 4px 0 0; }
.panel-foot { display: flex; align-items: center; justify-content: flex-end; gap: 8px; flex-wrap: wrap; margin-top: 14px; padding-top: 14px; border-top: 1px solid var(--border); }
.panel-foot .spacer { flex: 1 1 auto; }
.panel-toolbar { display: flex; align-items: center; gap: 10px; flex-wrap: wrap; margin-bottom: 12px; }
.search { position: relative; flex: 0 1 300px; }
.search svg { position: absolute; left: 10px; top: 50%; transform: translateY(-50%); color: var(--text-3); }
.search input { width: 100%; padding-left: 32px; }

/* Tables: bordered shell with a muted header row. */
.tablewrap { border: 1px solid var(--border); border-radius: 8px; overflow-x: auto; background: var(--bg-card); scrollbar-width: thin; }
table { width: 100%; border-collapse: separate; border-spacing: 0; font-size: 13px; }
th, td { text-align: left; padding: 11px 12px; border-bottom: 1px solid var(--border); vertical-align: top; white-space: nowrap; }
thead th { background: var(--bg-muted); color: var(--text-2); font-size: 12px; font-weight: 550; vertical-align: middle; position: sticky; top: 0; }
tbody tr { transition: background-color .15s ease; }
tbody tr:hover td { background: color-mix(in srgb, var(--bg-muted) 50%, transparent); }
tbody tr:last-child td { border-bottom: 0; }
td.num, th.num { text-align: right; font-variant-numeric: tabular-nums; }
td.actions { text-align: right; }
/* The key table can outgrow narrow screens; its actions stay pinned in view. */
#panelKeys th:last-child, #panelKeys td.actions { position: sticky; right: 0; background: var(--bg-card); box-shadow: inset 1px 0 0 var(--border); }
#panelKeys thead th:last-child { background: var(--bg-muted); z-index: 1; }
tbody tr.live td:first-child { box-shadow: inset 3px 0 0 var(--ok); }
tbody tr.paused td { color: var(--text-2); }
tr.total td { font-weight: 650; background: var(--bg-muted); }
td.empty { padding: 28px 14px; text-align: center; color: var(--text-2); white-space: normal; }
td.empty .btn { margin-top: 10px; }
table input { padding: 6px 9px; }

/* The note is how operators tell keys apart, so it leads the cell. */
.key-note { font-size: 15px; font-weight: 650; line-height: 1.35; color: var(--note); margin-bottom: 2px; }
.key-name { font-weight: 600; }
.key-note + .key-name { font-weight: 500; color: var(--text-2); }
.meter { height: 4px; border-radius: 999px; background: var(--border); overflow: hidden; margin: 3px 0 5px; min-width: 96px; }
.meter > span { display: block; height: 100%; background: var(--ok); border-radius: inherit; }
.meter.warn > span { background: var(--warn); }
.meter.bad > span { background: var(--bad); }
.dimline { display: flex; gap: 6px; align-items: baseline; font-variant-numeric: tabular-nums; }
.dimline .tag { color: var(--text-3); font-family: var(--mono); font-size: 11px; width: 10px; }

.pill { display: inline-flex; align-items: center; gap: 5px; border: 1px solid var(--border); border-radius: 999px; padding: 0 8px; font-size: 11.5px; line-height: 1.7; color: var(--text-2); background: var(--bg-page); white-space: nowrap; }
.pill.on { color: var(--text); border-color: var(--border-strong); background: var(--bg-muted); }
.pill.bad { color: var(--bad-text); border-color: color-mix(in srgb, var(--bad) 35%, transparent); background: color-mix(in srgb, var(--bad) 12%, transparent); }
.pill.warn { color: var(--warn-text); border-color: color-mix(in srgb, var(--warn) 35%, transparent); background: color-mix(in srgb, var(--warn) 10%, transparent); }
.pill.live { color: var(--ok-text); border-color: color-mix(in srgb, var(--ok) 40%, transparent); background: color-mix(in srgb, var(--ok) 10%, transparent); }
.pill.live::before { content: ""; width: 6px; height: 6px; border-radius: 50%; background: var(--ok); animation: livePulse 1.4s ease-in-out infinite; }
@keyframes livePulse { 50% { opacity: .25; } }
@media (prefers-reduced-motion: reduce) { .pill.live::before, .btn.spin svg { animation: none; } }
.badges { display: flex; gap: 4px; margin-top: 5px; }

.hint { color: var(--text-2); font-size: 12.5px; }
.faint { color: var(--text-3); }
.err { color: var(--bad); }
.stack { display: flex; flex-direction: column; gap: 1px; }
.wrapcell { white-space: normal; max-width: 260px; word-break: break-word; }
.rowbtns { display: inline-flex; gap: 6px; align-items: center; }

/* Row selection and the batch bar that acts on it. */
input.pick { padding: 0; width: 16px; height: 16px; margin: 0; accent-color: var(--primary); cursor: pointer; vertical-align: middle; box-shadow: none; }
th.sel, td.sel { width: 1%; padding-right: 0; }
tbody tr.picked td { background: color-mix(in srgb, var(--primary) 6%, transparent); }
.selbar { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; margin-left: auto; padding: 4px 6px 4px 12px; border: 1px solid var(--border-strong); border-radius: 8px; background: var(--bg-muted); }
.selbar[hidden] { display: none; }
.selcount { font-size: 12.5px; font-weight: 650; margin-right: 4px; }
.targets { list-style: none; margin: 0 0 14px; padding: 8px 12px; border: 1px solid var(--border); border-radius: 8px; background: var(--bg-page); max-height: 160px; overflow-y: auto; font-size: 13px; }
.targets li { display: flex; gap: 8px; align-items: baseline; padding: 2px 0; }
.targets li .mono { color: var(--text-3); }
.targets[hidden] { display: none; }

/* Live rate column and schedule editor. */
.rateline .tag { width: 30px; }
.win-list { display: flex; flex-direction: column; gap: 10px; margin-bottom: 10px; }
.win { border: 1px solid var(--border); border-radius: 8px; background: var(--bg-page); padding: 12px; display: flex; flex-direction: column; gap: 10px; }
.win-row { display: grid; grid-template-columns: minmax(0, 1fr) 132px 132px minmax(0, 1fr) auto; gap: 8px; align-items: center; }
.win-row input, .win-row select { width: 100%; min-width: 0; }
.win-days { display: flex; gap: 4px; flex-wrap: wrap; }
.win-days button { border: 1px solid var(--border); background: var(--bg-card); color: var(--text-2); border-radius: 999px; padding: 2px 10px; font-size: 12px; cursor: pointer; }
.win-days button.on { background: var(--primary); border-color: var(--primary); color: var(--primary-contrast); }
.win-limits { display: grid; grid-template-columns: 72px minmax(0, 1fr) minmax(0, 1fr); gap: 8px 10px; align-items: center; }
.win-limits .period { font-size: 12.5px; color: var(--text-2); }
.win-limits input, .win-limits select { width: 100%; min-width: 0; }
.win-limits .rates { display: grid; grid-template-columns: repeat(3, minmax(0, 1fr)); gap: 8px; grid-column: 2 / -1; }
.rateGrid { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.rateGrid label { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--text-2); }
.rateGrid input { width: 100%; min-width: 0; }
.sched-body[hidden] { display: none; }

/* Pie charts: a donut beside its legend; the table below stays the exact view. */
.viz { display: flex; gap: 20px; align-items: center; flex-wrap: wrap; padding: 14px; border: 1px solid var(--border); border-radius: 8px; background: var(--bg-card); margin-bottom: 14px; }
.viz[hidden] { display: none; }
.viz svg { width: 180px; height: 180px; flex-shrink: 0; overflow: visible; }
.viz path { stroke: var(--bg-card); stroke-width: 2; stroke-linejoin: round; cursor: pointer; transition: opacity .15s ease; }
.viz.focus path:not(.hot) { opacity: .35; }
.viz .center-value { font-size: 15px; font-weight: 700; fill: var(--text); }
.viz .center-label { font-size: 11px; fill: var(--text-2); }
.viz-side { flex: 1 1 240px; max-width: 520px; min-width: 0; display: flex; flex-direction: column; gap: 10px; }
.viz-metrics { display: inline-flex; border: 1px solid var(--border); border-radius: 8px; overflow: hidden; align-self: flex-start; }
.viz-metrics button { border: 0; background: transparent; color: var(--text-2); padding: 4px 12px; font-size: 12.5px; cursor: pointer; }
.viz-metrics button + button { border-left: 1px solid var(--border); }
.viz-metrics button.on { background: var(--bg-muted); color: var(--text); font-weight: 650; }
.legend { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; font-size: 12.5px; }
.legend li { display: grid; grid-template-columns: 10px minmax(0, 1fr) auto 52px; gap: 8px; align-items: center; padding: 3px 6px; border-radius: 6px; cursor: default; }
.legend li:hover, .legend li.hot { background: var(--bg-muted); }
.legend .sw { width: 10px; height: 10px; border-radius: 3px; }
.legend .name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; font-family: var(--mono); font-size: 12px; }
.legend .val { font-variant-numeric: tabular-nums; color: var(--text); }
.legend .pct { font-variant-numeric: tabular-nums; color: var(--text-2); text-align: right; }
.viz-empty { color: var(--text-2); font-size: 13px; }
.viz-tip { position: fixed; z-index: 50; pointer-events: none; background: var(--floating); color: var(--text); border: 1px solid var(--border-strong); border-radius: 8px; box-shadow: var(--shadow-lg); padding: 7px 10px; font-size: 12.5px; line-height: 1.5; white-space: nowrap; }
.viz-tip[hidden] { display: none; }
.viz-tip .mono { font-weight: 650; }
.headSwitch { display: inline-flex; align-items: center; gap: 8px; font-size: 13px; color: var(--text-2); cursor: pointer; }

.source { display: inline-flex; }

/* Dialogs follow the host modal: header, scrolling body, footer. */
dialog {
  border: 1px solid var(--border);
  border-radius: 12px;
  background: var(--bg-card);
  color: var(--text);
  padding: 0;
  width: min(560px, calc(100vw - 32px));
  max-height: calc(100vh - 48px);
  box-shadow: var(--shadow-lg);
}
dialog.wide { width: min(720px, calc(100vw - 32px)); }
dialog[open] { display: flex; flex-direction: column; animation: dlgIn .25s var(--ease); }
@keyframes dlgIn { from { opacity: 0; transform: translateY(8px) scale(.98); } }
dialog::backdrop { background: rgba(0, 0, 0, .35); }
dialog form { display: flex; flex-direction: column; min-height: 0; max-height: inherit; }
.modal-head { display: flex; align-items: flex-start; gap: 12px; padding: 16px 20px; border-bottom: 1px solid var(--border); }
.modal-head > div { flex: 1 1 auto; min-width: 0; }
.modal-head h3 { margin: 0; font-size: 17px; font-weight: 700; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.modal-head .hint { margin: 3px 0 0; }
.modal-close { border: 1px solid var(--border); background: var(--bg-page); color: var(--text-2); width: 30px; height: 30px; border-radius: 50%; display: inline-flex; align-items: center; justify-content: center; cursor: pointer; flex-shrink: 0; transition: background-color .15s ease, color .15s ease; }
.modal-close:hover { background: var(--bg-muted); color: var(--text); }
.modal-body { padding: 18px 20px; overflow: auto; min-height: 0; }
.modal-foot { display: flex; align-items: center; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid var(--border); flex-wrap: wrap; }
.modal-foot .spacer { flex: 1 1 auto; }
.modal-foot .err { margin: 0; flex: 1 1 100%; white-space: pre-line; }
.modal-foot .err:empty { display: none; }

.field { display: flex; flex-direction: column; gap: 6px; margin-bottom: 16px; }
.field > .label { font-size: 13px; font-weight: 600; }
.field .hint { font-size: 12px; }
.inline { display: flex; gap: 8px; align-items: center; }
.inline > input, .inline > select, .inline > textarea { flex: 1 1 auto; min-width: 0; }
.inline.top { align-items: flex-start; }
#newKeyOut { white-space: pre; overflow-wrap: normal; overflow-x: auto; resize: none; }

.section { border-top: 1px solid var(--border); padding-top: 16px; margin-top: 4px; margin-bottom: 16px; }
.section:last-child { margin-bottom: 0; }
.section-title { font-size: 13px; font-weight: 700; margin: 0 0 10px; }
.limitGrid { display: grid; grid-template-columns: 72px minmax(0, 1fr) minmax(0, 1fr); gap: 8px 10px; align-items: center; }
.limitGrid .period { font-size: 13px; color: var(--text-2); font-weight: 550; }
.limitGrid input, .limitGrid select { width: 100%; min-width: 0; }
.notice { margin: 0; padding: 12px 14px 12px 30px; border: 1px solid color-mix(in srgb, var(--warn) 35%, transparent); border-radius: 8px; background: color-mix(in srgb, var(--warn) 8%, transparent); color: var(--text); font-size: 13px; }
.notice li + li { margin-top: 4px; }
.warn-text { color: var(--warn-text); }
.toggleRow { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 10px 12px; border: 1px solid var(--border); border-radius: 8px; background: var(--bg-page); margin-bottom: 16px; }
.toggleRow .hint { margin: 2px 0 0; }

.taglist { list-style: none; margin: 0 0 10px; padding: 0; display: flex; flex-wrap: wrap; gap: 6px; }
.taglist li {
  display: inline-flex; align-items: center; gap: 2px;
  border: 1px solid var(--border); background: var(--bg-page);
  border-radius: 999px; padding: 2px 3px 2px 10px;
  font-family: var(--mono); font-size: 12px;
}
.taglist li.empty { border: 1px dashed var(--border); background: transparent; border-radius: 8px; padding: 8px 12px; color: var(--text-2); font-family: inherit; width: 100%; }
.taglist button { border: 0; background: transparent; width: 20px; height: 20px; border-radius: 50%; color: var(--text-3); cursor: pointer; line-height: 1; }
.taglist button:hover { color: var(--bad); background: color-mix(in srgb, var(--bad) 12%, transparent); }
.mapGrid { display: grid; grid-template-columns: 1fr 1fr .8fr auto; gap: 8px; align-items: start; }
.mapGrid input, .mapGrid select { width: 100%; min-width: 0; }
.mapFrom { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.priceGrid { display: grid; grid-template-columns: repeat(5, minmax(0, 1fr)); gap: 10px; padding: 14px; border: 1px solid var(--border); border-radius: 8px; background: var(--bg-page); margin-bottom: 14px; }
.priceGrid .title { grid-column: 1 / -1; font-size: 13px; font-weight: 650; }
.priceGrid label { display: flex; flex-direction: column; gap: 5px; font-size: 12px; color: var(--text-2); }
.priceGrid input { width: 100%; min-width: 0; }

@media (max-width: 720px) {
  .priceGrid { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .mapGrid { grid-template-columns: 1fr 1fr; }
  .win-row { grid-template-columns: 1fr 1fr; }
  .rateGrid, .win-limits .rates { grid-template-columns: 1fr; }
  .win-limits { grid-template-columns: 1fr 1fr; }
  .head { padding-right: 0; }
  .limitGrid { grid-template-columns: 1fr 1fr; }
  .limitGrid .period { grid-column: 1 / -1; margin-top: 4px; }
  .panel { padding: 16px; }
  .modal-foot .btn { flex: 1 1 auto; }
}
</style>
`

const dashboardBody = `</head>
<body>
<svg width="0" height="0" style="position:absolute" aria-hidden="true">
  <symbol id="i-refresh" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M21 12a9 9 0 1 1-2.64-6.36"/><path d="M21 3v6h-6"/></symbol>
  <symbol id="i-settings" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 0 0 .33 1.82l.06.06a2 2 0 1 1-2.83 2.83l-.06-.06a1.65 1.65 0 0 0-1.82-.33 1.65 1.65 0 0 0-1 1.51V21a2 2 0 1 1-4 0v-.09A1.65 1.65 0 0 0 9 19.4a1.65 1.65 0 0 0-1.82.33l-.06.06a2 2 0 1 1-2.83-2.83l.06-.06A1.65 1.65 0 0 0 4.68 15a1.65 1.65 0 0 0-1.51-1H3a2 2 0 1 1 0-4h.09A1.65 1.65 0 0 0 4.6 9a1.65 1.65 0 0 0-.33-1.82l-.06-.06a2 2 0 1 1 2.83-2.83l.06.06A1.65 1.65 0 0 0 9 4.68a1.65 1.65 0 0 0 1-1.51V3a2 2 0 1 1 4 0v.09a1.65 1.65 0 0 0 1 1.51 1.65 1.65 0 0 0 1.82-.33l.06-.06a2 2 0 1 1 2.83 2.83l-.06.06A1.65 1.65 0 0 0 19.4 9a1.65 1.65 0 0 0 1.51 1H21a2 2 0 1 1 0 4h-.09a1.65 1.65 0 0 0-1.51 1z"/></symbol>
  <symbol id="i-x" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M18 6 6 18M6 6l12 12"/></symbol>
  <symbol id="i-key" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="7.5" cy="15.5" r="4.5"/><path d="m10.7 12.3 9.3-9.3M16 7l3 3M19 4l2 2"/></symbol>
  <symbol id="i-route" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="6" cy="19" r="3"/><circle cx="18" cy="5" r="3"/><path d="M9 19h8.5a3.5 3.5 0 0 0 0-7h-11a3.5 3.5 0 0 1 0-7H15"/></symbol>
  <symbol id="i-tag" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><path d="M20.6 13.4 13.4 20.6a2 2 0 0 1-2.8 0L3 13V3h10l7.6 7.6a2 2 0 0 1 0 2.8z"/><circle cx="7.5" cy="7.5" r="1.5"/></symbol>
  <symbol id="i-search" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><circle cx="11" cy="11" r="7"/><path d="m20 20-3.5-3.5"/></symbol>
  <symbol id="i-plus" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round"><path d="M12 5v14M5 12h14"/></symbol>
  <symbol id="i-users" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round"><circle cx="9" cy="8" r="4"/><path d="M2 21v-1a6 6 0 0 1 6-6h2a6 6 0 0 1 6 6v1"/><path d="M16 4.1a4 4 0 0 1 0 7.8M22 21v-1a6 6 0 0 0-4-5.7"/></symbol>
</svg>

<div class="wrap">
  <header class="head">
    <div class="head-text">
      <h1 data-i18n="title">API Key Quota</h1>
      <p class="sub" data-i18n="subtitle">Usage and limits per client API key, served by the apikey-quota plugin.</p>
      <div class="statusline" id="statusLine">
        <span class="dot"></span>
        <span id="status"></span>
      </div>
    </div>
    <div class="head-actions">
      <label class="refresh">
        <span data-i18n="autoRefresh">Auto refresh</span>
        <select id="interval">
          <option value="0" data-i18n="off">off</option>
          <option value="10">10s</option>
          <option value="30" selected>30s</option>
          <option value="60">60s</option>
        </select>
      </label>
      <button type="button" id="reload" class="btn primary">
        <svg class="icon"><use href="#i-refresh"/></svg><span data-i18n="reload">Reload</span>
      </button>
      <button type="button" id="openSettings" class="btn icon-only" data-i18n-title="settings" title="Settings">
        <svg class="icon"><use href="#i-settings"/></svg>
      </button>
    </div>
  </header>

  <section class="cards" id="cards"></section>

  <nav class="tabs" role="tablist">
    <button type="button" class="tab on" id="tabKeys" data-tab="keys" role="tab">
      <svg class="icon"><use href="#i-key"/></svg><span data-i18n="tabKeys">Keys</span><span class="count" id="countKeys"></span>
    </button>
    <button type="button" class="tab" id="tabRules" data-tab="rules" role="tab">
      <svg class="icon"><use href="#i-route"/></svg><span data-i18n="tabRules">Model rules</span><span class="count" id="countRules"></span><span class="dirty" id="dirtyRules" hidden></span>
    </button>
    <button type="button" class="tab" id="tabAccounts" data-tab="accounts" role="tab">
      <svg class="icon"><use href="#i-users"/></svg><span data-i18n="tabAccounts">Accounts</span><span class="count" id="countAccounts"></span><span class="dirty" id="dirtyAccounts" hidden></span>
    </button>
    <button type="button" class="tab" id="tabPricing" data-tab="pricing" role="tab">
      <svg class="icon"><use href="#i-tag"/></svg><span data-i18n="tabPricing">Pricing</span><span class="count" id="countPricing"></span>
    </button>
  </nav>

  <section id="panelKeys" class="panel">
    <div class="panel-toolbar">
      <label class="search">
        <svg class="icon"><use href="#i-search"/></svg>
        <input id="keyFilter" type="search" data-i18n-ph="phFilter" autocomplete="off" spellcheck="false">
      </label>
      <span class="hint" id="keyFilterInfo"></span>
      <div class="selbar" id="selBar" hidden>
        <span class="selcount" id="selCount"></span>
        <button type="button" class="btn sm ghost" id="batchPause" data-i18n="pauseBtn">Pause</button>
        <button type="button" class="btn sm ghost" id="batchEnable" data-i18n="enableBtn">Enable</button>
        <button type="button" class="btn sm ghost" id="batchRotate" data-i18n="rotateBtn">Rotate</button>
        <button type="button" class="btn sm ghost" id="batchReset" data-i18n="resetBtn">Reset</button>
        <button type="button" class="btn sm danger" id="batchDelete" data-i18n="deleteBtn">Delete</button>
        <button type="button" class="btn sm ghost icon-only" id="batchClear" data-i18n-title="clearSelection" title="Clear selection"><svg class="icon"><use href="#i-x"/></svg></button>
      </div>
    </div>
    <div class="tablewrap">
      <table>
        <thead>
          <tr>
            <th class="sel"><input type="checkbox" class="pick" id="selectAll" data-i18n-title="selectAll" title="Select all"></th>
            <th data-i18n="colKey">Key</th>
            <th data-i18n="colActive">Live</th>
            <th data-i18n="colDaily">Daily</th>
            <th data-i18n="colMonthly">Monthly</th>
            <th data-i18n="colTotal">Total</th>
            <th data-i18n="colModels">Blocked models</th>
            <th class="num" data-i18n="colBlocked">Rejected</th>
            <th data-i18n="colLastUsed">Last used</th>
            <th class="num" data-i18n="colActions">Actions</th>
          </tr>
        </thead>
        <tbody id="rows"></tbody>
      </table>
    </div>
  </section>

  <section id="panelRules" hidden>
    <div class="panel">
      <div class="panel-head">
        <div>
          <h2 data-i18n="globalBlocked">Globally blocked models</h2>
          <p class="hint" data-i18n="globalBlockedHint">One glob pattern per line. Every API key is refused these models with 403.</p>
        </div>
        <span id="blockedState" class="pill"></span>
      </div>
      <textarea id="globalBlocked" rows="5" spellcheck="false"></textarea>
      <div class="panel-foot">
        <button type="button" id="clearBlocked" class="btn ghost" data-i18n="useConfig">Use configuration</button>
        <button type="button" id="saveBlocked" class="btn primary" data-i18n="save">Save</button>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h2 data-i18n="mappings">Model mappings</h2>
          <p class="hint" data-i18n="mappingsHint">Rewrites a requested model to another model before routing. The first matching rule wins, and a key's own rules are checked before these.</p>
          <p class="hint faint" id="providersHint"></p>
        </div>
        <span id="mappingsState" class="pill"></span>
      </div>
      <div class="tablewrap">
        <table>
          <thead>
            <tr>
              <th data-i18n="colFrom">Requested model</th>
              <th data-i18n="colTo">Mapped to</th>
              <th data-i18n="colProvider">Provider</th>
              <th></th>
            </tr>
          </thead>
          <tbody id="mappingRows"></tbody>
        </table>
      </div>
      <datalist id="providerSuggestions"></datalist>
      <div class="panel-foot">
        <button type="button" id="addMapping" class="btn">
          <svg class="icon"><use href="#i-plus"/></svg><span data-i18n="addRule">Add rule</span>
        </button>
        <span class="spacer"></span>
        <button type="button" id="clearMappings" class="btn ghost" data-i18n="useConfig">Use configuration</button>
        <button type="button" id="saveMappings" class="btn primary" data-i18n="save">Save</button>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h2 data-i18n="fallbacks">Fallback models</h2>
          <p class="hint" data-i18n="fallbacksHint">One model per line, tried in order when the requested model fails upstream. Applies to keys without a list of their own.</p>
        </div>
        <span id="fallbacksState" class="pill"></span>
      </div>
      <textarea id="globalFallbacks" rows="4" spellcheck="false"></textarea>
      <div class="panel-foot">
        <button type="button" id="clearFallbacks" class="btn ghost" data-i18n="useConfig">Use configuration</button>
        <button type="button" id="saveFallbacks" class="btn primary" data-i18n="save">Save</button>
      </div>
    </div>
  </section>

  <section id="panelAccounts" hidden>
    <div class="panel">
      <div class="panel-head">
        <div>
          <h2 data-i18n="accountsTitle">Accounts</h2>
          <p class="hint" data-i18n="accountsHint">Host credentials with the requests they are serving right now, the rule that applies to them, and the keys bound to them.</p>
          <p class="hint warn-text" id="accountsListError" hidden></p>
        </div>
      </div>
      <div class="tablewrap">
        <table>
          <thead>
            <tr>
              <th data-i18n="colAccount">Account</th>
              <th data-i18n="colLive">Live</th>
              <th data-i18n="colRule">Rule</th>
              <th data-i18n="colBoundKeys">Bound keys</th>
            </tr>
          </thead>
          <tbody id="accountRows"></tbody>
        </table>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h2 data-i18n="accountRules">Account rules</h2>
          <p class="hint" data-i18n="accountRulesHint">Matched against the account ID, first matching rule wins. Concurrency caps the requests an account serves at once, blank or 0 for unlimited. An exclusive account only serves the keys bound to it.</p>
        </div>
        <span id="accountRulesState" class="pill"></span>
      </div>
      <div class="tablewrap">
        <table>
          <thead>
            <tr>
              <th data-i18n="colPattern">Pattern</th>
              <th data-i18n="colConcurrency">Concurrency</th>
              <th data-i18n="colExclusive">Exclusive</th>
              <th></th>
            </tr>
          </thead>
          <tbody id="accountRuleRows"></tbody>
        </table>
      </div>
      <datalist id="accountSuggestions"></datalist>
      <div class="panel-foot">
        <button type="button" id="addAccountRule" class="btn">
          <svg class="icon"><use href="#i-plus"/></svg><span data-i18n="addRule">Add rule</span>
        </button>
        <span class="spacer"></span>
        <button type="button" id="clearAccountRules" class="btn ghost" data-i18n="useConfig">Use configuration</button>
        <button type="button" id="saveAccountRules" class="btn primary" data-i18n="save">Save</button>
      </div>
    </div>
  </section>

  <section id="panelPricing" hidden>
    <div class="panel">
      <div class="panel-head">
        <div>
          <h2 data-i18n="priceTable">Price table</h2>
          <p class="hint" data-i18n="priceHint">USD per one million tokens. The longest matching pattern wins; the default applies when nothing matches.</p>
        </div>
        <span class="rowbtns">
          <span id="priceSource" class="pill"></span>
          <span id="pricingState" class="pill"></span>
        </span>
      </div>
      <div class="priceGrid">
        <div class="title" data-i18n="defaultPrice">Default price</div>
        <label><span data-i18n="pInput">Input</span><input type="number" step="any" min="0" id="p_def_input"></label>
        <label><span data-i18n="pOutput">Output</span><input type="number" step="any" min="0" id="p_def_output"></label>
        <label><span data-i18n="pReasoning">Reasoning</span><input type="number" step="any" min="0" id="p_def_reasoning"></label>
        <label><span data-i18n="pCacheRead">Cache read</span><input type="number" step="any" min="0" id="p_def_cache_read"></label>
        <label><span data-i18n="pCacheWrite">Cache write</span><input type="number" step="any" min="0" id="p_def_cache_write"></label>
      </div>
      <div class="tablewrap">
        <table>
          <thead>
            <tr>
              <th data-i18n="colPattern">Pattern</th>
              <th class="num" data-i18n="pInput">Input</th>
              <th class="num" data-i18n="pOutput">Output</th>
              <th class="num" data-i18n="pReasoning">Reasoning</th>
              <th class="num" data-i18n="pCacheRead">Cache read</th>
              <th class="num" data-i18n="pCacheWrite">Cache write</th>
              <th></th>
            </tr>
          </thead>
          <tbody id="priceRows"></tbody>
        </table>
      </div>
      <div class="panel-foot">
        <button type="button" id="addPrice" class="btn">
          <svg class="icon"><use href="#i-plus"/></svg><span data-i18n="addRow">Add model</span>
        </button>
        <span class="spacer"></span>
        <button type="button" id="clearPricing" class="btn ghost" data-i18n="useConfig">Use configuration</button>
        <button type="button" id="savePricing" class="btn primary" data-i18n="save">Save</button>
      </div>
    </div>

    <div class="panel">
      <div class="panel-head">
        <div>
          <h2 data-i18n="observed">Recognized models</h2>
          <p class="hint" data-i18n="observedHint">Models the plugin has actually billed, with the price rule each one resolved to.</p>
        </div>
        <label class="headSwitch">
          <span data-i18n="showPie">Show pie chart</span>
          <span class="switch"><input type="checkbox" id="observedPie"><span class="track"></span></span>
        </label>
      </div>
      <div class="viz" id="observedViz" hidden></div>
      <div class="tablewrap">
        <table>
          <thead>
            <tr>
              <th data-i18n="colModel">Model</th>
              <th data-i18n="colMatched">Matched rule</th>
              <th class="num" data-i18n="pInput">Input</th>
              <th class="num" data-i18n="pOutput">Output</th>
              <th class="num" data-i18n="colTokens">Tokens</th>
              <th class="num" data-i18n="colRequests">Requests</th>
              <th class="num" data-i18n="colCost">Cost</th>
              <th></th>
            </tr>
          </thead>
          <tbody id="observedRows"></tbody>
        </table>
      </div>
    </div>
  </section>
</div>

<dialog id="settingsDialog">
  <form method="dialog">
    <div class="modal-head">
      <div>
        <h3 data-i18n="settings">Settings</h3>
        <p class="hint" data-i18n="settingsHint">The key is kept in this browser tab only and is sent with every request.</p>
      </div>
      <button type="button" class="modal-close" data-close data-i18n-title="close" title="Close"><svg class="icon"><use href="#i-x"/></svg></button>
    </div>
    <div class="modal-body">
      <label class="field">
        <span class="label" data-i18n="mgmtKey">Management key</span>
        <span class="inline">
          <input id="mgmtKey" type="password" data-i18n-ph="mgmtKeyHint"
                 name="apikey-quota-management-key" autocomplete="new-password"
                 data-lpignore="true" data-1p-ignore data-bwignore>
          <button type="button" id="revealKey" class="btn" data-i18n="show">Show</button>
        </span>
      </label>
      <div class="field" style="margin-bottom:0">
        <span class="label" data-i18n="language">Language</span>
        <span class="seg">
          <button type="button" id="langEn">English</button>
          <button type="button" id="langZh">中文</button>
        </span>
      </div>
    </div>
    <div class="modal-foot">
      <button type="button" class="btn" data-close data-i18n="cancel">Cancel</button>
      <button type="button" class="btn primary" id="connect" data-i18n="connect">Connect</button>
    </div>
  </form>
</dialog>

<dialog id="modelsDialog" class="wide">
  <form method="dialog">
    <div class="modal-head">
      <div>
        <h3 id="modelsTitle">Model usage</h3>
        <p class="hint" data-i18n="modelsHint">Totals this key has accumulated per model, across the whole tracking history.</p>
      </div>
      <button type="button" class="modal-close" data-close data-i18n-title="close" title="Close"><svg class="icon"><use href="#i-x"/></svg></button>
    </div>
    <div class="modal-body">
      <div class="viz" id="modelsViz" hidden></div>
      <div class="tablewrap">
        <table>
          <thead>
            <tr>
              <th data-i18n="colModel">Model</th>
              <th class="num" data-i18n="colRequests">Requests</th>
              <th class="num" data-i18n="colTokens">Tokens</th>
              <th class="num" data-i18n="colCost">Cost</th>
              <th class="num" data-i18n="colShare">Share</th>
            </tr>
          </thead>
          <tbody id="modelsRows"></tbody>
        </table>
      </div>
    </div>
    <div class="modal-foot">
      <button type="button" class="btn primary" data-close data-i18n="close">Close</button>
    </div>
  </form>
</dialog>

<dialog id="resetDialog">
  <form method="dialog">
    <div class="modal-head">
      <div>
        <h3 id="resetTitle">Reset counters</h3>
        <p class="hint" data-i18n="resetHint">Clears the accumulated counters of the selected period. This cannot be undone.</p>
      </div>
      <button type="button" class="modal-close" data-close data-i18n-title="close" title="Close"><svg class="icon"><use href="#i-x"/></svg></button>
    </div>
    <div class="modal-body">
      <ul class="targets" id="resetTargets" hidden></ul>
      <label class="field" style="margin-bottom:0">
        <span class="label" data-i18n="resetPeriod">Period</span>
        <select id="resetSelect">
          <option value="daily" data-i18n="colDaily">Daily</option>
          <option value="monthly" data-i18n="colMonthly">Monthly</option>
          <option value="total" data-i18n="colTotal">Total</option>
          <option value="all" data-i18n="resetAll">Everything, including the model breakdown and reject counter</option>
        </select>
      </label>
    </div>
    <div class="modal-foot">
      <p id="resetError" class="err hint"></p>
      <button type="button" class="btn" data-close data-i18n="cancel">Cancel</button>
      <button type="button" class="btn danger" id="confirmReset" data-i18n="resetBtn">Reset</button>
    </div>
  </form>
</dialog>

<dialog id="rotateDialog">
  <form method="dialog">
    <div class="modal-head">
      <div>
        <h3 id="rotateTitle">Rotate key</h3>
        <p class="hint" data-i18n="rotateHint">Replace this key with a newly generated random key.</p>
      </div>
      <button type="button" class="modal-close" data-close data-i18n-title="close" title="Close"><svg class="icon"><use href="#i-x"/></svg></button>
    </div>
    <div class="modal-body">
      <ul class="targets" id="rotateTargets" hidden></ul>
      <ul class="notice" id="rotateConfirm">
        <li data-i18n="rotateWarnOld">The old key stops working immediately; clients using it must switch to the new key.</li>
        <li data-i18n="rotateWarnMove">Usage, limits, note, blocked models and mappings move to the new key.</li>
        <li data-i18n="rotateWarnOnce">The new key is shown only once. Copy it right away.</li>
      </ul>
      <div id="rotateResult" hidden>
        <label class="field">
          <span class="label" data-i18n="newKey">New key</span>
          <span class="inline top">
            <textarea id="newKeyOut" rows="1" readonly spellcheck="false" autocomplete="off"></textarea>
            <button type="button" id="copyNewKey" class="btn primary" data-i18n="copy">Copy</button>
          </span>
        </label>
        <p class="hint" data-i18n="rotateWarnOnce">The new key is shown only once. Copy it right away.</p>
        <p class="hint warn-text" id="rotateConfigured" data-i18n="rotateConfigured" hidden></p>
      </div>
    </div>
    <div class="modal-foot">
      <p id="rotateError" class="err hint"></p>
      <button type="button" class="btn" id="rotateCancel" data-close data-i18n="cancel">Cancel</button>
      <button type="button" class="btn danger" id="rotateGo" data-i18n="rotateGo">Generate and replace</button>
      <button type="button" class="btn" id="rotateRetry" data-i18n="retryMigrate" hidden>Retry migration</button>
      <button type="button" class="btn primary" id="rotateDone" data-close data-i18n="rotateDone" hidden>I have saved it</button>
    </div>
  </form>
</dialog>

<dialog id="deleteDialog">
  <form method="dialog">
    <div class="modal-head">
      <div>
        <h3 id="deleteTitle">Delete keys</h3>
        <p class="hint" data-i18n="deleteHint">Removes the usage records, limits, note, blocked models and mappings. This cannot be undone.</p>
      </div>
      <button type="button" class="modal-close" data-close data-i18n-title="close" title="Close"><svg class="icon"><use href="#i-x"/></svg></button>
    </div>
    <div class="modal-body">
      <ul class="targets" id="deleteTargets"></ul>
      <label class="toggleRow" style="margin-bottom:0">
        <span>
          <span style="font-weight:600" data-i18n="deleteFromHost">Also remove from CPA api-keys</span>
          <p class="hint" data-i18n="deleteFromHostHint">The key stops working immediately. Without this, only the record here is cleared, and the key is tracked again on its next request.</p>
        </span>
        <span class="switch"><input type="checkbox" id="deleteFromHost" checked><span class="track"></span></span>
      </label>
      <ul class="notice" id="deleteReport" style="margin-top:14px" hidden></ul>
    </div>
    <div class="modal-foot">
      <p id="deleteError" class="err hint"></p>
      <button type="button" class="btn" id="deleteCancel" data-close data-i18n="cancel">Cancel</button>
      <button type="button" class="btn danger" id="confirmDelete" data-i18n="deleteBtn">Delete</button>
      <button type="button" class="btn primary" id="deleteDone" data-close data-i18n="close" hidden>Close</button>
    </div>
  </form>
</dialog>

<dialog id="editor" class="wide">
  <form method="dialog">
    <div class="modal-head">
      <div>
        <h3 id="editTitle">Edit key</h3>
        <p class="hint mono" id="editSub"></p>
      </div>
      <button type="button" class="modal-close" data-close data-i18n-title="close" title="Close"><svg class="icon"><use href="#i-x"/></svg></button>
    </div>
    <div class="modal-body">
      <label class="field">
        <span class="label" data-i18n="note">Note</span>
        <input id="keyNote" type="text" data-i18n-ph="phNote" autocomplete="off" spellcheck="false" autofocus>
      </label>
      <label class="toggleRow">
        <span>
          <span style="font-weight:600" data-i18n="pauseKey">Pause this key</span>
          <p class="hint" data-i18n="disableKey">Requests are rejected with 403 until the key is enabled again.</p>
        </span>
        <span class="switch"><input type="checkbox" id="disabledFlag"><span class="track"></span></span>
      </label>

      <div class="section">
        <p class="section-title" data-i18n="limitsSection">Limits</p>
        <p class="hint" style="margin:-4px 0 10px" data-i18n="editHint">Blank or 0 inherits the configured limit. Use -1 for unlimited.</p>
        <div class="limitGrid">
          <span class="period" data-i18n="colDaily">Daily</span>
          <select id="d_dim">
            <option value="" data-i18n="limInherit">Inherit configuration</option>
            <option value="cost_usd" data-i18n="limCost">Budget (USD)</option>
            <option value="tokens" data-i18n="limTokens">Tokens</option>
            <option value="requests" data-i18n="limRequests">Requests</option>
          </select>
          <input type="number" step="any" id="d_value">
          <span class="period" data-i18n="colMonthly">Monthly</span>
          <select id="m_dim">
            <option value="" data-i18n="limInherit">Inherit configuration</option>
            <option value="cost_usd" data-i18n="limCost">Budget (USD)</option>
            <option value="tokens" data-i18n="limTokens">Tokens</option>
            <option value="requests" data-i18n="limRequests">Requests</option>
          </select>
          <input type="number" step="any" id="m_value">
          <span class="period" data-i18n="colTotal">Total</span>
          <select id="t_dim">
            <option value="" data-i18n="limInherit">Inherit configuration</option>
            <option value="cost_usd" data-i18n="limCost">Budget (USD)</option>
            <option value="tokens" data-i18n="limTokens">Tokens</option>
            <option value="requests" data-i18n="limRequests">Requests</option>
          </select>
          <input type="number" step="any" id="t_value">
        </div>
      </div>

      <div class="section">
        <p class="section-title" data-i18n="rateSection">Rate limits</p>
        <p class="hint" style="margin:-4px 0 10px" data-i18n="rateHint">Measured over the last minute. Blank or 0 inherits the configured value. Use -1 for unlimited.</p>
        <div class="rateGrid">
          <label><span data-i18n="rateRPM">Requests per minute</span><input type="number" step="1" id="r_rpm"></label>
          <label><span data-i18n="rateTPM">Tokens per minute</span><input type="number" step="1" id="r_tpm"></label>
          <label><span data-i18n="rateConc">Concurrent requests</span><input type="number" step="1" id="r_conc"></label>
          <label><span data-i18n="rateQueue">Queue wait (seconds)</span><input type="number" step="1" id="r_queue"></label>
        </div>
        <p class="hint" style="margin:8px 0 0" data-i18n="rateQueueHint">When the key or every account it may use is at its concurrency limit, a request waits this long for a free slot before it is rejected.</p>
      </div>

      <div class="section">
        <p class="section-title" data-i18n="keyAccounts">Accounts for this key</p>
        <p class="hint" style="margin:-4px 0 10px" data-i18n="keyAccountsHint">Account IDs or glob patterns. Requests of this key are scheduled on these accounts first. Leave it empty to use every account.</p>
        <ul id="keyAccountList" class="taglist"></ul>
        <div class="inline">
          <input id="keyAccountInput" class="mono" list="accountSuggestions" data-i18n-ph="phAccount"
                 spellcheck="false" autocomplete="off">
          <button type="button" id="keyAccountAdd" class="btn" data-i18n="addBtn">Add</button>
        </div>
        <label class="toggleRow" style="margin:12px 0 0">
          <span>
            <span style="font-weight:600" data-i18n="strictAccounts">Only use these accounts</span>
            <p class="hint" data-i18n="strictAccountsHint">When none of them can serve a request, it fails instead of falling back to the shared accounts.</p>
          </span>
          <span class="switch"><input type="checkbox" id="strictAccounts"><span class="track"></span></span>
        </label>
      </div>

      <div class="section">
        <p class="section-title" data-i18n="schedSection">Schedule</p>
        <label class="toggleRow">
          <span>
            <span style="font-weight:600" data-i18n="schedOwn">Use a schedule of its own</span>
            <p class="hint" id="schedSource"></p>
          </span>
          <span class="switch"><input type="checkbox" id="schedOwn"><span class="track"></span></span>
        </label>
        <div class="sched-body" id="schedBody">
          <p class="hint" style="margin:0 0 10px" data-i18n="schedHint">The first window covering a moment applies, so list narrow windows such as a lunch break before wide ones. An end at or before the start runs past midnight; equal times cover the whole day. Window quotas restart with every occurrence.</p>
          <label class="field">
            <span class="label" data-i18n="schedOutside">Outside every window</span>
            <select id="schedOutside">
              <option value="allow" data-i18n="schedAllow">Allow, with the normal limits</option>
              <option value="block" data-i18n="schedBlockAll">Refuse the key</option>
            </select>
          </label>
          <div class="win-list" id="winList"></div>
          <button type="button" class="btn sm" id="addWindow" data-i18n="addWindow">Add window</button>
        </div>
      </div>

      <div class="section">
        <p class="section-title" data-i18n="keyBlocked">Blocked models for this key</p>
        <ul id="keyBlockedList" class="taglist"></ul>
        <div class="inline">
          <input id="keyBlockedInput" class="mono" list="modelSuggestions" data-i18n-ph="phBlockedOne"
                 spellcheck="false" autocomplete="off">
          <button type="button" id="keyBlockedAdd" class="btn" data-i18n="addBtn">Add</button>
        </div>
        <datalist id="modelSuggestions"></datalist>
      </div>

      <div class="section">
        <p class="section-title" data-i18n="keyMappings">Model mappings for this key</p>
        <ul id="keyMappingList" class="taglist"></ul>
        <div class="mapGrid">
          <div class="mapFrom">
            <select id="keyMapFromSelect" class="mono"></select>
            <input id="keyMapFrom" class="mono" data-i18n-ph="phMapFromCustom" spellcheck="false" autocomplete="off" hidden>
          </div>
          <input id="keyMapTo" class="mono" list="modelSuggestions" data-i18n-ph="phMapTo" spellcheck="false" autocomplete="off">
          <input id="keyMapProvider" class="mono" list="providerSuggestions" data-i18n-ph="phMapProvider" spellcheck="false" autocomplete="off">
          <button type="button" id="keyMapAdd" class="btn" data-i18n="addBtn">Add</button>
        </div>
      </div>

      <div class="section">
        <p class="section-title" data-i18n="keyFallbacks">Fallback models for this key</p>
        <label class="toggleRow">
          <span>
            <span style="font-weight:600" data-i18n="fallbackOwn">Use a fallback list of its own</span>
            <p class="hint" id="fallbackSource"></p>
          </span>
          <span class="switch"><input type="checkbox" id="fallbackOwn"><span class="track"></span></span>
        </label>
        <textarea id="keyFallbacks" class="mono" rows="3" spellcheck="false" data-i18n-ph="phFallbacks"></textarea>
      </div>
    </div>
    <div class="modal-foot">
      <p id="editError" class="err hint"></p>
      <button type="button" id="clearOverrides" class="btn danger" data-i18n="clearOverrides">Clear overrides</button>
      <span class="spacer"></span>
      <button type="button" class="btn" data-close data-i18n="cancel">Cancel</button>
      <button type="button" class="btn primary" id="saveEdit" data-i18n="save">Save</button>
    </div>
  </form>
</dialog>

<div class="viz-tip" id="vizTip" hidden></div>
`
