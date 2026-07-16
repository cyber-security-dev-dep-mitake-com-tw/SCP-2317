# Entry point for `julia --project bin/run.jl`.
# Activates the project, loads the module, and starts the tray event loop.
import Pkg
Pkg.activate(dirname(@__DIR__))

push!(LOAD_PATH, joinpath(dirname(@__DIR__), "src"))
include(joinpath(dirname(@__DIR__), "src", "SCP2317Tray.jl"))

using .SCP2317Tray
SCP2317Tray.run()
