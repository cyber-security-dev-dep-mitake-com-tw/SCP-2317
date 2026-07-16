#!/usr/bin/env bash
# Generate a SELF-SIGNED code-signing certificate and emit the two CI secrets
# used by .github/workflows/package.yml:
#   SCP2317_CERT_PFX_BASE64  (base64 of the .pfx, single line)
#   SCP2317_CERT_PASSWORD    (the .pfx password)
#
# A self-signed cert is FREE and proves the signing pipeline works, but Windows
# SmartScreen still warns users because it is not chained to a trusted CA. For
# distribution to real users, buy an OV/EV cert instead (see docs/SIGNING.md).
#
# Usage:
#   scripts/gen-selfsigned-cert.sh [output_dir]
# Then either paste the printed values into GitHub secrets, or run:
#   gh secret set SCP2317_CERT_PFX_BASE64 < <outdir>/cert.pfx.b64
#   gh secret set SCP2317_CERT_PASSWORD --body '<the password>'
set -euo pipefail

OUT_DIR="${1:-$(pwd)/codesign-out}"
mkdir -p "$OUT_DIR"

# Password: use $SCP2317_CERT_PASSWORD if set, else generate a random one.
PFX_PW="${SCP2317_CERT_PASSWORD:-$(openssl rand -base64 18)}"
SUBJECT="${CERT_SUBJECT:-/CN=SCP-2317 Development}"

KEY="$OUT_DIR/key.pem"
CRT="$OUT_DIR/cert.pem"
PFX="$OUT_DIR/cert.pfx"
B64="$OUT_DIR/cert.pfx.b64"

echo ">> Generating RSA key + self-signed code-signing certificate..."
openssl req -x509 -newkey rsa:3072 -keyout "$KEY" -out "$CRT" \
  -days 825 -nodes \
  -subj "$SUBJECT" \
  -addext "keyUsage=digitalSignature" \
  -addext "extendedKeyUsage=codeSigning"

echo ">> Bundling into PKCS#12 (.pfx)..."
openssl pkcs12 -export -out "$PFX" -inkey "$KEY" -in "$CRT" -passout pass:"$PFX_PW"

echo ">> Encoding .pfx to single-line base64..."
openssl base64 -A -in "$PFX" > "$B64"

# The private key and cert are no longer needed once the .pfx exists.
rm -f "$KEY" "$CRT"

echo
echo "=================================================================="
echo " Self-signed code-signing cert created in: $OUT_DIR"
echo "   cert.pfx      - the certificate (keep secret, do NOT commit)"
echo "   cert.pfx.b64  - base64 for the CI secret"
echo
echo " Set these two GitHub Actions secrets:"
echo "   SCP2317_CERT_PFX_BASE64 = contents of $B64"
echo "   SCP2317_CERT_PASSWORD   = $PFX_PW"
echo
echo " Or with the gh CLI (from the repo root):"
echo "   gh secret set SCP2317_CERT_PFX_BASE64 < \"$B64\""
echo "   gh secret set SCP2317_CERT_PASSWORD --body '$PFX_PW'"
echo
echo " NOTE: self-signed = Windows SmartScreen will still warn users."
echo "       For trusted distribution, buy an OV/EV cert (docs/SIGNING.md)."
echo " CLEANUP: delete $OUT_DIR once the secrets are set."
echo "=================================================================="
