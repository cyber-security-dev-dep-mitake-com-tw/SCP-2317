# Threat model (starter)

Protected assets: email content, credentials, scan results, tenant policy and analyst identities. Trust boundaries exist at the Outlook add-in, native endpoint provider, API, object store, queue and disposable analysis VM.

Primary threats include malicious MIME parsers, decompression bombs, SSRF through URLs, attachment path traversal, sandbox escape, forged endpoint identity, replayed jobs, cross-tenant data access and sensitive-content retention.

Required production controls include mTLS device identity, tenant-scoped authorization, MIME parsing in a low-privilege process, archive recursion/ratio limits, URL canonicalization, object encryption, immutable audit events, per-tenant keys, retention deletion, signed updates and a kill switch.

Current implementation only covers the API heuristic scanner, add-in and native scaffolds, and CI/package automation. The controls above remain required before any real tenant deployment.
