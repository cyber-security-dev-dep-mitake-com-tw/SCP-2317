//! SCP-2317 native Outlook COM Add-in.
//!
//! Unlike `outlook-addin/` (an Office.js web add-in that must be sideloaded
//! or centrally deployed through Microsoft 365), this crate builds a classic
//! in-process COM server that the Windows installer registers directly under
//! `HKCU\Software\Microsoft\Office\Outlook\Addins`. Classic Outlook loads it
//! on next startup with no manifest sideload step, and it can draw a fully
//! custom Ribbon tab (`IRibbonExtensibility`) with no platform restriction.
//!
//! Module layout:
//! - [`scan_client`] and [`ribbon_xml`] are portable (no `windows` crate, no
//!   `unsafe`) and are covered by `cargo test` on any host OS.
//! - [`win`] contains the Windows-only COM plumbing (`IDTExtensibility2`,
//!   `IRibbonExtensibility`, `IDispatch`, class factory, DLL entry points,
//!   self-registration) and only compiles under `cfg(windows)`.

pub mod ribbon_xml;
pub mod scan_client;

#[cfg(windows)]
pub mod win;
