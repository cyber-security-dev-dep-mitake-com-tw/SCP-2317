$ErrorActionPreference = 'Stop'

$RootDir = (Resolve-Path (Join-Path $PSScriptRoot '..\..')).Path
$OutDir = Join-Path $RootDir 'dist\windows'
$PayloadDir = Join-Path $OutDir 'payload\SCP-2317'
$InstallerDir = Join-Path $RootDir 'installer\windows'
$ApiBinary = Join-Path $PayloadDir 'scp2317-api.exe'
$MsiOut = Join-Path $OutDir 'SCP-2317.msi'

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

  if (-not (Test-Path (Join-Path $RootDir 'assets\logo.ico'))) {
    throw 'assets/logo.ico is missing; generate branding assets before packaging.'
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

  $wix = Get-Command wix -ErrorAction SilentlyContinue
  if ($wix) {
    & $wix.Source build (Join-Path $InstallerDir 'mailguard-product.wxs') -o $MsiOut
  } elseif ((Get-Command candle.exe -ErrorAction SilentlyContinue) -and (Get-Command light.exe -ErrorAction SilentlyContinue)) {
    $objDir = Join-Path $OutDir 'obj'
    New-Item -ItemType Directory -Force -Path $objDir | Out-Null
    & candle.exe -out (Join-Path $objDir 'mailguard-product.wixobj') (Join-Path $InstallerDir 'mailguard-product.wxs')
    & light.exe -out $MsiOut (Join-Path $objDir 'mailguard-product.wixobj')
  } else {
    throw 'WiX toolchain was not found. Install wix.exe or candle/light on the Windows runner.'
  }

  Write-Host "Wrote $(Join-Path $OutDir 'SCP-2317-Setup.exe') and $MsiOut"
}
finally {
  Pop-Location
}
