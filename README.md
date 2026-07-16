# SCP-2317 Agent Starter

SCP-2317 is a defensive starter monorepo for email protection on Windows and macOS, plus a safe scanning backend. It is deliberately an MVP: the scanner performs **non-executing, offline heuristic analysis** across several defense-in-depth engines and **never executes untrusted files**.

## Detection engines

The Go scanner (`internal/scanner`) runs these bounded, offline engines and merges their findings into a single verdict (`clean` / `suspicious` / `malicious`). Each finding carries a `category`.

| Engine | Category | What it catches |
|---|---|---|
| URL heuristics | `url` | Cleartext HTTP and punycode links |
| Attachment policy | `attachment` | High-risk extensions, oversize inline files |
| Authentication | `authentication` | SPF/DKIM/**DMARC** failures parsed from the `Authentication-Results` header the client supplies (the endpoint never does DNS) |
| Spoofing | `spoofing` | Homograph/punycode sender domains, From vs Reply-To mismatch, display-name brand impersonation, look-alike/typosquat of a trusted domain |
| Social-engineering | `social-engineering` | BEC/phishing intent — urgency, authority, payment-change, credential-harvest language in English **and** Traditional Chinese |
| Quishing | `quishing` | QR-code phishing: inline PNG/JPEG/GIF images are decoded server-side by a bundled pure-Go QR reader (`internal/qr`, Reed-Solomon, zero deps, no network) and the hidden URL is scored |
| DLP (outbound) | `dlp` | Data loss on send: Luhn-valid card numbers, Taiwan national IDs, private keys, AWS keys, plaintext passwords |

Optional, **disabled-by-default** LLM augmentation (`internal/llm`) can escalate ambiguous mail via OpenAI-, Grok (xAI)-, or Mistral-format `/chat/completions` APIs. It is only active when `SCP2317_LLM_API_KEY` is set, so the default build stays fully offline. See [`implementation-plan.md`](implementation-plan.md).

The Office.js add-in sends the identity signals it can read (display name, Reply-To, `Authentication-Results`) and blocks outbound sends that trip DLP via an `OnMessageSend` handler. Everything remains metadata/heuristic; **no attachment is ever executed**.

## Components

| Path | Role | Status |
|---|---|---|
| `outlook-addin/` | Cross-platform Outlook JS client | ItemRead taskpane MVP (`tsc` buildable) |
| `platform/windows-addin/` | Native classic Outlook COM add-in | Installer-registered Ribbon add-in; no manifest sideload |
| `platform/windows-mapi/` | Classic Outlook/MAPI native provider | Buildable Rust scaffold (no MAPI calls) |
| `platform/macos-es/` | macOS Endpoint Security system extension core | Buildable Swift scaffold; entitlement required |
| `cmd/api`, `internal/` | Go scan API, multi-engine heuristic scanner, sandbox stub | Runnable MVP |
| `internal/qr` | Pure-Go QR decoder/encoder (quishing) | Unit-tested, zero deps |
| `internal/llm` | Optional OpenAI/Grok/Mistral analyzer (off by default) | Unit-tested via httptest |
| `api/openapi.yaml` | HTTP contract | Synced with handlers |
| `rules/attachment_policy.json` | High-risk extension / size limits | Loaded by scanner |
| `samples/` | Clean / suspicious / malicious fixtures | Used by Robot tests |
| `docs/SANDBOX.md` | Disposable VM worker contract | Stub worker in `internal/sandbox` |

## Quick start

```bash
go test ./...
go run ./cmd/api
curl -sS http://127.0.0.1:8080/v1/email/scan \
  -H 'content-type: application/json' \
  -d '{"message_id":"demo","sender":"billing@example.test","subject":"invoice","urls":["http://xn--demo.test"],"attachments":[{"name":"invoice.exe","content_type":"application/octet-stream"}]}'
```

Or run `docker compose up --build`. The service binds only to localhost. Override the listen address with `SCP2317_ADDR` (default `127.0.0.1:8080`).

To enable the optional LLM analyzer (otherwise fully offline):

```bash
export SCP2317_LLM_API_KEY=sk-...        # required to enable
export SCP2317_LLM_FORMAT=openai         # openai | grok | mistral
# optional overrides: SCP2317_LLM_BASE_URL, SCP2317_LLM_MODEL
```

## Release packages (GitHub Actions)

Workflow: [`.github/workflows/package.yml`](.github/workflows/package.yml)

Triggers: push to `main`/`master`, version tags (`v*`), or **Run workflow** (manual dispatch).

| Runner | Artifacts |
|---|---|
| `windows-latest` | `SCP-2317-Setup.exe`, `SCP-2317.msi` |
| `macos-14` (Apple Silicon) | `SCP-2317.pkg`, `SCP-2317.dmg` |
| `macos-13` (Intel) | `SCP-2317.pkg`, `SCP-2317.dmg` |

Artifacts are unsigned developer builds. Installers embed the SCP-2317 logo from `assets/logo.*` (see `assets/ATTRIBUTION.md`).
On Windows, the installer also installs and registers the native classic Outlook COM add-in with `regsvr32`, so classic Outlook loads the embedded Ribbon UI on startup without a `manifest.xml` sideload step.

Local packaging:

```bash
# macOS
bash scripts/gitlab/package-macos.sh

# Windows (Inno Setup 6 + WiX 4 required)
powershell -File scripts/gitlab/package-windows.ps1
```

## Verification and release scaffolding

| Item | Status |
|---|---|
| `go test ./...` | Verified locally / CI |
| Robot Framework API smoke (`make test-robot`) | Verified when Python + robotframework installed |
| Outlook add-in typecheck (`make build-addin`) | Verified via `tsc` |
| GitHub Actions (`.github/workflows/ci.yml`) | Present: unit, robot, add-in, scaffold checks |
| GitHub Actions package (`.github/workflows/package.yml`) | Windows `.exe`/`.msi` + macOS `.pkg`/`.dmg` on tag/main or manual dispatch |
| GitLab CI (`.gitlab-ci.yml`) | Present: lint/unit/robot; package jobs manual + runner-tagged |
| Packaging scripts (`scripts/gitlab/`) | Present; require Windows/macOS toolchains |
| App logo (`assets/logo.png`, `.ico`, `.icns`) | SCP-2317 artwork (see `assets/ATTRIBUTION.md`) |
| Installer sources (`installer/windows/`) | Present (unsigned) |
| Release report (`docs/reports/`) | Updated for this patch |

## Recommended implementation order

1. Complete tenant authentication, mTLS device enrollment and policy distribution.
2. Finish the Outlook JS add-in for supported read/compose and send events.
3. Add MIME/archive parsing in a separate restricted process, followed by ClamAV/YARA adapters.
4. Replace the sandbox stub with the queue/object-store + disposable VM worker in `docs/SANDBOX.md`.
5. Add Windows MAPI only for classic Outlook coverage gaps; keep it read-only until code signing, deployment and rollback are proven.
6. Package macOS as a signed containing app + System Extension after Apple grants the Endpoint Security entitlement.

## Important platform realities

- Office.js is the cross-platform path. Event-based handlers are short-lived and have client/API-set constraints.
- MAPI/COM is Windows/classic-Outlook specific. The native COM add-in is actively registered by the Windows installer for classic Outlook; new Outlook for Windows does not support COM/VSTO add-ins.
- Apple Endpoint Security observes filesystem/process events, not Outlook message semantics. Combine it with Office.js or server-side mail scanning.
- Never run submitted attachments inside the API container, on the endpoint, or with host filesystem/network access.

## Production gaps (explicitly deferred)

Certificate enrollment, admin UI, persistence, message attachment download, real queue/object storage, YARA/ClamAV, a real VM sandbox, notarized installers, telemetry, updater and SIEM export remain next modules—not hidden assumptions.

## Reference projects and docs

- Microsoft `mapi-rs` / `outlook-mapi` for Rust MAPI bindings
- Microsoft Office Add-in samples and Outlook event-based activation documentation
- Apple Endpoint Security and System Extensions documentation
- Aurora Agent for ETW/Sigma architectural study
- Cheshire/LitterBox only as analyzer-chain research; do not copy its C2-oriented workflow into endpoint production code

Use only on systems and mail tenants you are authorized to protect.
