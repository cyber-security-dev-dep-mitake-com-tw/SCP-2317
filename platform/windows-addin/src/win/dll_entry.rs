//! Classic in-process COM server entry points (`DllGetClassObject`,
//! `DllCanUnloadNow`) plus a minimal `IClassFactory` for `CLSID_CONNECT`.
//!
//! `DllRegisterServer`/`DllUnregisterServer` live in [`super::register`].

use super::connect::Connect;
use super::guids::CLSID_CONNECT;
use std::ffi::c_void;
use windows::core::{implement, GUID, IUnknown, Interface, Result, BOOL, HRESULT};
use windows::Win32::Foundation::{CLASS_E_CLASSNOTAVAILABLE, CLASS_E_NOAGGREGATION, S_FALSE};
use windows::Win32::System::Com::{IClassFactory, IClassFactory_Impl};

use super::interfaces::IDTExtensibility2;

#[implement(IClassFactory)]
struct ClassFactory;

impl IClassFactory_Impl for ClassFactory_Impl {
    fn CreateInstance(
        &self,
        outer: windows::core::Ref<'_, IUnknown>,
        iid: *const GUID,
        object: *mut *mut c_void,
    ) -> Result<()> {
        if outer.is_some() {
            return Err(CLASS_E_NOAGGREGATION.into());
        }
        let connect: IDTExtensibility2 = Connect::new().into();
        unsafe { connect.query(&*iid, object) }.ok()
    }

    fn LockServer(&self, _lock: BOOL) -> Result<()> {
        Ok(())
    }
}

/// # Safety
/// Called by COM (`regsvr32`/Outlook's own loader) with a valid `rclsid` and
/// `riid`, and a valid `ppv` out-pointer, per the standard `DllGetClassObject`
/// contract.
#[no_mangle]
unsafe extern "system" fn DllGetClassObject(
    rclsid: *const GUID,
    riid: *const GUID,
    ppv: *mut *mut c_void,
) -> HRESULT {
    if rclsid.is_null() || riid.is_null() || ppv.is_null() {
        return windows::Win32::Foundation::E_POINTER;
    }
    if *rclsid != CLSID_CONNECT {
        return CLASS_E_CLASSNOTAVAILABLE;
    }
    let factory: IClassFactory = ClassFactory.into();
    match factory.query(&*riid, ppv).ok() {
        Ok(()) => windows::Win32::Foundation::S_OK,
        Err(e) => e.code(),
    }
}

/// We keep this process-lifetime-only in-proc server (no lock/instance
/// accounting): Outlook loads add-ins once at startup and unloads the whole
/// process at shutdown, so there is no real-world benefit to supporting an
/// earlier unload, and it removes a class of "unloaded while a callback is
/// still executing" bugs.
#[no_mangle]
extern "system" fn DllCanUnloadNow() -> HRESULT {
    S_FALSE
}
