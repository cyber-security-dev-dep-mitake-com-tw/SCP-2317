# Windows code signing

The Windows packaging job (`.github/workflows/package.yml` →
`scripts/gitlab/package-windows.ps1`) Authenticode-signs the service binary, the
COM add-in, the tray executable, and both installers **when two repository
secrets are present**. Without them the build still succeeds and produces
**unsigned** developer artifacts (the signature-verify step is skipped).

| Secret | Value |
|---|---|
| `SCP2317_CERT_PFX_BASE64` | your code-signing `.pfx`, base64-encoded to a single line |
| `SCP2317_CERT_PASSWORD` | the `.pfx` password |

You only need **one artifact — a `.pfx`** — and the two secrets fall out of it.

## Option A — self-signed (free, CLI, testing only)

Self-signed certs are free and prove the pipeline signs end-to-end, but Windows
SmartScreen/Defender still shows an "unknown publisher" warning because the cert
is not chained to a trusted CA. **Do not distribute self-signed builds to real
users.**

Run the helper (works on macOS/Linux with OpenSSL):

```bash
scripts/gen-selfsigned-cert.sh            # writes ./codesign-out/
# or choose the password and subject:
SCP2317_CERT_PASSWORD='my-strong-pw' CERT_SUBJECT='/CN=Your Org' \
  scripts/gen-selfsigned-cert.sh
```

It prints the two secret values and the `gh` commands to set them. Equivalent
manual steps:

```bash
PFX_PW='choose-a-strong-password'
openssl req -x509 -newkey rsa:3072 -keyout key.pem -out cert.pem -days 825 -nodes \
  -subj "/CN=SCP-2317 Development" \
  -addext "keyUsage=digitalSignature" \
  -addext "extendedKeyUsage=codeSigning"
openssl pkcs12 -export -out cert.pfx -inkey key.pem -in cert.pem -passout pass:"$PFX_PW"
openssl base64 -A -in cert.pfx > cert.pfx.b64
```

On Windows the native equivalent is PowerShell:

```powershell
$c = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=SCP-2317 Development" `
  -CertStoreLocation Cert:\CurrentUser\My -KeyExportPolicy Exportable
$pw = ConvertTo-SecureString 'choose-a-strong-password' -AsPlainText -Force
Export-PfxCertificate -Cert $c -FilePath cert.pfx -Password $pw
[Convert]::ToBase64String([IO.File]::ReadAllBytes('cert.pfx')) | Set-Content cert.pfx.b64
```

## Option B — trusted CA (paid, for distribution)

Buy an **OV** or **EV** code-signing cert from a CA (DigiCert, Sectigo, SSL.com,
GlobalSign). DigiCert et al. do **not** issue free/self-signed certs — that's the
whole point of paying: their root is trusted, so Windows does not warn. You get a
`.pfx` (or export one) plus a password; then just:

```bash
openssl base64 -A -in cert.pfx > cert.pfx.b64
```

and set the two secrets as above.

| | Self-signed | OV/EV from a CA |
|---|---|---|
| Cost | free | ~$200–600/yr+ |
| Signature valid | yes | yes |
| Trusted by Windows (no SmartScreen warning) | **no** | yes (EV instant; OV builds reputation) |

## EV-on-token caveat

Many **EV** certs are issued on a hardware token/HSM and **cannot be exported to
a `.pfx`**, so the base64-pfx method will not work. In that case use cloud
signing (Azure Trusted Signing, or the CA's KeyLocker/KMS `signtool` integration)
— a different workflow wiring. Ask before going this route and the `signtool`
step + workflow can be adapted.

## Set the secrets

```bash
gh secret set SCP2317_CERT_PFX_BASE64 < cert.pfx.b64
gh secret set SCP2317_CERT_PASSWORD --body 'the-password'
```

Or GitHub → Settings → Secrets and variables → Actions → New repository secret.

## Never commit key material

`cert.pfx`, `cert.pfx.b64`, `key.pem`, `cert.pem`, and the `codesign-out/`
directory are secrets. Delete them once the GitHub secrets are set. They are
covered by `.gitignore`.
