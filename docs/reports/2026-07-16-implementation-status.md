# SCP-2317 Gap Patch Status Report

Date: 2026-07-16

## Scope

Closes the honesty/buildability gap between documentation and the repository, then hardens the MVP API contract, Outlook taskpane, sandbox stub, and CI scaffolding.

## Completed In This Pass

### Integrity (Phase 0)

- fixed `.gitignore` so packaging scripts and `ApiLibrary.py` are tracked;
- added `.github/workflows/ci.yml` and `.gitlab-ci.yml`;
- aligned Windows packaging script with `mailguard-setup.iss` / `mailguard-product.wxs`;
- fixed macOS `Package.swift` path to `Sources/MailGuardES`;
- corrected README verification claims.

### API / scanner (Phase 1)

- Finding JSON tags (`engine`, `severity`, `summary`);
- structured 4xx `APIError` responses;
- request validation (`message_id`, URL/attachment caps, invalid base64);
- body_preview URL extraction;
- `rules/attachment_policy.json`, `samples/*.json`, `api/openapi.yaml`;
- expanded Go and Robot tests.

### Outlook add-in (Phase 2)

- removed conflicting `declare const Office`;
- `Office.onReady`, sender, URL extraction, configurable API base (`http://127.0.0.1:8080`);
- sideload notes in `outlook-addin/README.md`.

### Native scaffolds / Makefile (Phase 3)

- platform README boundaries;
- `make test-robot` / `make build-addin`; zip target directory name fixed.

### Sandbox stub (Phase 4)

- `internal/sandbox` MemoryWorker (`not_executed`);
- HTTP routes for submit/status/report;
- `docs/SANDBOX.md` mapped to code.

### Deferred list (Phase 5)

- documented in README, IMPLEMENTATION_PLAN, and this report (auth/MIME/real VM/signing).

## Verification

| Check | Result |
|---|---|
| `go test ./...` | Run as part of this pass |
| Robot smoke (`scripts/gitlab/run-robot.sh`) | Run when Python + robotframework available |
| Outlook `npm test` (`tsc --noEmit`) | Run as part of this pass |
| Windows/macOS installer builds | Not run locally; require platform runners |
| Real disposable VM sandbox | Not implemented (stub only) |

## Remaining Work

- tenant auth, mTLS, policy persistence;
- MIME/archive isolated parsing, ClamAV/YARA;
- queue/object store + real disposable VMs;
- signed/notarized installers and updater;
- Outlook send/compose events and native MAPI/ES production clients;
- packaging job validation on real Windows and macOS GitLab runners.
