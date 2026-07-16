//! Self-registration under `HKEY_CURRENT_USER`.
//!
//! Two independent registration facts have to exist for classic Outlook to
//! load this add-in with no manifest sideload:
//! 1. Standard COM registration (`CLSID` -> `InprocServer32`/`ProgID`), the
//!    same thing `regsvr32` normally drives via `DllRegisterServer`.
//! 2. The Office-specific `...\Outlook\Addins\<ProgID>` key with
//!    `LoadBehavior=3`, which is *not* part of plain COM and which
//!    `regsvr32` does not create on its own — Office reads it itself.
//!
//! Everything is written under `HKEY_CURRENT_USER\Software\Classes` (which
//! Windows transparently merges into `HKEY_CLASSES_ROOT` for the current
//! user) instead of `HKEY_LOCAL_MACHINE`, so registration needs no
//! administrator elevation. The installer runs `regsvr32 /s <dll>` (and
//! `regsvr32 /u /s <dll>` on uninstall) so this file is the single source of
//! truth for exactly what gets written — installer scripts should not
//! duplicate these registry paths.

use super::guids::{CLSID_CONNECT, DESCRIPTION, FRIENDLY_NAME, PROG_ID};
use windows::core::{w, HRESULT, GUID};
use windows::Win32::Foundation::{HMODULE, MAX_PATH};
use windows::Win32::System::LibraryLoader::{
    GetModuleFileNameW, GetModuleHandleExW, GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
};
use windows::Win32::System::Registry::{
    RegCloseKey, RegCreateKeyExW, RegDeleteTreeW, RegSetValueExW, HKEY, HKEY_CURRENT_USER,
    KEY_WRITE, REG_DWORD, REG_OPTION_NON_VOLATILE, REG_SZ,
};

fn clsid_to_registry_string(clsid: &GUID) -> String {
    format!("{{{clsid:?}}}")
}

unsafe fn dll_path() -> Option<String> {
    let mut hmodule = HMODULE::default();
    GetModuleHandleExW(
        GET_MODULE_HANDLE_EX_FLAG_FROM_ADDRESS,
        windows::core::PCWSTR(dll_path as *const () as *const u16),
        &mut hmodule,
    )
    .ok()?;
    let mut buf = vec![0u16; MAX_PATH as usize];
    let len = GetModuleFileNameW(Some(hmodule), &mut buf);
    if len == 0 {
        return None;
    }
    Some(String::from_utf16_lossy(&buf[..len as usize]))
}

unsafe fn set_string_value(key: HKEY, name: Option<&[u16]>, value: &str) -> windows::core::Result<()> {
    let mut wide: Vec<u16> = value.encode_utf16().collect();
    wide.push(0);
    let bytes = std::slice::from_raw_parts(wide.as_ptr() as *const u8, wide.len() * 2);
    let name_ptr = name
        .map(|n| windows::core::PCWSTR::from_raw(n.as_ptr()))
        .unwrap_or(windows::core::PCWSTR::null());
    RegSetValueExW(key, name_ptr, Some(0), REG_SZ, Some(bytes)).ok()
}

unsafe fn set_dword_value(key: HKEY, name: &[u16], value: u32) -> windows::core::Result<()> {
    let bytes = value.to_le_bytes();
    RegSetValueExW(
        key,
        windows::core::PCWSTR::from_raw(name.as_ptr()),
        Some(0),
        REG_DWORD,
        Some(&bytes),
    )
    .ok()
}

unsafe fn create_key(root: HKEY, subkey: &str) -> windows::core::Result<HKEY> {
    let mut wide: Vec<u16> = subkey.encode_utf16().collect();
    wide.push(0);
    let mut hkey = HKEY::default();
    RegCreateKeyExW(
        root,
        windows::core::PCWSTR(wide.as_ptr()),
        Some(0),
        None,
        REG_OPTION_NON_VOLATILE,
        KEY_WRITE,
        None,
        &mut hkey,
        None,
    )
    .ok()?;
    Ok(hkey)
}

unsafe fn register() -> windows::core::Result<()> {
    let clsid_str = clsid_to_registry_string(&CLSID_CONNECT);
    let dll = dll_path().ok_or_else(|| windows::core::Error::from(windows::Win32::Foundation::E_FAIL))?;

    // HKCU\Software\Classes\CLSID\{...}\InprocServer32
    let clsid_key = create_key(
        HKEY_CURRENT_USER,
        &format!("Software\\Classes\\CLSID\\{clsid_str}"),
    )?;
    set_string_value(clsid_key, None, FRIENDLY_NAME)?;
    RegCloseKey(clsid_key).ok()?;

    let inproc_key = create_key(
        HKEY_CURRENT_USER,
        &format!("Software\\Classes\\CLSID\\{clsid_str}\\InprocServer32"),
    )?;
    set_string_value(inproc_key, None, &dll)?;
    set_string_value(inproc_key, Some(w!("ThreadingModel").as_wide()), "Apartment")?;
    RegCloseKey(inproc_key).ok()?;

    let progid_key = create_key(
        HKEY_CURRENT_USER,
        &format!("Software\\Classes\\CLSID\\{clsid_str}\\ProgID"),
    )?;
    set_string_value(progid_key, None, PROG_ID)?;
    RegCloseKey(progid_key).ok()?;

    // HKCU\Software\Classes\<ProgID>\CLSID
    let progid_root = create_key(HKEY_CURRENT_USER, &format!("Software\\Classes\\{PROG_ID}\\CLSID"))?;
    set_string_value(progid_root, None, &clsid_str)?;
    RegCloseKey(progid_root).ok()?;

    // HKCU\Software\Microsoft\Office\Outlook\Addins\<ProgID> (the Office-
    // specific part; LoadBehavior=3 means "load at startup").
    let addin_key = create_key(
        HKEY_CURRENT_USER,
        &format!("Software\\Microsoft\\Office\\Outlook\\Addins\\{PROG_ID}"),
    )?;
    set_dword_value(addin_key, w!("LoadBehavior").as_wide(), 3)?;
    set_string_value(addin_key, Some(w!("FriendlyName").as_wide()), FRIENDLY_NAME)?;
    set_string_value(addin_key, Some(w!("Description").as_wide()), DESCRIPTION)?;
    RegCloseKey(addin_key).ok()?;

    Ok(())
}

unsafe fn unregister() -> windows::core::Result<()> {
    let clsid_str = clsid_to_registry_string(&CLSID_CONNECT);

    let mut path: Vec<u16> = format!("Software\\Microsoft\\Office\\Outlook\\Addins\\{PROG_ID}")
        .encode_utf16()
        .collect();
    path.push(0);
    let _ = RegDeleteTreeW(HKEY_CURRENT_USER, windows::core::PCWSTR(path.as_ptr()));

    let mut path: Vec<u16> = format!("Software\\Classes\\{PROG_ID}").encode_utf16().collect();
    path.push(0);
    let _ = RegDeleteTreeW(HKEY_CURRENT_USER, windows::core::PCWSTR(path.as_ptr()));

    let mut path: Vec<u16> = format!("Software\\Classes\\CLSID\\{clsid_str}")
        .encode_utf16()
        .collect();
    path.push(0);
    let _ = RegDeleteTreeW(HKEY_CURRENT_USER, windows::core::PCWSTR(path.as_ptr()));

    Ok(())
}

/// # Safety
/// Standard `DllRegisterServer` export contract: called by `regsvr32` with
/// no arguments, on an arbitrary thread, before any class factory requests.
#[no_mangle]
unsafe extern "system" fn DllRegisterServer() -> HRESULT {
    match register() {
        Ok(()) => windows::Win32::Foundation::S_OK,
        Err(e) => e.code(),
    }
}

/// # Safety
/// Standard `DllUnregisterServer` export contract, see [`DllRegisterServer`].
#[no_mangle]
unsafe extern "system" fn DllUnregisterServer() -> HRESULT {
    match unregister() {
        Ok(()) => windows::Win32::Foundation::S_OK,
        Err(e) => e.code(),
    }
}
