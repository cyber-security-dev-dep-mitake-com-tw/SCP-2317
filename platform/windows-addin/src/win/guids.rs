//! Well-known and project-specific COM identifiers.
//!
//! `CLSID_CONNECT` was generated once with `uuidgen` and must stay stable
//! across releases: the installer writes it into the registry, and changing
//! it would silently orphan any previously-registered add-in.

use windows::core::GUID;

/// Our `Connect` COM class. Do not regenerate.
pub const CLSID_CONNECT: GUID = GUID::from_u128(0x24DA2E20_FE3A_4F63_8022_50F15C7DFC26);

/// `ProgID` registered alongside `CLSID_CONNECT`, and the key name Outlook
/// looks for under `...\Outlook\Addins`.
pub const PROG_ID: &str = "SCP2317.Connect";

pub const FRIENDLY_NAME: &str = "SCP-2317 Email Scanner";
pub const DESCRIPTION: &str =
    "Heuristic email scan add-in for classic Outlook. Attachments are never executed.";

/// `IDTExtensibility2`, the interface every Office COM Add-in implements.
/// https://learn.microsoft.com/office/vba/api/overview/shared/idtextensibility2-onconnection
pub const IID_IDTEXTENSIBILITY2: GUID = GUID::from_u128(0xB65AD801_ABAF_11D0_BB8B_00A0C90F2744);

/// `IRibbonExtensibility`, used to supply RibbonX (Custom UI) markup.
pub const IID_IRIBBONEXTENSIBILITY: GUID = GUID::from_u128(0x000C0396_0000_0000_C000_000000000046);
