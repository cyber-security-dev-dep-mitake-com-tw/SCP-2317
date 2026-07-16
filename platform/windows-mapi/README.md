# Windows MAPI provider scaffold

Buildable Rust crate that prints a scaffold banner. It does **not** call MAPI/COM.

```bash
cargo check --manifest-path platform/windows-mapi/Cargo.toml
# On Windows targets only, the binary is meaningful:
cargo build --manifest-path platform/windows-mapi/Cargo.toml
```

Production work (signed COM registration, MAPI notification subscription) requires code signing, deployment, and rollback before any mailbox access.
