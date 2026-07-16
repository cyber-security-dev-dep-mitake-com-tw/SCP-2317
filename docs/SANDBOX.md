# Dynamic sandbox contract

The included API never executes attachments. Production dynamic analysis must run in disposable, network-denied VMs, not in the API container or endpoint agent.

Current CI uses Robot Framework smoke tests to verify the API boundary and packaging metadata only; those tests do not replace the sandbox worker described here.

Minimum worker controls:

- one sample per ephemeral VM; destroy the VM after every job;
- deny outbound network by default and use a controlled fake-internet service;
- immutable golden image, secure boot where supported, and no host mounts;
- strict CPU, memory, file-size and wall-clock limits;
- signed job envelopes, SHA-256 addressing, audit logs and short retention;
- return normalized findings only; never return executable artifacts to clients.

Suggested interface: `Submit(hash, objectRef, policy) -> jobID`, `Status(jobID)`, and `Report(jobID)`. Put a queue between the API and workers and require mutual TLS.
