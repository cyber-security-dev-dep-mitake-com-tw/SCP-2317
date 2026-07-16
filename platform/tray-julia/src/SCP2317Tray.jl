"""
SCP2317Tray — a silent system-tray companion UI for the SCP-2317 endpoint.

Design: the Go companion service runs headless in the background (no console,
loopback-only). This process shows *only* a tray icon; clicking it pops out the
full dashboard window, which polls the local API over `http://127.0.0.1:8080`.
Nothing here scans mail itself — it is a read-only view onto the service.

UI: Qt Quick via QML.jl (`qml/main.qml`). Julia exposes small data-fetch
functions to QML through `@qmlfunction`; QML polls them on a timer.
"""
module SCP2317Tray

using QML
using HTTP
using JSON3

const API_BASE = get(ENV, "SCP2317_API_BASE", "http://127.0.0.1:8080")

function _get(path::AbstractString)::String
    try
        r = HTTP.get(API_BASE * path; readtimeout = 3, retry = false)
        return r.status == 200 ? String(r.body) : ""
    catch
        return ""
    end
end

"""Return `/healthz` as "ok" or "down" (consumed by QML as a status string)."""
fetch_health()::String = _get("/healthz") == "ok" ? "ok" : "down"

"""Return the `/v1/stats` JSON (or "{}" when the service is unreachable)."""
function fetch_stats()::String
    s = _get("/v1/stats")
    return isempty(s) ? "{}" : s
end

"""Return the `/v1/events` JSON array (or "[]" when unreachable)."""
function fetch_events()::String
    s = _get("/v1/events?n=50")
    return isempty(s) ? "[]" : s
end

"""Time-of-click check: POST a URL to `/v1/urlcheck`, return the verdict JSON."""
function check_url(url::AbstractString)::String
    try
        body = JSON3.write(Dict("url" => String(url)))
        r = HTTP.post(API_BASE * "/v1/urlcheck", ["Content-Type" => "application/json"], body;
                      readtimeout = 4, retry = false)
        return r.status == 200 ? String(r.body) : "{}"
    catch
        return "{}"
    end
end

"""Launch the tray app. Blocks in the Qt event loop until the user quits."""
function run()
    @qmlfunction fetch_health fetch_stats fetch_events check_url
    qml_file = joinpath(dirname(@__DIR__), "qml", "main.qml")
    loadqml(qml_file; apiBase = API_BASE)
    exec()
    return 0
end

"""PackageCompiler entry point (see build.jl). Returns a process exit code."""
function julia_main()::Cint
    try
        run()
        return 0
    catch err
        @error "tray app crashed" exception = (err, catch_backtrace())
        return 1
    end
end

end # module
