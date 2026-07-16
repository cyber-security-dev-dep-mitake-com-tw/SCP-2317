//! Windows-only COM plumbing for the classic Outlook Ribbon Add-in.
//! See the crate-level docs in `lib.rs` for how this fits together with the
//! portable `scan_client`/`ribbon_xml` modules.

pub mod guids;
pub mod interfaces;

pub mod automation;
pub mod connect;
pub mod dll_entry;
pub mod register;
pub mod ui;

pub use guids::CLSID_CONNECT;
