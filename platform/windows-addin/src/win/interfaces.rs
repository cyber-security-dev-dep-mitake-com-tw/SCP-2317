//! Custom COM interface declarations that `windows-rs` does not ship
//! (they come from Office's own type libraries, not Win32 metadata).
//!
//! The parameter types below intentionally use raw, non-owning pointers
//! (`*const c_void`, `*const SAFEARRAY`) for `IDTExtensibility2`, and
//! `Ref`/`OutRef` (borrowed, not owned) for the `BSTR` in
//! `IRibbonExtensibility::GetCustomUI`. This matches how Outlook actually
//! calls these methods: it owns and frees the underlying `BSTR`/`SAFEARRAY`
//! memory itself. An earlier real-world Rust Outlook add-in
//! (https://tritium.legal/blog/outlook) shipped with owned types here
//! (`Option<&IDispatch>`, `SAFEARRAY`, `BSTR`) and hit random use-after-free
//! crashes in *other* vendors' add-ins because `windows-rs`'s generated
//! `Drop` impls freed memory Outlook still owned. Do not "simplify" these
//! signatures back to owned types.

use windows::core::{interface, OutRef, Ref, BSTR, HRESULT};
use windows::Win32::System::Com::{IDispatch, SAFEARRAY};
use core::ffi::c_void;

#[interface("B65AD801-ABAF-11D0-BB8B-00A0C90F2744")]
pub unsafe trait IDTExtensibility2: IDispatch {
    unsafe fn OnConnection(
        &self,
        application: *const c_void,
        connect_mode: i32,
        add_in_inst: *const c_void,
        custom: *const SAFEARRAY,
    ) -> HRESULT;
    unsafe fn OnDisconnection(&self, remove_mode: i32, custom: *const SAFEARRAY) -> HRESULT;
    unsafe fn OnAddInsUpdate(&self, custom: *const SAFEARRAY) -> HRESULT;
    unsafe fn OnStartupComplete(&self, custom: *const SAFEARRAY) -> HRESULT;
    unsafe fn OnBeginShutdown(&self, custom: *const SAFEARRAY) -> HRESULT;
}

#[interface("000C0396-0000-0000-C000-000000000046")]
pub unsafe trait IRibbonExtensibility: IDispatch {
    unsafe fn GetCustomUI(&self, ribbon_id: Ref<BSTR>, out: OutRef<BSTR>) -> HRESULT;
}
