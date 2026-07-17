//! Native, dependency-free result UI.
//!
//! MVP scope: verdicts are shown with a plain Win32 message box. This is
//! deliberately not a WebView2-hosted task pane or an Outlook InfoBar —
//! both work, but are a meaningfully larger surface (hosting, IPC, extra
//! signing/packaging) that isn't justified until the Ribbon/COM plumbing
//! itself has been validated against a real Outlook install. Tracked as a
//! documented follow-up in `platform/windows-addin/README.md`.

use windows::core::PCWSTR;
use windows::Win32::UI::Shell::ShellExecuteW;
use windows::Win32::UI::WindowsAndMessaging::{
    MessageBoxW, SW_SHOWNORMAL, MB_ICONERROR, MB_ICONINFORMATION, MB_ICONWARNING, MB_OK,
};

pub enum Severity {
    Info,
    Warning,
    Error,
}

fn to_wide(s: &str) -> Vec<u16> {
    s.encode_utf16().chain(std::iter::once(0)).collect()
}

/// Shows a modal message box on the current thread. Safe to call from any
/// thread with a message queue; callers running scan work on a background
/// thread should marshal back or accept the (harmless) extra message loop.
pub fn show_message(title: &str, body: &str, severity: Severity) {
    let wtitle = to_wide(title);
    let wbody = to_wide(body);
    let icon = match severity {
        Severity::Info => MB_ICONINFORMATION,
        Severity::Warning => MB_ICONWARNING,
        Severity::Error => MB_ICONERROR,
    };
    unsafe {
        MessageBoxW(
            None,
            PCWSTR(wbody.as_ptr()),
            PCWSTR(wtitle.as_ptr()),
            MB_OK | icon,
        );
    }
}

/// Opens a URL in the user's default browser via the shell. Used to launch the
/// local diagnostics dashboard served by the Go API.
pub fn open_in_browser(url: &str) {
    let wurl = to_wide(url);
    let wopen = to_wide("open");
    unsafe {
        ShellExecuteW(
            None,
            PCWSTR(wopen.as_ptr()),
            PCWSTR(wurl.as_ptr()),
            PCWSTR::null(),
            PCWSTR::null(),
            SW_SHOWNORMAL,
        );
    }
}
