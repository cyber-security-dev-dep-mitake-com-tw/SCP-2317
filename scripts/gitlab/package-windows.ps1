$ErrorActionPreference = 'Stop'

function Assert-LastExitCode {
  param([string]$Step)
  if ($LASTEXITCODE -ne 0) {
    throw "$Step failed with exit code $LASTEXITCODE"
  }
}

# Authenticode-sign a file when a code-signing certificate is configured via
# SCP2317_CERT_PFX (path to .pfx) and SCP2317_CERT_PASSWORD. When no cert is
# present the build still succeeds but emits UNSIGNED developer artifacts.
# An EV certificate is recommended so Windows SmartScreen does not warn users.
function Invoke-CodeSign {
  param([string]$Path)
  if (-not (Test-Path $Path)) { return }
  $pfx = $env:SCP2317_CERT_PFX
  if (-not $pfx -or -not (Test-Path $pfx)) {
    Write-Warning "SCP2317_CERT_PFX not set/found; leaving UNSIGNED: $Path"
    return
  }
  $signtool = Get-Command signtool.exe -ErrorAction SilentlyContinue
  if (-not $signtool) {
    Write-Warning "signtool.exe not found (install the Windows SDK); leaving UNSIGNED: $Path"
    return
  }
  & $signtool.Source sign /fd SHA256 /f $pfx /p $env:SCP2317_CERT_PASSWORD `
    /tr http://timestamp.digicert.com /td SHA256 $Path
  Assert-LastExitCode -Step "signtool sign $Path"
  Write-Host "Signed $Path"
}

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$OutDir = Join-Path $RootDir 'dist\windows'
$PayloadDir = Join-Path $OutDir 'payload\SCP-2317'
$InstallerDir = Join-Path $RootDir 'installer\windows'
$ApiBinary = Join-Path $PayloadDir 'scp2317-api.exe'
$AddinTarget = 'x86_64-pc-windows-msvc'
$AddinDllName = 'scp2317_outlook_comaddin.dll'
$AddinBuildDll = Join-Path $RootDir "platform\windows-addin\target\$AddinTarget\release\$AddinDllName"
$AddinPayloadDll = Join-Path $PayloadDir $AddinDllName
$MsiOut = Join-Path $OutDir 'SCP-2317.msi'
$LogoIco = Join-Path $RootDir 'assets\logo.ico'
$LogoPng = Join-Path $RootDir 'assets\logo.png'

if (Test-Path $OutDir) {
  Remove-Item -Recurse -Force $OutDir
}
New-Item -ItemType Directory -Force -Path $PayloadDir | Out-Null

Push-Location $RootDir
try {
  $env:GOOS = 'windows'
  $env:GOARCH = 'amd64'
  $env:CGO_ENABLED = '0'
  # -H=windowsgui makes the background service run silently (no console window).
  go build -ldflags='-s -w -H=windowsgui' -o $ApiBinary ./cmd/api
  Assert-LastExitCode -Step 'go build'

  cargo build --manifest-path (Join-Path $RootDir 'platform\windows-addin\Cargo.toml') --target $AddinTarget --release
  Assert-LastExitCode -Step 'cargo build windows-addin'

  if (-not (Test-Path $ApiBinary)) {
    throw "go build reported success but $ApiBinary was not created."
  }
  if (-not (Test-Path $AddinBuildDll)) {
    throw "cargo build reported success but $AddinBuildDll was not created."
  }
  Copy-Item -Force $AddinBuildDll $AddinPayloadDll

  # Optional: build the Julia/QML tray companion into the payload. Skipped when
  # Julia is unavailable on the runner (the core installer does not require it).
  $julia = Get-Command julia -ErrorAction SilentlyContinue
  $TrayExeSrc = Join-Path $RootDir 'dist\tray-julia\bin\scp2317-tray.exe'
  if ($julia) {
    & $julia.Source --project=(Join-Path $RootDir 'platform\tray-julia') (Join-Path $RootDir 'platform\tray-julia\build.jl')
    if (($LASTEXITCODE -eq 0) -and (Test-Path $TrayExeSrc)) {
      Copy-Item -Recurse -Force (Join-Path $RootDir 'dist\tray-julia\*') $PayloadDir
      Invoke-CodeSign -Path (Join-Path $PayloadDir 'bin\scp2317-tray.exe')
    } else {
      Write-Warning 'Julia tray build did not produce an executable; packaging service without tray UI.'
    }
  } else {
    Write-Warning 'julia not found on runner; packaging service without the tray UI.'
  }

  # Sign the service binary and the COM add-in before they are wrapped by the
  # installers (which are then signed too).
  Invoke-CodeSign -Path $ApiBinary
  Invoke-CodeSign -Path $AddinPayloadDll

  if (-not (Test-Path $LogoIco)) {
    throw 'assets/logo.ico is missing; generate branding assets before packaging.'
  }
  if (-not (Test-Path $LogoPng)) {
    throw 'assets/logo.png is missing; generate branding assets before packaging.'
  }

  $iscc = Get-Command ISCC.exe -ErrorAction SilentlyContinue
  if (-not $iscc) {
    $isccPath = Join-Path ${env:ProgramFiles(x86)} 'Inno Setup 6\ISCC.exe'
    if (Test-Path $isccPath) {
      $iscc = @{ Source = $isccPath }
    } else {
      throw 'ISCC.exe was not found. Install Inno Setup 6 on the Windows runner.'
    }
  }
  & $iscc.Source (Join-Path $InstallerDir 'mailguard-setup.iss')
  Assert-LastExitCode -Step 'ISCC.exe (Inno Setup)'

  $ExeOut = Join-Path $OutDir 'SCP-2317-Setup.exe'
  if (-not (Test-Path $ExeOut)) {
    throw "ISCC.exe reported success but $ExeOut was not created."
  }

  # WiX resolves relative Source/SourceFile paths against the process's
  # current directory, not the .wxs file location (unlike Inno Setup above),
  # so the payload/logo paths are passed in as absolute preprocessor variables.
  $wixDefines = @(
    "-d", "ApiBinaryPath=$ApiBinary",
    "-d", "AddinDllPath=$AddinPayloadDll",
    "-d", "LogoIcoPath=$LogoIco",
    "-d", "LogoPngPath=$LogoPng"
  )

  $wix = Get-Command wix -ErrorAction SilentlyContinue
  if ($wix) {
    & $wix.Source build (Join-Path $InstallerDir 'mailguard-product.wxs') @wixDefines -o $MsiOut
    Assert-LastExitCode -Step 'wix build'
  } elseif ((Get-Command candle.exe -ErrorAction SilentlyContinue) -and (Get-Command light.exe -ErrorAction SilentlyContinue)) {
    $objDir = Join-Path $OutDir 'obj'
    New-Item -ItemType Directory -Force -Path $objDir | Out-Null
    & candle.exe `
      "-dApiBinaryPath=$ApiBinary" `
      "-dAddinDllPath=$AddinPayloadDll" `
      "-dLogoIcoPath=$LogoIco" `
      "-dLogoPngPath=$LogoPng" `
      -out (Join-Path $objDir 'mailguard-product.wixobj') `
      (Join-Path $InstallerDir 'mailguard-product.wxs')
    Assert-LastExitCode -Step 'candle.exe'
    & light.exe -out $MsiOut (Join-Path $objDir 'mailguard-product.wixobj')
    Assert-LastExitCode -Step 'light.exe'
  } else {
    throw 'WiX toolchain was not found. Install wix.exe or candle/light on the Windows runner.'
  }

  if (-not (Test-Path $MsiOut)) {
    throw "wix build reported success but $MsiOut was not created."
  }

  # Sign the finished installers.
  Invoke-CodeSign -Path $ExeOut
  Invoke-CodeSign -Path $MsiOut

  Write-Host "Wrote $ExeOut and $MsiOut"
}
finally {
  Pop-Location
}
