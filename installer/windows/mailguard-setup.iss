#define MyAppName "SCP-2317"
#define MyAppVersion "0.1.0"
#define MyAppPublisher "SCP-2317"
#define MyAppExeName "scp2317-api.exe"

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
Source: "..\..\assets\logo.ico"; DestDir: "{app}"; DestName: "scp2317.ico"; Flags: ignoreversion
Source: "..\..\assets\logo.png"; DestDir: "{app}"; Flags: ignoreversion

[Icons]
Name: "{autoprograms}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\scp2317.ico"
Name: "{autodesktop}\{#MyAppName}"; Filename: "{app}\{#MyAppExeName}"; IconFilename: "{app}\scp2317.ico"; Tasks: desktopicon

[Tasks]
Name: "desktopicon"; Description: "Create a desktop icon"; GroupDescription: "Additional icons:"; Flags: unchecked
