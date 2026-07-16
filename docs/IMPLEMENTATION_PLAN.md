# Implementation Plan

## Purpose

SCP-2317 is a defensive mail-protection starter monorepo for Windows and macOS. The current codebase already provides:

- a minimal Go scan API and heuristic scanner;
- an Outlook add-in scaffold for client-side scans;
- buildable native scaffolds for Windows MAPI and macOS Endpoint Security;
- basic architecture notes for sandboxing and threat modeling.

This plan turns that starter into a verified, packaged, releaseable product with:

- stronger scanning and policy enforcement;
- automated end-to-end validation;
- GitLab and GitHub Actions-based CI/CD;
- Windows and macOS installers;
- documentation and release reporting.

## Current State

### Implemented

- `cmd/api` Go API entrypoint
- `/healthz` and `/v1/email/scan` handlers
- heuristic verdict pipeline for risky URLs and attachments
- Outlook add-in scaffold with a taskpane scan helper
- Windows Rust MAPI scaffold
- macOS Swift Endpoint Security scaffold
- initial threat model and sandbox guidance
- normalized Go API entrypoint at `cmd/api/main.go`
- GitLab CI/CD scaffold with lint, unit test, Robot Framework, package, and publish stages
- GitHub Actions mirror of the GitLab pipeline
- Robot Framework API smoke tests under `tests/robot/`
- Go unit tests for the API and scanner pipeline
- Windows and macOS packaging scripts under `scripts/gitlab/`
- Windows installer source files for `.exe` and `.msi`
- macOS packaging script that emits `.pkg` and `.dmg`
- release report artifact under `docs/reports/`

### Missing or incomplete

- authenticated tenant/device enrollment
- real policy distribution and persistence
- MIME and archive parsing in a restricted process
- object store / queue / sandbox worker contract
- attachment download and server-side content analysis
- signed update and rollback strategy
- automated UI/end-to-end tests beyond the API smoke coverage
- runner validation for the GitLab CI/CD jobs on real Windows and macOS infrastructure
- installer signing, notarization, and code-signing credentials
- package validation on real Windows and macOS runners
- release notes polish beyond the initial implementation status report

## Target Outcomes

1. The API can scan representative email payloads with deterministic, testable verdicts.
2. The Outlook add-in can submit content to the API and display results.
3. Platform-specific native scaffolds remain buildable and clearly separated from unsupported production behaviors.
4. Robot Framework coverage validates the user-facing flows and packaging assumptions.
5. GitLab CI/CD builds, tests, and packages artifacts on the correct platform runners.
6. Documentation accurately describes the architecture, operational boundaries, and release status.

## Implementation Phases

### Phase 1: Baseline hardening

- Normalize module/package names and build entrypoints.
- Expand scanner tests for URLs, attachments, and size controls.
- Add request validation and consistent API responses.
- Capture the current architecture and operational limits in docs.

### Phase 2: Product workflow

- Complete the Outlook add-in scan path for taskpane and event-based flows.
- Define the API contract for message metadata, attachments, and verdicts.
- Add repeatable fixtures for clean, suspicious, and malicious samples.

### Phase 3: Automated verification

- Add Robot Framework tests for API health, scan endpoints, and basic UI smoke coverage.
- Define a test matrix with at least:
  - API health and scan contract checks;
  - Outlook add-in smoke interactions where a browser or Office host is available;
  - packaging smoke checks that confirm installer artifacts are created and named correctly.
- Store reusable Robot keywords and suites under `tests/robot/` or a similar top-level test directory.
- Add platform-aware local test scripts.
- Wire the tests into CI so packaging only runs after verification passes.

### Phase 4: GitLab CI/CD

- Add a GitLab pipeline with separate stages for lint, unit tests, integration tests, and packaging.
- Prefer GitLab-native stages that separate verification from release artifact generation:
  - `lint`
  - `unit-test`
  - `robot-test`
  - `package`
  - `publish`
- Define Windows build jobs for installer artifacts.
- Define macOS build jobs for app packaging artifacts.
- Provide GitLab scripts for deterministic build steps, ideally in `scripts/gitlab/`.
- Publish versioned artifacts and retain logs for release auditing.

### Phase 5: Installer and release packaging

- Build Windows setup outputs as `.exe` and `.msi`.
- Build macOS installer outputs as `.pkg` and `.dmg`.
- Document signing, notarization, and code-signing prerequisites.

### Phase 6: Reporting and documentation

- Add a report markdown file under `docs/reports/`.
- Update `README.md` and security docs to match the implemented behavior.
- Record known limitations, verification evidence, and release readiness gaps.

## Open Questions

- Do you want a single cross-platform Robot Framework suite, or separate suites for API, UI, and packaging verification?
- What GitLab runner topology is available for Windows and macOS packaging?
- Are code signing certificates and Apple notarization credentials already available?
- Should the first release target be developer test builds or a production-ready signed release?
- Should I keep the current `cmd/api` layout as the long-term Go entrypoint, or do you want a different command/package naming convention?

## Immediate Next Steps

1. Confirm packaging and signing constraints.
2. Finalize the Robot Framework scope.
3. Add CI/CD and packaging scripts.
4. Implement the higher-priority missing scanner and add-in behaviors.
5. Produce the first verification report under `docs/reports/`.
