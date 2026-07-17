//! Late-bound access to the Outlook Object Model (OOM) via `IDispatch`.
//!
//! We deliberately do *not* generate bindings from Outlook's type library:
//! that would need a build-time dependency on a local Office install. Late
//! binding (`GetIDsOfNames` + `Invoke`, exactly how VBA/VBScript talk to
//! Outlook) works against any installed Outlook version and needs nothing
//! at build time beyond the standard `IDispatch` interface.
//!
//! This is the Outlook **Object Model**, not Extended MAPI — a different,
//! higher-level automation surface than `platform/windows-mapi` targets.

use crate::scan_client::Attachment;
use windows::core::{Result, BSTR, PCWSTR};
use windows::Win32::Globalization::GetUserDefaultLCID;
use windows::Win32::System::Com::{DISPATCH_METHOD, DISPATCH_PROPERTYGET, DISPPARAMS, IDispatch};
use windows::Win32::System::Variant::{VARIANT, VT_BSTR, VT_DISPATCH, VT_I4};

unsafe fn name_to_dispid(disp: &IDispatch, name: &str) -> Result<i32> {
    let wide: Vec<u16> = name.encode_utf16().chain(std::iter::once(0)).collect();
    let pcwstr = PCWSTR(wide.as_ptr());
    let mut dispid = 0i32;
    disp.GetIDsOfNames(&windows::core::GUID::zeroed(), &pcwstr, 1, 0, &mut dispid)?;
    Ok(dispid)
}

unsafe fn invoke(
    disp: &IDispatch,
    name: &str,
    args: &[VARIANT],
    flags: windows::Win32::System::Com::DISPATCH_FLAGS,
) -> Result<VARIANT> {
    let dispid = name_to_dispid(disp, name)?;
    // COM passes positional arguments in *reverse* order.
    let mut rgvarg: Vec<VARIANT> = args.iter().rev().cloned().collect();
    let params = DISPPARAMS {
        rgvarg: if rgvarg.is_empty() {
            std::ptr::null_mut()
        } else {
            rgvarg.as_mut_ptr()
        },
        rgdispidNamedArgs: std::ptr::null_mut(),
        cArgs: rgvarg.len() as u32,
        cNamedArgs: 0,
    };
    let lcid = GetUserDefaultLCID();
    let mut result = VARIANT::default();
    disp.Invoke(
        dispid,
        &windows::core::GUID::zeroed(),
        lcid as u32,
        flags,
        &params,
        Some(&mut result),
        None,
        None,
    )?;
    Ok(result)
}

/// `object.<name>` with no arguments.
pub unsafe fn get_prop(disp: &IDispatch, name: &str) -> Result<VARIANT> {
    invoke(disp, name, &[], DISPATCH_PROPERTYGET)
}

/// `object.<name>(args...)`.
pub unsafe fn call_method(disp: &IDispatch, name: &str, args: &[VARIANT]) -> Result<VARIANT> {
    invoke(disp, name, args, DISPATCH_METHOD)
}

pub unsafe fn variant_as_dispatch(v: &VARIANT) -> Option<IDispatch> {
    if v.Anonymous.Anonymous.vt == VT_DISPATCH {
        v.Anonymous.Anonymous.Anonymous.pdispVal.as_ref().cloned()
    } else {
        None
    }
}

pub unsafe fn variant_as_string(v: &VARIANT) -> Option<String> {
    if v.Anonymous.Anonymous.vt == VT_BSTR {
        let bstr = &v.Anonymous.Anonymous.Anonymous.bstrVal;
        Some(bstr.to_string())
    } else {
        None
    }
}

pub unsafe fn variant_as_i32(v: &VARIANT) -> Option<i32> {
    if v.Anonymous.Anonymous.vt == VT_I4 {
        Some(v.Anonymous.Anonymous.Anonymous.lVal)
    } else {
        None
    }
}

pub fn bstr_arg(value: &str) -> VARIANT {
    VARIANT::from(BSTR::from(value))
}

pub fn i32_arg(value: i32) -> VARIANT {
    VARIANT::from(value)
}

/// Fields read from the current Outlook item, ready to hand to
/// [`crate::scan_client::build_request`].
pub struct CurrentItemFields {
    pub subject: String,
    pub sender_email: String,
    pub sender_display: String,
    pub reply_to: String,
    pub auth_results: String,
    pub received: Vec<String>,
    pub body_preview: String,
    pub attachments: Vec<Attachment>,
}

const BODY_PREVIEW_MAX_CHARS: usize = 4096;

/// Resolves the item the user is currently looking at: the single selected
/// item in an Explorer's reading pane/list, falling back to an open
/// Inspector's item (e.g. a double-clicked message window).
unsafe fn resolve_current_item(application: &IDispatch) -> Result<IDispatch> {
    if let Ok(explorer_v) = get_prop(application, "ActiveExplorer") {
        if let Some(explorer) = variant_as_dispatch(&explorer_v) {
            if let Ok(selection_v) = get_prop(&explorer, "Selection") {
                if let Some(selection) = variant_as_dispatch(&selection_v) {
                    if let Ok(count_v) = get_prop(&selection, "Count") {
                        if variant_as_i32(&count_v).unwrap_or(0) >= 1 {
                            let item_v =
                                call_method(&selection, "Item", &[i32_arg(1)])?;
                            if let Some(item) = variant_as_dispatch(&item_v) {
                                return Ok(item);
                            }
                        }
                    }
                }
            }
        }
    }

    let inspector_v = get_prop(application, "ActiveInspector")?;
    let inspector = variant_as_dispatch(&inspector_v)
        .ok_or_else(|| windows::core::Error::from(windows::Win32::Foundation::E_FAIL))?;
    let item_v = get_prop(&inspector, "CurrentItem")?;
    variant_as_dispatch(&item_v)
        .ok_or_else(|| windows::core::Error::from(windows::Win32::Foundation::E_FAIL))
}

fn guess_content_type(file_name: &str) -> String {
    let lower = file_name.to_ascii_lowercase();
    let known: &[(&str, &str)] = &[
        (".exe", "application/x-msdownload"),
        (".dll", "application/x-msdownload"),
        (".zip", "application/zip"),
        (".docm", "application/vnd.ms-word.document.macroEnabled.12"),
        (".xlsm", "application/vnd.ms-excel.sheet.macroEnabled.12"),
        (".pdf", "application/pdf"),
        (".htm", "text/html"),
        (".html", "text/html"),
        (".js", "text/javascript"),
    ];
    for (ext, mime) in known {
        if lower.ends_with(ext) {
            return mime.to_string();
        }
    }
    "application/octet-stream".to_string()
}

unsafe fn read_attachments(item: &IDispatch) -> Vec<Attachment> {
    let mut out = Vec::new();
    let attachments_v = match get_prop(item, "Attachments") {
        Ok(v) => v,
        Err(_) => return out,
    };
    let Some(attachments) = variant_as_dispatch(&attachments_v) else {
        return out;
    };
    let count = get_prop(&attachments, "Count")
        .ok()
        .and_then(|v| variant_as_i32(&v))
        .unwrap_or(0);
    for i in 1..=count {
        let Ok(item_v) = call_method(&attachments, "Item", &[i32_arg(i)]) else {
            continue;
        };
        let Some(attachment) = variant_as_dispatch(&item_v) else {
            continue;
        };
        let name = get_prop(&attachment, "FileName")
            .ok()
            .and_then(|v| variant_as_string(&v))
            .or_else(|| {
                get_prop(&attachment, "DisplayName")
                    .ok()
                    .and_then(|v| variant_as_string(&v))
            })
            .unwrap_or_else(|| format!("attachment-{i}"));
        out.push(Attachment {
            content_type: guess_content_type(&name),
            name,
        });
    }
    out
}

/// PR_TRANSPORT_MESSAGE_HEADERS (the raw internet headers as a string).
const PR_TRANSPORT_MESSAGE_HEADERS: &str =
    "http://schemas.microsoft.com/mapi/proptag/0x007D001E";

/// Reads the raw internet headers via the item's PropertyAccessor. Returns ""
/// when unavailable (e.g. an item that never traversed SMTP).
unsafe fn read_transport_headers(item: &IDispatch) -> String {
    let Ok(pa_v) = get_prop(item, "PropertyAccessor") else {
        return String::new();
    };
    let Some(pa) = variant_as_dispatch(&pa_v) else {
        return String::new();
    };
    call_method(&pa, "GetProperty", &[bstr_arg(PR_TRANSPORT_MESSAGE_HEADERS)])
        .ok()
        .and_then(|v| variant_as_string(&v))
        .unwrap_or_default()
}

/// Reads the first Reply-To address, if the sender set one.
unsafe fn read_reply_to(item: &IDispatch) -> String {
    let Ok(rr_v) = get_prop(item, "ReplyRecipients") else {
        return String::new();
    };
    let Some(rr) = variant_as_dispatch(&rr_v) else {
        return String::new();
    };
    let count = get_prop(&rr, "Count")
        .ok()
        .and_then(|v| variant_as_i32(&v))
        .unwrap_or(0);
    if count < 1 {
        return String::new();
    }
    let Ok(r1_v) = call_method(&rr, "Item", &[i32_arg(1)]) else {
        return String::new();
    };
    let Some(r1) = variant_as_dispatch(&r1_v) else {
        return String::new();
    };
    get_prop(&r1, "Address")
        .ok()
        .and_then(|v| variant_as_string(&v))
        .unwrap_or_default()
}

/// Reads Subject/Sender/Body/Attachments plus the identity signals the
/// defense-in-depth engines need (display name, reply-to, Authentication-Results,
/// Received chain) off the current item using only the Outlook Object Model
/// (no MAPI). `application` is the `Application` `IDispatch*` Outlook hands
/// `OnConnection`.
pub unsafe fn read_current_item(application: &IDispatch) -> Result<CurrentItemFields> {
    let item = resolve_current_item(application)?;

    let subject = get_prop(&item, "Subject")
        .ok()
        .and_then(|v| variant_as_string(&v))
        .unwrap_or_default();
    let sender_email = get_prop(&item, "SenderEmailAddress")
        .ok()
        .and_then(|v| variant_as_string(&v))
        .unwrap_or_default();
    let sender_display = get_prop(&item, "SenderName")
        .ok()
        .and_then(|v| variant_as_string(&v))
        .unwrap_or_default();
    let reply_to = read_reply_to(&item);
    let (auth_results, received) =
        crate::scan_client::parse_transport_headers(&read_transport_headers(&item));
    let mut body_preview = get_prop(&item, "Body")
        .ok()
        .and_then(|v| variant_as_string(&v))
        .unwrap_or_default();
    if body_preview.len() > BODY_PREVIEW_MAX_CHARS {
        body_preview.truncate(BODY_PREVIEW_MAX_CHARS);
    }
    let attachments = read_attachments(&item);

    Ok(CurrentItemFields {
        subject,
        sender_email,
        sender_display,
        reply_to,
        auth_results,
        received,
        body_preview,
        attachments,
    })
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn guess_content_type_matches_known_extensions() {
        assert_eq!(guess_content_type("invoice.PDF"), "application/pdf");
        assert_eq!(guess_content_type("payload.exe"), "application/x-msdownload");
        assert_eq!(
            guess_content_type("notes.txt"),
            "application/octet-stream"
        );
    }
}
