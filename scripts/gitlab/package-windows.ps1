$ErrorActionPreference = 'Stop'

function Assert-LastExitCode {
  param([string]$Step)
  if ($LASTEXITCODE -ne 0) {
    throw "$Step failed with exit code $LASTEXITCODE"
  }
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
  go build -ldflags='-s -w' -o $ApiBinary ./cmd/api
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

  Write-Host "Wrote $ExeOut and $MsiOut"
}
finally {
  Pop-Location
}
