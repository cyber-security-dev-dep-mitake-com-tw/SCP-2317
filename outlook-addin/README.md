# Outlook add-in (MVP)

ItemRead add-in with:

- a dedicated **SCP-2317** ribbon tab (like a security-vendor add-in: Quick Scan, Report Suspicious, and an "SCP-2317 Endpoint" menu with status/about), and
- a taskpane that posts message metadata to the local SCP-2317 API and shows the full verdict.

## Build

```bash
cd outlook-addin
npm install
npm run build   # emits dist/taskpane.js, dist/commands.js, dist/scpApi.js
```

## Sideload (developer)

1. Start the API: `go run ./cmd/api` (listens on `http://127.0.0.1:8080`).
2. Serve this folder over **HTTPS** on `https://localhost:3000` (Office requires HTTPS for the taskpane/commands origin). A local trusted cert (for example via `npx office-addin-dev-certs`) is enough for sideload.
3. Sideload `manifest.xml` in classic Outlook desktop (Windows) using the standard add-in sideload flow (**Insert > Add-ins > My Add-ins > Add a custom add-in**).
4. Open a message in read mode. A new **SCP-2317** ribbon tab appears with:
   - **Scan & Report** – opens the taskpane with the full JSON verdict.
   - **Quick Scan** – runs the same scan and shows the verdict inline as an Outlook notification bar, no taskpane needed.
   - **Report Suspicious** – scans the message and records a job on the sandbox stub (`POST /v1/sandbox/submit`); the job is always `not_executed`.
   - **SCP-2317 Endpoint** menu – **Check API Status** (pings `/healthz`) and **About SCP-2317**.

## Ribbon tab client support (honest)

The dedicated **SCP-2317** tab uses the manifest `CustomTab` element. Per Microsoft's documentation, `CustomTab` in a Mail add-in only renders in **classic Outlook desktop on Windows**. Outlook on the web, new Outlook on Windows, and Outlook on Mac ignore `CustomTab`; on those clients the add-in still installs and works through the ItemRead taskpane form, just without the dedicated tab shown above. See `manifest.xml` for the reference link.

## Limits (honest)

- Read mode only; no `OnMessageSend` / compose interception yet.
- Attachments are metadata-only (name + content type); content is not downloaded.
- Default API base is `http://127.0.0.1:8080` (override with `globalThis.SCP2317_API_BASE`).
- "Report Suspicious" only reaches the in-memory sandbox stub (`internal/sandbox`); it never executes anything and is not persisted across API restarts.
- Not a production-signed add-in package.
