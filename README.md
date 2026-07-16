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
| URL context | `url` / `spoofing` | Credential-lure links on abused trusted SaaS (Google Forms, SharePoint, Notion…), and in-body look-alike/typosquat links |
| Quishing | `quishing` | QR-code phishing: inline PNG/JPEG/GIF images are decoded server-side by a bundled pure-Go QR reader (`internal/qr`, Reed-Solomon, zero deps, no network) and the hidden URL is scored |
| Route forensics | `route-forensics` | Received-chain analysis: implausibly long relay chains, bare-IP relays with no reverse DNS |
| Threat intel (IOC) | `threat-intel` | Local, offline IOC list (`rules/ioc.json`): known-bad attachment SHA-256, sender/URL domains, phishing-kit URL patterns |
| DLP (outbound) | `dlp` | Data loss on send: Luhn-valid card numbers, Taiwan national IDs, private keys, AWS keys, plaintext passwords |

The social-engineering engine covers the psychological vectors in `ref.txt` §5: authority, urgency, financial/payment-change, credential harvest, reciprocity/reward, and sympathy/gift-card lures — in English and Traditional Chinese. The attachment engine also flags double-extension masquerades (`invoice.pdf.png`).

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

## `docs/ref.txt` coverage matrix

`docs/ref.txt` is a full defense-in-depth catalogue spanning protocol, gateway,
cloud, cryptography, forensics, governance, and human-layer controls. This
product is **one node** of that architecture: an offline, non-executing endpoint
add-in plus a local heuristic API. The table below accounts for **every**
technique named in the reference and states, honestly, what this repo does and —
where it does nothing — the architectural reason it cannot.

Legend: ✅ implemented · 🟡 partial (bounded by the endpoint role) · ❌ not
implementable in this product (reason given).

### 1. Sender authentication & domain protection
| Technique | Status | Notes / why not |
|---|---|---|
| SPF / DKIM / DMARC result handling | 🟡 | `authentication` engine parses the `Authentication-Results` header the client already has. **Verification** (DNS, signature crypto) is done by the receiving mail server; an endpoint doing its own DNS would be unreliable and defeat the offline guarantee. |
| DMARC policy enforcement (none/quarantine/reject) | ❌ | Enforcement is a server-side mail-flow decision (MX/gateway), not something a read-side add-in can impose. |
| BIMI (verified brand logo) | ❌ | Requires a DMARC `p=reject` posture, a Verified Mark Certificate, and **client rendering** of the logo. The mail client controls the inbox chrome; an add-in cannot paint the sender avatar. |

### 2. Gateway / inbound protection
| Technique | Status | Notes / why not |
|---|---|---|
| Anti-spam / anti-virus signatures | 🟡 | `threat-intel` engine matches attachment SHA-256 and domains/URLs against a **local** IOC list (`rules/ioc.json`). A full AV engine (byte signatures/heuristics) needs a ClamAV/YARA process — a deferred production module, not bundled. |
| IP reputation | 🟡 | Covered as offline domain/URL IOC matching. Live IP-reputation feeds need a network service and licensing. |
| Sandboxing (zero-day detonation) | ❌ | The API **never executes** attachments by design. Real dynamic analysis must run in disposable, network-denied VMs — see `docs/SANDBOX.md`. The included worker is a `not_executed` stub. |
| CDR (content disarm & reconstruct) | ❌ | Rebuilding files (stripping macros/scripts) requires format-specific parsers/rewriters and changes the user's attachment; out of scope for a heuristic scanner and risky to do on the endpoint. |
| Time-of-Click / URL rewriting | ❌ | Rewriting links and re-checking them on click requires a mail-gateway that mutates message bodies and hosts a click-proxy. An add-in cannot rewrite delivered mail server-side. |

### 3. Data protection & confidentiality
| Technique | Status | Notes / why not |
|---|---|---|
| DLP (PII/secret detection, block on send) | ✅ | Outbound `dlp` engine + Office.js `OnMessageSend` soft-block. |
| Redaction / forced encryption on send | ❌ | Mutating or encrypting the outgoing message body/policy routing is a mail-transport/IRM action; the add-in blocks the send instead. |
| TLS (server-to-server transport) | ❌ | A property of the mail transport (MTA), invisible to and uncontrollable by an endpoint add-in. |
| S/MIME / PGP end-to-end encryption | ❌ | Key management and message crypto belong to the mail client / IRM stack; providing it would duplicate client functionality and mishandle private keys. |

### 4. Access control & internal defense
| Technique | Status | Notes / why not |
|---|---|---|
| Behavioral / AI analysis (intent, urgency, authority) | 🟡 | Deterministic multilingual `social-engineering` engine + optional LLM analyzer. |
| Writing-style / stylometry baselining | ❌ | Needs per-user historical mail to build a baseline — i.e. mailbox-history access via a tenant API (ICES), which this local add-in does not have. |
| IAM / MFA / conditional access | ❌ | An identity-provider (Entra ID/Okta) concern; cannot be implemented from an Outlook add-in. |

### 5. Human layer
| Technique | Status | Notes / why not |
|---|---|---|
| One-click "Report Phishing" | ✅ | Ribbon **Report Suspicious** → sandbox stub job. |
| Security-awareness training | ❌ | An organizational program, not software in this repo. |
| Phishing simulation | ❌ | Requires a campaign platform that sends and tracks simulated lures across the tenant. |

### Advanced dimensions (Gartner ICES / Zero Trust)
| Technique | Status | Notes / why not |
|---|---|---|
| ICES API integration (internal-to-internal, historical context, zero-deploy) | ❌ | Requires OAuth/Graph API access to the M365/Workspace tenant and back-office services. This product is an endpoint node, not a tenant-wide API service. |
| Post-delivery remediation / clawback / continuous re-evaluation | ❌ | Pulling already-delivered mail from every mailbox needs tenant-wide mailbox API + a scheduler; impossible from a single read-side add-in. |
| Look-alike / typosquat domain detection | ✅ | `spoofing` + `url-context` engines (confusable folding + Levenshtein vs trusted domains). |
| Vendor relationship profiling | ❌ | Requires organization-wide historical mail-flow to build a trust graph — a tenant-API/data-warehouse capability. |
| XDR / SIEM correlation | ❌ | Needs a SIEM to ship logs to and endpoint/firewall telemetry to correlate; deferred (no telemetry/SIEM export module). |
| SOAR automated playbooks (reset AD, kill sessions, isolate host) | ❌ | Requires privileged automation across IdP/EDR/AD — well outside an add-in's authority. |

### Emerging threat vectors (2026)
| Technique | Status | Notes / why not |
|---|---|---|
| Quishing (QR OCR / computer vision) | ✅ | Server-side decode of inline images with the bundled pure-Go QR reader; `qr_urls` fallback for client-rasterized PDFs. |
| Legitimate-SaaS abuse (Forms/SharePoint phishing) | ✅ | `url-context` engine. |
| Cross-channel / hybrid (smishing, vishing, voice cloning) | ❌ | Defending SMS/voice needs telecom + MDM signal sources this product has no access to. |
| Collaboration-platform (Teams/Slack/Zoom) cross-SaaS remediation | ❌ | Requires API integrations into those SaaS platforms; out of an Outlook add-in's scope. |
| Authentication-drift / EASM / subdomain-takeover monitoring | ❌ | Requires continuous external DNS/attack-surface scanning of all org domains — a cloud EASM service, not an endpoint. |
| GenAI NLP intent / relationship-graph / dynamic banners | 🟡 | Optional LLM analyzer covers intent; the inline verdict notification is a basic in-context banner. Relationship graphs need mailbox history (tenant API). |

### Deepest dimensions (crypto / forensics / governance / isolation)
| Technique | Status | Notes / why not |
|---|---|---|
| Header / route forensics (Received chain) | 🟡 | `route-forensics` engine flags long relay chains and bare-IP hops. Full ASN/geo/TTL analysis needs a live IP-intelligence database (licensed, network-dependent). |
| Retroactive threat hunting over the archive | ❌ | Requires an indexed historical mail archive and tenant-wide re-scan; no persistence/archive here. |
| Post-Quantum Cryptography (ML-KEM/ML-DSA, hybrid) | ❌ | PQC belongs in the TLS/S/MIME/DKIM crypto layers of the transport and CA ecosystem, not in a heuristic content scanner. |
| WORM archiving / eDiscovery / legal hold | ❌ | A compliance-archive product with immutable storage and retention/search — a separate system of record. |
| Remote Browser Isolation (RBI) | ❌ | Needs cloud browser-isolation infrastructure (pixel-streaming containers); cannot run on the endpoint. |

### Deception & active countermeasures
| Technique | Status | Notes / why not |
|---|---|---|
| Honeytokens / decoy mailboxes / decoy emails | ❌ | Requires provisioning tenant mailboxes and monitoring access org-wide — an administrative/SOC deployment, not endpoint code. |
| Automated phishing-site takedown / abuse reporting | ❌ | Requires authenticated integrations with registrars/hosts/CERTs and legal process; also an active-response function outside this defensive scanner's remit. |
| Phishing-kit reverse engineering | ❌ | A manual/analyst research activity; automating credential-drop extraction edges into offensive tooling and is intentionally excluded. |
| Global anonymized telemetry / ISAC / STIX-TAXII sharing | 🟡 | Approximated offline by the local IOC list (`rules/ioc.json`), which can be fed from a STIX/TAXII export operationally. Live global sharing needs a vendor cloud + membership. |

### Social-engineering threat dimensions (`ref.txt` final section)
| Technique | Status | Notes / why not |
|---|---|---|
| Psychological vectors (authority, urgency, reciprocity, sympathy) | ✅ | `social-engineering` lexicon, EN + 繁中. |
| Contextual spoofing (spear phishing, whaling, BEC/invoice fraud) | ✅ | `spoofing` (display-name/reply-to/look-alike) + BEC financial+authority pattern. |
| Homograph / IDN attacks | ✅ | Confusable folding + punycode detection. |
| Typosquat / look-alike domains | ✅ | Levenshtein vs trusted domains, sender and in-body links. |
| Living off trusted SaaS | ✅ | `url-context` engine. |
| AI-scaled spear phishing | 🟡 | Optional LLM analyzer. |
| Deepfake / voice-clone multichannel | ❌ | Cross-channel (audio/video) defense is outside email and this product's inputs. |

If you need any ❌ item, it belongs in one of the deferred production modules
(tenant API/ICES, real sandbox/CDR, SIEM/SOAR, archive/eDiscovery, EASM, RBI) —
see **Production gaps** below. Nothing above is a hidden assumption.

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
