package api

import "net/http"

// dashboardHTML is a self-contained diagnostics UI served by the local API.
// It is the always-available fallback surface for the companion features
// (event log, stats, time-of-click URL check, EASM) when the optional tray
// app is not installed. No external assets; everything inlined; loopback only.
const dashboardHTML = `<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8">
<meta name="viewport" content="width=device-width, initial-scale=1">
<title>SCP-2317 Dashboard</title>
<style>
  :root { color-scheme: dark; }
  * { box-sizing: border-box; margin: 0; }
  body { background:#0f1419; color:#c9d1d9; font:14px/1.5 system-ui,'Segoe UI',sans-serif; padding:24px; max-width:960px; margin:0 auto; }
  h1 { color:#e6edf3; font-size:22px; margin-bottom:4px; }
  h2 { color:#e6edf3; font-size:15px; margin:20px 0 8px; }
  .muted { color:#8b949e; font-size:12px; }
  .pill { display:inline-block; padding:2px 10px; border-radius:12px; font-size:12px; border:1px solid; }
  .ok   { color:#3fb950; border-color:#2ea043; background:#132e1a; }
  .bad  { color:#f85149; border-color:#f85149; background:#3a1414; }
  .tiles { display:flex; gap:12px; margin:12px 0; }
  .tile { flex:1; background:#161b22; border:1px solid #30363d; border-radius:10px; padding:14px; text-align:center; }
  .tile b { display:block; font-size:26px; }
  .mal b { color:#f85149; } .sus b { color:#d29922; } .cln b { color:#3fb950; }
  table { width:100%; border-collapse:collapse; background:#161b22; border:1px solid #30363d; border-radius:10px; overflow:hidden; }
  th,td { text-align:left; padding:6px 10px; border-bottom:1px solid #21262d; font-size:12px; }
  th { color:#8b949e; font-weight:600; }
  td.s-malicious { color:#f85149; } td.s-suspicious { color:#d29922; } td.s-clean { color:#3fb950; }
  form { display:flex; gap:8px; margin:8px 0; }
  input[type=text] { flex:1; background:#0d1117; color:#c9d1d9; border:1px solid #30363d; border-radius:6px; padding:7px 10px; }
  button { background:#21262d; color:#c9d1d9; border:1px solid #30363d; border-radius:6px; padding:7px 14px; cursor:pointer; }
  button:hover { background:#30363d; }
  pre { background:#0d1117; border:1px solid #30363d; border-radius:6px; padding:10px; font-size:12px; overflow-x:auto; white-space:pre-wrap; }
</style>
</head>
<body>
  <h1>SCP-2317 <span id="health" class="pill bad">checking…</span></h1>
  <div class="muted">Local diagnostics dashboard. Loopback only; nothing leaves this machine.</div>

  <div class="tiles">
    <div class="tile mal"><b id="n-mal">0</b>Malicious</div>
    <div class="tile sus"><b id="n-sus">0</b>Suspicious</div>
    <div class="tile cln"><b id="n-cln">0</b>Clean</div>
    <div class="tile"><b id="n-tot">0</b>Total events</div>
  </div>

  <h2>Time-of-click URL check</h2>
  <form id="urlform"><input type="text" id="url" placeholder="https://example.com/link-from-an-email"><button>Check</button></form>
  <pre id="urlout" hidden></pre>

  <h2>EASM — SPF/DMARC/MX drift (domains you own)</h2>
  <form id="easmform"><input type="text" id="domain" placeholder="yourcompany.com"><button>Check</button></form>
  <pre id="easmout" hidden></pre>

  <h2>Event log <span class="muted">(auto-refreshes every 5s)</span></h2>
  <table><thead><tr><th>Time (UTC)</th><th>Kind</th><th>Status</th><th>Score</th><th>Summary</th></tr></thead>
  <tbody id="events"><tr><td colspan="5" class="muted">loading…</td></tr></tbody></table>

<script>
async function jget(p){ const r = await fetch(p); return r.json(); }
function esc(s){ const d = document.createElement('span'); d.textContent = s ?? ''; return d.innerHTML; }
async function refresh(){
  try {
    const h = await fetch('/healthz');
    const el = document.getElementById('health');
    el.textContent = h.ok ? 'service ok' : 'service error';
    el.className = 'pill ' + (h.ok ? 'ok' : 'bad');
  } catch { const el=document.getElementById('health'); el.textContent='offline'; el.className='pill bad'; }
  try {
    const s = await jget('/v1/stats');
    const by = s.by_status || {};
    document.getElementById('n-mal').textContent = by.malicious || 0;
    document.getElementById('n-sus').textContent = by.suspicious || 0;
    document.getElementById('n-cln').textContent = by.clean || 0;
    document.getElementById('n-tot').textContent = s.total || 0;
  } catch {}
  try {
    const evs = await jget('/v1/events?n=50');
    const tb = document.getElementById('events');
    if (!Array.isArray(evs) || evs.length === 0) { tb.innerHTML = '<tr><td colspan="5" class="muted">no events yet — run a Quick Scan in Outlook</td></tr>'; return; }
    tb.innerHTML = evs.map(e =>
      '<tr><td>' + esc((e.time||'').replace('T',' ').slice(0,19)) + '</td><td>' + esc(e.kind) +
      '</td><td class="s-' + esc(e.status||'') + '">' + esc(e.status||'') + '</td><td>' + (e.score ?? '') +
      '</td><td>' + esc(e.summary||'') + '</td></tr>').join('');
  } catch {}
}
document.getElementById('urlform').onsubmit = async (ev) => {
  ev.preventDefault();
  const out = document.getElementById('urlout'); out.hidden = false; out.textContent = '…';
  try {
    const r = await fetch('/v1/urlcheck', { method:'POST', headers:{'content-type':'application/json'},
      body: JSON.stringify({ url: document.getElementById('url').value }) });
    out.textContent = JSON.stringify(await r.json(), null, 2);
  } catch (e) { out.textContent = String(e); }
  refresh();
};
document.getElementById('easmform').onsubmit = async (ev) => {
  ev.preventDefault();
  const out = document.getElementById('easmout'); out.hidden = false; out.textContent = '…';
  try {
    const r = await fetch('/v1/easm?domain=' + encodeURIComponent(document.getElementById('domain').value));
    out.textContent = JSON.stringify(await r.json(), null, 2);
  } catch (e) { out.textContent = String(e); }
  refresh();
};
refresh(); setInterval(refresh, 5000);
</script>
</body>
</html>`

func (h *Handler) dashboard(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(dashboardHTML))
}
