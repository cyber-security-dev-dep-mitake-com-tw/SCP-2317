# Outlook add-in (MVP)

ItemRead taskpane that posts message metadata to the local SCP-2317 API.

## Build

```bash
cd outlook-addin
npm install
npm run build   # emits dist/taskpane.js
```

## Sideload (developer)

1. Start the API: `go run ./cmd/api` (listens on `http://127.0.0.1:8080`).
2. Serve this folder over **HTTPS** on `https://localhost:3000` (Office requires HTTPS for the taskpane origin). A local trusted cert (for example via `npx office-addin-dev-certs`) is enough for sideload.
3. Sideload `manifest.xml` in Outlook (Windows/macOS) using the standard add-in sideload flow.
4. Open a message in read mode and click **Scan email**.

## Limits (honest)

- Read mode only; no `OnMessageSend` / compose interception yet.
- Attachments are metadata-only (name + content type); content is not downloaded.
- Default API base is `http://127.0.0.1:8080` (override with `globalThis.SCP2317_API_BASE`).
- Not a production-signed add-in package.
