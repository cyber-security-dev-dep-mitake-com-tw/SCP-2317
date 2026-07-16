# SCP-2317 Email Security — Implementation Plan

> Scope of this plan: turn the running Outlook add-in + local Go heuristic API
> from a URL/extension checker into a **defense-in-depth email risk scanner**
> that catches the attack classes in [`docs/ref.txt`](docs/ref.txt) which are
> *actually achievable* on this architecture (endpoint add-in + local,
> non-executing, offline heuristic API). No cloud, no MX change, no attachment
> execution.

## 0. Motivating gap (observed)

The live add-in scanned a real **quishing** email (QR-code phishing PDF, spoofed
internal sender "資安研發部", body lure "這裡下載圖片") and returned
`CLEAN (score 0)`. The current pipeline only flags cleartext-HTTP / punycode
URLs and high-risk attachment *extensions*. It is blind to:

- sender / display-name spoofing and look-alike domains,
- SPF/DKIM/DMARC authentication results,
- social-engineering / BEC language (urgency, authority, payment change),
- QR-code (quishing) lures,
- outbound data loss (PII / secrets) on compose.

## 1. Which `ref.txt` techniques map onto this architecture

| ref.txt technique | Feasible here? | How |
|---|---|---|
| SPF / DKIM / DMARC | **Surface, not enforce** | Parse `Authentication-Results` header the client extracts; flag `fail`/`none`. Enforcement belongs to the mail server, not an endpoint add-in. |
| Look-alike / homograph / typosquat domains | **Yes** | Pure-Go analysis of sender/reply-to/URL hosts vs a trusted-domain list; Unicode confusable + punycode + edit-distance checks. |
| Display-name / reply-to spoofing (BEC) | **Yes** | Compare display name, From domain, Reply-To domain. |
| AI/behavioral language analysis (BEC, urgency, authority) | **Heuristic + optional LLM** | Deterministic multilingual (EN + 繁中) pattern scoring always runs offline. An **optional, disabled-by-default** LLM analyzer (`internal/llm`) can escalate ambiguous mail via OpenAI-, Grok(xAI)-, or Mistral-format `/chat/completions` APIs, enabled only by explicit config/env. Default build stays fully offline. |
| Quishing (QR-code phishing) | **Yes — real server-side decode** | API decodes inline PNG/JPEG/GIF images (stdlib `image/*`) and runs a bundled **pure-Go QR decoder** (`internal/qr`, Reed-Solomon, no external deps, no network) to extract the hidden URL, then runs full URL heuristics on it. Falls back to client-supplied `QRURLs` + lure-language combos. |
| DLP (PII / secrets outbound) | **Yes** | Compose-time scan: Luhn credit-card, TW national ID, generic secret patterns; `direction=outbound`. |
| Anti-spam / AV signatures, sandboxing, CDR | **No (already stubbed)** | Sandbox stays `not_executed`; real AV/YARA/VM deferred per README. |
| ICES API, post-delivery clawback, RBI, PQC, WORM, global TI, take-down | **No** | Require server/tenant infra out of this repo's boundary. Documented as deferred. |

## 1b. Decisions from design review ("grill me")

- **QR/quishing → real decoder.** Bundle a pure-Go, zero-dependency, offline QR
  decoder (`internal/qr`) plus stdlib image decoding server-side. Round-trip
  tested with an in-repo encoder so there are no external fixtures or network.
- **BEC → deterministic *and* optional LLM.** Ship the offline heuristic as the
  default; add a pluggable `internal/llm` analyzer with OpenAI / Grok / Mistral
  request formats, **off unless configured**, so the default remains offline and
  testable. Networked LLM calls never run in CI.
- **DLP → full send-blocking.** Implement the DLP engine in Go and wire the
  Office.js `OnMessageSend` handler to actually block sends on a hit.

## 2. Design

Add **five new pure-Go detection engines** to `internal/scanner`, all
deterministic, offline, stdlib-only, and unit-tested. Extend the request model
with the extra signals the Outlook clients can already read from a message.

### Model additions (`internal/model`)
- `EmailScanRequest.SenderDisplay string` — From display name.
- `EmailScanRequest.ReplyTo string` — Reply-To address.
- `EmailScanRequest.AuthResults string` — raw `Authentication-Results` header.
- `EmailScanRequest.Direction string` — `inbound` (default) | `outbound`.
- `EmailScanRequest.QRURLs []string` — URLs the client decoded from QR images.
- `Finding.Category string` — groups findings (`authentication`, `spoofing`,
  `social-engineering`, `quishing`, `dlp`, `url`, `attachment`).

### Engines
1. **authentication** — parse `AuthResults`; `dmarc=fail`/`spf=fail`/`dkim=fail`
   → high; missing auth on external mail → low.
2. **spoofing** — homograph (non-ASCII confusables in host), punycode host,
   From vs Reply-To domain mismatch, display-name-claims-brand vs off-domain,
   look-alike of a trusted domain (Levenshtein ≤ 2).
3. **social-engineering** — weighted multilingual keyword/intent scoring;
   urgency + payment-change combination escalates (classic BEC).
4. **quishing** — treat `QRURLs` as untrusted URLs through the URL engine;
   flag image attachment + QR-lure body language even with no decoded URL.
5. **dlp** (outbound only) — Luhn-valid card numbers, TW ID, private-key /
   password patterns → block-advisory finding.

Existing URL + attachment + size engines are refactored to emit `Category` and
reused by the quishing engine (no duplicate URL logic).

### Policy (`rules/attachment_policy.json`)
Add `trusted_domains`, and per-category score weights, all with safe defaults so
existing behavior/tests keep working.

### Clients
- `outlook-addin` (Office.js) and `platform/windows-addin` (Rust) send the new
  fields. Office.js reads internet headers via
  `getAllInternetHeadersAsync` for `AuthResults`; both extract QR payloads where
  the host allows (documented limitation otherwise).
- **Full send-blocking DLP**: Office.js `OnMessageSend` handler wired end-to-end
  via manifest `LaunchEvent` + `<Runtimes>`; on an outbound DLP hit the handler
  calls `event.completed({ allowEvent: false })` to block the send and shows the
  user why.

## 3. Testing & verification
- Go unit tests per engine (clean/positive/negative) in
  `internal/scanner/*_test.go`; keep existing tests green.
- Update `samples/` with a quishing + a BEC fixture; extend Robot smoke.
- `go test ./...`, `go vet ./...`, add-in `tsc`, Rust `cargo test` (portable
  modules) all pass.
- Manual `curl` reproduction of the observed quishing mail → now `suspicious`/
  `malicious` instead of `clean`.

## 4. Out of scope (explicitly deferred — see README "Production gaps")
Real AV/YARA/ClamAV, executing sandbox/CDR, ICES tenant API, clawback, RBI,
PQC, WORM/eDiscovery, global threat-intel, automated take-down, MFA/IAM.

## 5. Rollout order
1. Model + policy fields (back-compatible).
2. Refactor URL/attachment engines to categories.
3. Add authentication, spoofing, social-engineering, quishing, dlp engines + tests.
4. Wire Office.js + Rust clients to send new signals.
5. Samples + Robot + docs (README, SANDBOX, this plan).
