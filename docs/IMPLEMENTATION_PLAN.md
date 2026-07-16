# Implementation Plan

## Purpose

SCP-2317 is a defensive mail-protection starter monorepo for Windows and macOS. The current codebase provides:

- a Go scan API and heuristic scanner with request validation;
- an Outlook add-in ItemRead taskpane that can typecheck and call the API;
- buildable native scaffolds for Windows MAPI and macOS Endpoint Security;
- a sandbox **stub** that never executes samples;
- GitLab and GitHub Actions CI for unit/Robot/add-in checks;
- packaging scripts and unsigned installer sources.

This plan tracks progress toward a verified, packaged, releaseable product.

## Current State

### Implemented (verified in-repo)

- `cmd/api` Go API entrypoint
- `/healthz`, `/v1/email/scan`, sandbox stub routes
- heuristic verdict pipeline (URL, attachment policy file, size, body URL extraction)
- OpenAPI at `api/openapi.yaml` and fixtures under `samples/`
- Outlook add-in scaffold with taskpane scan helper (`tsc` builds)
- Windows Rust MAPI scaffold (no MAPI calls)
- macOS Swift Endpoint Security scaffold (path fixed; buildable when Swift is present)
- threat model and sandbox guidance
- GitLab CI/CD and GitHub Actions workflows
- Robot Framework API smoke tests under `tests/robot/`
- Go unit tests for API, scanner, and sandbox stub
- Windows and macOS packaging scripts under `scripts/gitlab/`
- Windows installer source files for `.exe` and `.msi` (unsigned)
- macOS packaging script that emits `.pkg` and `.dmg` (unsigned; needs runner tools)
- release/status reports under `docs/reports/`

### Missing or incomplete (deferred)

- authenticated tenant/device enrollment and mTLS
- real policy distribution and persistence
- MIME and archive parsing in a restricted process
- object store / queue / **real** disposable VM sandbox workers
- attachment download and server-side content analysis (ClamAV/YARA)
- signed update and rollback strategy
- automated UI/end-to-end Outlook host tests
- runner validation for packaging jobs on real Windows and macOS infrastructure
- installer signing, notarization, and code-signing credentials
- Outlook compose / `OnMessageSend` event handlers
- real MAPI notification subscription and macOS System Extension

## Target Outcomes

1. The API can scan representative email payloads with deterministic, testable verdicts.
2. The Outlook add-in can submit content to the API and display results.
3. Platform-specific native scaffolds remain buildable and clearly separated from unsupported production behaviors.
4. Robot Framework coverage validates the user-facing API flows and packaging assumptions.
5. GitLab CI/CD builds, tests, and packages artifacts on the correct platform runners.
6. Documentation accurately describes the architecture, operational boundaries, and release status.

## Implementation Phases

### Phase 0–1: Baseline integrity and hardening — done in current patch

- CI files present and tracked; packaging scripts no longer gitignored.
- Scanner validation, Finding JSON tags, policy file, samples, OpenAPI.
- macOS Package.swift path and Windows installer script name alignment.

### Phase 2: Product workflow — partial

- Outlook ItemRead path works for local sideload with HTTPS host.
- Remaining: compose/send events, attachment content download.

### Phase 3: Automated verification — partial

- Robot API smoke uses `samples/` fixtures.
- Remaining: Outlook UI automation when an Office host is available.

### Phase 4–5: CI/CD and packaging — scaffold present

- Package jobs are manual and runner-tagged (`allow_failure`).
- Remaining: signed artifacts on real Windows/macOS runners.

### Phase 6: Reporting — ongoing

- Keep `docs/reports/` honest about verified vs scaffold-only work.

## Deferred production modules

Do not treat the following as implemented:

1. mTLS device enrollment, tenant auth, policy distribution, persistence
2. Isolated MIME/archive parsers and ClamAV/YARA adapters
3. Queue + encrypted object store + network-denied disposable VMs
4. Code signing, notarization, updater, kill switch, telemetry, SIEM export
5. Full MAPI COM provider and Apple System Extension with ES entitlement

## Immediate Next Steps

1. Confirm packaging runner tags and code-signing credentials.
2. Add Outlook compose/send event handlers where the API set allows.
3. Introduce a restricted-process MIME parser behind a clear process boundary.
4. Swap `MemoryWorker` for a queued VM worker that still defaults to deny-network.
