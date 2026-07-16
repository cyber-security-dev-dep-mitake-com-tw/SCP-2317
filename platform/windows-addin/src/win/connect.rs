//! The `Connect` COM object: implements `IDTExtensibility2` (Office Add-in
//! lifecycle), `IRibbonExtensibility` (Ribbon XML + callbacks) and the
//! `IDispatch` both of those interfaces are built on.
//!
//! Outlook calls `IDispatch::GetIDsOfNames` once per callback name the
//! first time it needs it, caches the returned dispid, then calls
//! `IDispatch::Invoke` with that dispid from then on — exactly the
//! late-binding mechanism VBA/VBScript use, just walked in the other
//! direction (Outlook is the caller here, we are the automation server).

use crate::ribbon_xml::{self, ribbon_xml};
use crate::scan_client::{self, Attachment};
use crate::win::automation::{self, CurrentItemFields};
use crate::win::ui::{show_message, Severity};
use std::ffi::c_void;
use std::sync::Mutex;
use windows::core::{implement, BSTR, GUID, HRESULT, Interface, OutRef, PCWSTR, Ref, Result};
use windows::Win32::Foundation::{DISP_E_MEMBERNOTFOUND, E_NOTIMPL, S_OK};
use windows::Win32::System::Com::{
    DISPATCH_FLAGS, DISPPARAMS, EXCEPINFO, IDispatch, ITypeInfo, SAFEARRAY,
};
use windows::Win32::System::Variant::VARIANT;

use super::interfaces::{IDTExtensibility2, IDTExtensibility2_Impl, IRibbonExtensibility, IRibbonExtensibility_Impl};

const DISPID_ON_LOAD: i32 = 1;
const DISPID_QUICK_SCAN: i32 = 2;
const DISPID_REPORT: i32 = 3;
const DISPID_STATUS: i32 = 4;
const DISPID_ABOUT: i32 = 5;
const DISPID_GET_IMAGE: i32 = 6;

fn dispid_for_name(name: &str) -> Option<i32> {
    if name.eq_ignore_ascii_case(ribbon_xml::CB_ON_LOAD) {
        Some(DISPID_ON_LOAD)
    } else if name.eq_ignore_ascii_case(ribbon_xml::CB_QUICK_SCAN) {
        Some(DISPID_QUICK_SCAN)
    } else if name.eq_ignore_ascii_case(ribbon_xml::CB_REPORT_SUSPICIOUS) {
        Some(DISPID_REPORT)
    } else if name.eq_ignore_ascii_case(ribbon_xml::CB_CHECK_STATUS) {
        Some(DISPID_STATUS)
    } else if name.eq_ignore_ascii_case(ribbon_xml::CB_ABOUT) {
        Some(DISPID_ABOUT)
    } else if name.eq_ignore_ascii_case(ribbon_xml::CB_GET_IMAGE) {
        Some(DISPID_GET_IMAGE)
    } else {
        None
    }
}

#[implement(IDTExtensibility2, IRibbonExtensibility, IDispatch)]
pub struct Connect {
    application: Mutex<Option<IDispatch>>,
    api_base: String,
}

impl Connect {
    pub fn new() -> Self {
        let api_base = std::env::var("SCP2317_API_BASE")
            .unwrap_or_else(|_| scan_client::DEFAULT_API_BASE.to_string());
        Self {
            application: Mutex::new(None),
            api_base,
        }
    }

    fn current_item(&self) -> Option<CurrentItemFields> {
        let guard = self.application.lock().ok()?;
        let app = guard.as_ref()?;
        unsafe { automation::read_current_item(app).ok() }
    }

    fn attachments_for_request(fields: &CurrentItemFields) -> Vec<Attachment> {
        fields
            .attachments
            .iter()
            .map(|a| Attachment {
                name: a.name.clone(),
                content_type: a.content_type.clone(),
            })
            .collect()
    }

    fn run_quick_scan(&self) {
        let Some(fields) = self.current_item() else {
            show_message(
                "SCP-2317",
                "No message is selected or open. Select or open a message, then try again.",
                Severity::Warning,
            );
            return;
        };
        let req = scan_client::build_request(
            &fields.sender_email,
            &fields.sender_email,
            &fields.subject,
            &fields.body_preview,
            Self::attachments_for_request(&fields),
        );
        match scan_client::call_scan_api(&self.api_base, &req) {
            Ok(verdict) => {
                let severity = if verdict.status == "malicious" {
                    Severity::Error
                } else if verdict.status == "suspicious" {
                    Severity::Warning
                } else {
                    Severity::Info
                };
                show_message("SCP-2317 Quick Scan", &scan_client::verdict_text(&verdict), severity);
            }
            Err(err) => show_message(
                "SCP-2317 Quick Scan",
                &format!("Scan failed: {err}"),
                Severity::Error,
            ),
        }
    }

    fn run_report_suspicious(&self) {
        let Some(fields) = self.current_item() else {
            show_message(
                "SCP-2317",
                "No message is selected or open. Select or open a message, then try again.",
                Severity::Warning,
            );
            return;
        };
        let message_ref = if fields.sender_email.is_empty() {
            fields.subject.clone()
        } else {
            fields.sender_email.clone()
        };
        let req = scan_client::build_request(
            &message_ref,
            &fields.sender_email,
            &fields.subject,
            &fields.body_preview,
            Self::attachments_for_request(&fields),
        );
        let verdict = scan_client::call_scan_api(&self.api_base, &req);
        match scan_client::report_suspicious(&self.api_base, &message_ref) {
            Ok(report) => {
                let verdict_summary = verdict
                    .map(|v| scan_client::verdict_text(&v))
                    .unwrap_or_else(|e| format!("(scan failed: {e})"));
                show_message(
                    "SCP-2317 Report Suspicious",
                    &format!(
                        "Reported (job {}, status stays not_executed). {verdict_summary}",
                        report.job_id
                    ),
                    Severity::Info,
                );
            }
            Err(err) => show_message(
                "SCP-2317 Report Suspicious",
                &format!("Report failed: {err}"),
                Severity::Error,
            ),
        }
    }

    fn run_check_status(&self) {
        let healthy = scan_client::check_api_health(&self.api_base);
        show_message(
            "SCP-2317 API Status",
            &if healthy {
                format!("SCP-2317 API is reachable at {}.", self.api_base)
            } else {
                format!("SCP-2317 API is unreachable at {}.", self.api_base)
            },
            if healthy { Severity::Info } else { Severity::Error },
        );
    }

    fn run_about(&self) {
        show_message(
            "About SCP-2317",
            &format!(
                "SCP-2317 Email Scanner (native Outlook add-in) {}\nHeuristic MVP. Attachments are never executed.\nAPI: {}",
                env!("CARGO_PKG_VERSION"),
                self.api_base
            ),
            Severity::Info,
        );
    }
}

impl Default for Connect {
    fn default() -> Self {
        Self::new()
    }
}

// SAFETY: see `win/interfaces.rs` — these signatures use non-owning raw
// pointers/`Ref`/`OutRef` to match what Outlook actually passes, which is
// *not* the same as the "obvious" owned-type signature an LLM or naive port
// from C# would produce. Do not change parameter types without re-reading
// that module's doc comment.
impl IDTExtensibility2_Impl for Connect_Impl {
    unsafe fn OnConnection(
        &self,
        application: *const c_void,
        _connect_mode: i32,
        _add_in_inst: *const c_void,
        _custom: *const SAFEARRAY,
    ) -> HRESULT {
        let ptr = application as *mut c_void;
        if let Some(disp) = IDispatch::from_raw_borrowed(&ptr) {
            if let Ok(mut guard) = self.application.lock() {
                *guard = Some(disp.clone());
            }
        }
        S_OK
    }

    unsafe fn OnDisconnection(&self, _remove_mode: i32, _custom: *const SAFEARRAY) -> HRESULT {
        if let Ok(mut guard) = self.application.lock() {
            *guard = None;
        }
        S_OK
    }

    unsafe fn OnAddInsUpdate(&self, _custom: *const SAFEARRAY) -> HRESULT {
        S_OK
    }

    unsafe fn OnStartupComplete(&self, _custom: *const SAFEARRAY) -> HRESULT {
        S_OK
    }

    unsafe fn OnBeginShutdown(&self, _custom: *const SAFEARRAY) -> HRESULT {
        S_OK
    }
}

impl IRibbonExtensibility_Impl for Connect_Impl {
    unsafe fn GetCustomUI(&self, _ribbon_id: Ref<BSTR>, out: OutRef<BSTR>) -> HRESULT {
        match out.write(BSTR::from(ribbon_xml())) {
            Ok(()) => S_OK,
            Err(e) => e.code(),
        }
    }
}

impl windows::Win32::System::Com::IDispatch_Impl for Connect_Impl {
    fn GetTypeInfoCount(&self) -> Result<u32> {
        Ok(0)
    }

    fn GetTypeInfo(&self, _itinfo: u32, _lcid: u32) -> Result<ITypeInfo> {
        Err(E_NOTIMPL.into())
    }

    fn GetIDsOfNames(
        &self,
        _riid: *const GUID,
        rgsz_names: *const PCWSTR,
        c_names: u32,
        _lcid: u32,
        rg_disp_id: *mut i32,
    ) -> Result<()> {
        if c_names == 0 || rgsz_names.is_null() || rg_disp_id.is_null() {
            return Err(DISP_E_MEMBERNOTFOUND.into());
        }
        // Ribbon callbacks are always invoked by member name only (no named
        // arguments), so we only need to resolve the first entry.
        let name = unsafe { (*rgsz_names).to_string().unwrap_or_default() };
        match dispid_for_name(&name) {
            Some(id) => {
                unsafe { *rg_disp_id = id };
                Ok(())
            }
            None => {
                unsafe { *rg_disp_id = -1 };
                Err(DISP_E_MEMBERNOTFOUND.into())
            }
        }
    }

    fn Invoke(
        &self,
        disp_id_member: i32,
        _riid: *const GUID,
        _lcid: u32,
        _flags: DISPATCH_FLAGS,
        _params: *const DISPPARAMS,
        result: *mut VARIANT,
        _excep_info: *mut EXCEPINFO,
        _arg_err: *mut u32,
    ) -> Result<()> {
        match disp_id_member {
            DISPID_ON_LOAD => Ok(()),
            DISPID_QUICK_SCAN => {
                self.run_quick_scan();
                Ok(())
            }
            DISPID_REPORT => {
                self.run_report_suspicious();
                Ok(())
            }
            DISPID_STATUS => {
                self.run_check_status();
                Ok(())
            }
            DISPID_ABOUT => {
                self.run_about();
                Ok(())
            }
            DISPID_GET_IMAGE => {
                // MVP: no branded icon returned yet (buttons fall back to
                // Office's default). Wiring a real IPictureDisp from our
                // installed .ico via OleLoadPicturePath is tracked in
                // platform/windows-addin/README.md.
                if !result.is_null() {
                    unsafe { *result = VARIANT::default() };
                }
                Ok(())
            }
            _ => Err(DISP_E_MEMBERNOTFOUND.into()),
        }
    }
}
