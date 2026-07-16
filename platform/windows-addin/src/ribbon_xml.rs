//! The RibbonX (Custom UI) markup returned from `IRibbonExtensibility::GetCustomUI`.
//!
//! This is a *different* technology from the Office.js `CustomTab` used by
//! `outlook-addin/manifest.xml`: it is rendered by classic Outlook itself
//! from a COM Add-in, is not restricted to any particular Office.js host
//! surface, and needs no manifest sideload. Keeping it in a plain function
//! (rather than inline in the COM glue) lets it be unit tested on any OS.
//!
//! Callback name constants are shared with `win::connect` so the mapping
//! between Ribbon XML `onAction`/`getImage`/`onLoad` attributes and our
//! `IDispatch::GetIDsOfNames` table can't drift apart.

pub const TAB_ID: &str = "SCP2317.RibbonTab";
pub const GROUP_ID: &str = "SCP2317.ScanGroup";

pub const CB_ON_LOAD: &str = "OnRibbonLoad";
pub const CB_QUICK_SCAN: &str = "OnQuickScanClick";
pub const CB_REPORT_SUSPICIOUS: &str = "OnReportSuspiciousClick";
pub const CB_CHECK_STATUS: &str = "OnCheckStatusClick";
pub const CB_ABOUT: &str = "OnAboutClick";
pub const CB_GET_IMAGE: &str = "OnGetButtonImage";

/// All callback names the add-in's `IDispatch` implementation must resolve
/// via `GetIDsOfNames`. Kept as a slice so `connect.rs` can build its dispid
/// table from the same list instead of repeating the strings.
pub const CALLBACK_NAMES: &[&str] = &[
    CB_ON_LOAD,
    CB_QUICK_SCAN,
    CB_REPORT_SUSPICIOUS,
    CB_CHECK_STATUS,
    CB_ABOUT,
    CB_GET_IMAGE,
];

pub fn ribbon_xml() -> String {
    format!(
        r#"<customUI xmlns="http://schemas.microsoft.com/office/2009/07/customui" onLoad="{on_load}">
  <ribbon>
    <tabs>
      <tab id="{tab_id}" label="SCP-2317" insertAfterMso="TabMail">
        <group id="{group_id}" label="Email Protection">
          <button id="SCP2317.QuickScan" label="Quick Scan" size="large"
                  screentip="Quick Scan"
                  supertip="Run a fast heuristic scan on the selected message and show the verdict. Attachments are never executed."
                  getImage="{get_image}"
                  onAction="{quick_scan}" />
          <button id="SCP2317.ReportSuspicious" label="Report Suspicious" size="large"
                  screentip="Report Suspicious"
                  supertip="Scan the message and record a sandbox job (status stays not_executed) for follow-up."
                  getImage="{get_image}"
                  onAction="{report}" />
          <button id="SCP2317.CheckStatus" label="Check API Status" size="normal"
                  screentip="Check API Status"
                  supertip="Ping the SCP-2317 scan API and show whether it is reachable."
                  getImage="{get_image}"
                  onAction="{status}" />
          <button id="SCP2317.About" label="About" size="normal"
                  screentip="About SCP-2317"
                  supertip="Show the SCP-2317 add-in version and scope."
                  getImage="{get_image}"
                  onAction="{about}" />
        </group>
      </tab>
    </tabs>
  </ribbon>
</customUI>"#,
        on_load = CB_ON_LOAD,
        tab_id = TAB_ID,
        group_id = GROUP_ID,
        get_image = CB_GET_IMAGE,
        quick_scan = CB_QUICK_SCAN,
        report = CB_REPORT_SUSPICIOUS,
        status = CB_CHECK_STATUS,
        about = CB_ABOUT,
    )
}

#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn ribbon_xml_references_every_declared_callback() {
        let xml = ribbon_xml();
        for name in CALLBACK_NAMES {
            assert!(
                xml.contains(name),
                "ribbon XML is missing a reference to callback `{name}`"
            );
        }
    }

    #[test]
    fn ribbon_xml_declares_tab_and_group_ids() {
        let xml = ribbon_xml();
        assert!(xml.contains(TAB_ID));
        assert!(xml.contains(GROUP_ID));
    }
}
