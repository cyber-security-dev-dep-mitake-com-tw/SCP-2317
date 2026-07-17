#define MyAppName "SCP-2317"
#define MyAppVersion "0.2.0"
#define MyAppPublisher "SCP-2317 Email Protection"
#define MyAppExeName "scp2317-api.exe"
#define MyOutlookAddinDll "scp2317_outlook_comaddin.dll"

[Setup]
AppId={{B9B7D7BA-4F54-4B2C-9AF0-0D6D8E9B7D51}}
AppName={#MyAppName}
AppVersion={#MyAppVersion}
AppPublisher={#MyAppPublisher}
DefaultDirName={autopf}\{#MyAppName}
DefaultGroupName={#MyAppName}
OutputDir=..\..\dist\windows
OutputBaseFilename=SCP-2317-Setup
Compression=lzma2
SolidCompression=yes
ArchitecturesAllowed=x64compatible
ArchitecturesInstallIn64BitMode=x64
DisableProgramGroupPage=yes
LicenseFile=..\..\LICENSE
SetupIconFile=..\..\assets\logo.ico
UninstallDisplayIcon={app}\{#MyAppExeName}
WizardStyle=modern
MinVersion=10.0
PrivilegesRequired=admin

[Files]
Source: "..\..\dist\windows\payload\SCP-2317\scp2317-api.exe"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\dist\windows\payload\SCP-2317\{#MyOutlookAddinDll}"; DestDir: "{app}"; Flags: ignoreversion
Source: "..\..\assets\logo.ico"; DestDir: "{app}"; DestName: "scp2317.ico"; Flags: ignoreversion
Source: "..\..\assets\logo.png"; DestDir: "{app}"; Flags: ignoreversion

[Registry]
; Autostart the silent background service for every user at logon. The service
; is built with -H=windowsgui, so it runs with no console window.
Root: HKLM; Subkey: "Software\Microsoft\Windows\CurrentVersion\Run"; ValueType: string; \
  ValueName: "{#MyAppName}"; ValueData: """{app}\{#MyAppExeName}"""; Flags: uninsdeletevalue

[Run]
Filename: "{sys}\regsvr32.exe"; Parameters: "/s ""{app}\{#MyOutlookAddinDll}"""; Flags: runhidden; StatusMsg: "Registering the Outlook add-in..."
; Start the service now so the add-in and dashboard work immediately after install.
Filename: "{app}\{#MyAppExeName}"; Flags: nowait runhidden; StatusMsg: "Starting the SCP-2317 service..."
; Offer to open the local diagnostics dashboard once installed.
Filename: "http://127.0.0.1:8080/dashboard"; Description: "Open the SCP-2317 dashboard"; Flags: postinstall shellexec skipifsilent nowait

[UninstallRun]
Filename: "{sys}\regsvr32.exe"; Parameters: "/u /s ""{app}\{#MyOutlookAddinDll}"""; Flags: runhidden
; Stop the running service before removing files.
Filename: "{sys}\taskkill.exe"; Parameters: "/f /im {#MyAppExeName}"; Flags: runhidden; RunOnceId: "StopSvc"

[Icons]
Name: "{autoprograms}\{#MyAppName} Dashboard"; Filename: "http://127.0.0.1:8080/dashboard"; IconFilename: "{app}\scp2317.ico"
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\scp2317.ico"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\scp2317.ico"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create a desktop icon"; GroupDescription: "Additional icons:"; Flags: unchecked
