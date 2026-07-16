# Compile the tray app into a self-contained, silent Windows executable using
# PackageCompiler.jl. The result lands in dist/tray-julia/ and bundles the Julia
# runtime, so end users do not need Julia installed. The produced .exe is then
# Authenticode-signed by scripts/gitlab/package-windows.ps1.
#
# Usage (on a machine with Julia + a Qt/QML.jl toolchain):
#   julia --project build.jl
#
# Notes:
# - create_app compiles the project referenced by `precompile_execution_file`.
# - On Windows, the entry app is built as a GUI (no console) so it runs silently;
#   only the system-tray icon is visible until clicked.
import Pkg
Pkg.activate(@__DIR__)
Pkg.instantiate()

using PackageCompiler

const PROJECT = @__DIR__
const OUTDIR = normpath(joinpath(@__DIR__, "..", "..", "dist", "tray-julia"))

create_app(
    PROJECT,
    OUTDIR;
    executables = ["scp2317-tray" => "julia_main"],
    force = true,
    include_lazy_artifacts = true,
    incremental = false,
)

@info "Built tray app" OUTDIR
