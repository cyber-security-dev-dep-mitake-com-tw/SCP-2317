# Dynamic sandbox contract

The included API never executes attachments. Production dynamic analysis must run in disposable, network-denied VMs, not in the API container or endpoint agent.

## Current code mapping

| Contract | Implementation |
|---|---|
| `Submit(hash, objectRef, policy) -> jobID` | [`internal/sandbox`](../internal/sandbox/worker.go) `Worker.Submit` + `POST /v1/sandbox/submit` |
| `Status(jobID)` | `Worker.Status` + `GET /v1/sandbox/status/{jobID}` |
| `Report(jobID)` | `Worker.Report` + `GET /v1/sandbox/report/{jobID}` |

The default worker is `MemoryWorker`: it acknowledges jobs and always returns status `not_executed` with a policy-only finding. Optional `POST /v1/email/scan?dynamic=true` also creates such a stub job; the heuristic verdict still comes from the non-executing scanner.

Current CI uses Robot Framework smoke tests to verify the API boundary and packaging metadata only; those tests do not replace a real sandbox worker.

## Minimum production worker controls

- one sample per ephemeral VM; destroy the VM after every job;
- deny outbound network by default and use a controlled fake-internet service;
- immutable golden image, secure boot where supported, and no host mounts;
- strict CPU, memory, file-size and wall-clock limits;
- signed job envelopes, SHA-256 addressing, audit logs and short retention;
- return normalized findings only; never return executable artifacts to clients.

Put a queue between the API and workers and require mutual TLS before enabling execution.
