# Threat model (starter)

Protected assets: email content, credentials, scan results, tenant policy and analyst identities. Trust boundaries exist at the Outlook add-in, native endpoint provider, API, object store, queue and disposable analysis VM.

Primary threats include malicious MIME parsers, decompression bombs, SSRF through URLs, attachment path traversal, sandbox escape, forged endpoint identity, replayed jobs, cross-tenant data access and sensitive-content retention.

Required production controls include mTLS device identity, tenant-scoped authorization, MIME parsing in a low-privilege process, archive recursion/ratio limits, URL canonicalization, object encryption, immutable audit events, per-tenant keys, retention deletion, signed updates and a kill switch.

## Current coverage

- Heuristic API scanner with validation and policy file (no attachment execution).
- Sandbox **stub** that always returns `not_executed` (see `internal/sandbox` and `docs/SANDBOX.md`).
- Outlook add-in and native scaffolds without production identity or interception guarantees.
- CI and unsigned packaging scaffolding.

The production controls above remain required before any real tenant deployment.
