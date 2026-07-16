# SCP-2317 Classic Outlook COM Add-in

This crate builds the native Outlook add-in used for ESET-style Windows desktop integration.
It is not an Office.js add-in and does not use `manifest.xml` sideloading.

The Windows installer installs `scp2317_outlook_comaddin.dll` and calls:

```powershell
regsvr32.exe /s scp2317_outlook_comaddin.dll
```

The DLL handles `DllRegisterServer` itself. Registration writes:

- standard COM class registration for `SCP2317.Connect`
- `HKCU\Software\Microsoft\Office\Outlook\Addins\SCP2317.Connect`
- `LoadBehavior=3`, which tells classic Outlook to load the add-in at startup

Uninstall calls:

```powershell
regsvr32.exe /u /s scp2317_outlook_comaddin.dll
```

This is intentionally current-user registration. It gives the installing Windows user the
embedded classic Outlook UI without a Microsoft 365 admin deployment step.

## Platform limits

- Supported: classic Outlook for Windows.
- Not supported: new Outlook for Windows. Microsoft does not load COM/VSTO add-ins there.
- Cross-platform Outlook coverage remains in `outlook-addin/` through Office.js.

## Build

```powershell
cargo build --manifest-path platform/windows-addin/Cargo.toml --target x86_64-pc-windows-msvc --release
```

The packaging script builds this target and includes the DLL in both the Inno Setup EXE and
WiX MSI artifacts.
