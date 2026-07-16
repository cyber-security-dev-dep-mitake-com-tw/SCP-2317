# SCP-2317 Agent Starter

SCP-2317 is a defensive starter monorepo for email protection on Windows and macOS, plus a safe scanning backend. It is deliberately an MVP: the default scanner performs metadata and URL/attachment policy checks and **does not execute untrusted files**.

## Components

| Path | Role | Status |
|---|---|---|
| `outlook-addin/` | Cross-platform Outlook JS client | Basic scan client |
| `platform/windows-mapi/` | Classic Outlook/MAPI native provider | Buildable Rust scaffold |
| `platform/macos-es/` | macOS Endpoint Security system extension core | Buildable Swift scaffold; entitlement required |
| `cmd/api`, `internal/` | Go scan API and analyzer pipeline | Runnable MVP |
| `docs/SANDBOX.md` | Disposable VM worker contract | Design baseline |

## Quick start

```bash
go test ./...
go run ./cmd/api
curl -sS http://localhost:8080/v1/email/scan \
  -H 'content-type: application/json' \
  -d '{"message_id":"demo","sender":"billing@example.test","subject":"invoice","urls":["http://xn--demo.test"],"attachments":[{"name":"invoice.exe","content_type":"application/octet-stream"}]}'
```

Or run `docker compose up --build`. The service binds only to localhost.

## Verification and release scaffolding

- Local API coverage: `go test ./...`
- Go unit tests: API routes and scanner heuristics
- Robot Framework smoke tests: `tests/robot/`
- GitLab CI/CD: `.gitlab-ci.yml`
- GitHub Actions: `.github/workflows/ci.yml`
- Packaging scripts: `scripts/gitlab/`
- Installer sources: `installer/windows/`
- Release report: `docs/reports/`

## Recommended implementation order

1. Complete tenant authentication, mTLS device enrollment and policy distribution.
2. Finish the Outlook JS add-in for supported read/compose and send events.
3. Add MIME/archive parsing in a separate restricted process, followed by ClamAV/YARA adapters.
4. Implement the queue/object-store contract and disposable VM sandbox described in `docs/SANDBOX.md`.
5. Add Windows MAPI only for classic Outlook coverage gaps; keep it read-only until code signing, deployment and rollback are proven.
6. Package macOS as a signed containing app + System Extension after Apple grants the Endpoint Security entitlement.

## Important platform realities

- Office.js is the cross-platform path. Event-based handlers are short-lived and have client/API-set constraints.
- MAPI/COM is Windows/classic-Outlook specific and is not equivalent to a supported universal pre-view interception hook.
- Apple Endpoint Security observes filesystem/process events, not Outlook message semantics. Combine it with Office.js or server-side mail scanning.
- Never run submitted attachments inside the API container, on the endpoint, or with host filesystem/network access.

## Production gaps

This starter omits certificate enrollment, admin UI, persistence, message attachment download, queue/object storage, YARA/ClamAV, a real VM sandbox, notarized installers, telemetry, updater and SIEM export. Those are explicit next modules, not hidden assumptions.

## Reference projects and docs

- Microsoft `mapi-rs` / `outlook-mapi` for Rust MAPI bindings
- Microsoft Office Add-in samples and Outlook event-based activation documentation
- Apple Endpoint Security and System Extensions documentation
- Aurora Agent for ETW/Sigma architectural study
- Cheshire/LitterBox only as analyzer-chain research; do not copy its C2-oriented workflow into endpoint production code

Use only on systems and mail tenants you are authorized to protect.
