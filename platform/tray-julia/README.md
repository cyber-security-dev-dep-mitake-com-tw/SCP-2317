# SCP-2317 Tray Companion (Julia / QML)

A silent, tray-only companion UI for the SCP-2317 endpoint. The Go companion
service runs headless in the background (loopback-only, no console window); this
Julia app shows **only a system-tray icon** until it is clicked, then pops out a
dashboard that reads the local API.

## What it shows

- service health (`/healthz`)
- scan verdict counts — malicious / suspicious / clean (`/v1/stats`)
- recent activity feed (`/v1/events`)
- a time-of-click URL check helper (`/v1/urlcheck`)

It never scans mail itself; it is a read-only view onto the service.

## Layout

| Path | Role |
|---|---|
| `src/SCP2317Tray.jl` | Module: HTTP client + `@qmlfunction` data fetchers + `run` / `julia_main` |
| `qml/main.qml` | Qt Quick UI: hidden `ApplicationWindow` + `SystemTrayIcon`; closing hides to tray |
| `qml/tray.png` | Tray icon (SCP-2317 logo) |
| `bin/run.jl` | Dev entry point |
| `build.jl` | PackageCompiler → self-contained signed `.exe` in `dist/tray-julia/` |

## Run (development)

Requires Julia 1.9+ and a working `QML.jl` (Qt) toolchain.

```bash
cd platform/tray-julia
julia --project -e 'import Pkg; Pkg.instantiate()'
julia --project bin/run.jl
```

Point it at a non-default API with `SCP2317_API_BASE=http://127.0.0.1:8099`.

## Build the silent Windows executable

```bash
julia --project build.jl
```

This produces `dist/tray-julia/bin/scp2317-tray.exe` (Julia runtime bundled, no
console). `scripts/gitlab/package-windows.ps1` then Authenticode-signs it and the
installer includes it with an autostart entry.

## Verification status

The Go API it talks to is fully tested (`go test ./internal/api`). This GUI is
**verified on Windows with Qt/Julia present** — it cannot be built or displayed
on a headless CI box. On non-Windows/non-Qt hosts, treat it as a code-complete
scaffold: syntax and API contract are fixed, but rendering and tray behavior
must be checked on a Windows runner.
