#!/usr/bin/env bash
# Run only against isolated fixtures, never a real fnOS installation.
set -euo pipefail
root="$(mktemp -d)"
trap 'rm -rf "$root"' EXIT
export TRIM_PKGVAR="$root/var" TRIM_PKGETC="$root/etc" TRIM_APPDEST="$root/target"
export TRIM_APPNAME="fonu-ci-test-$$" TRIM_TEMP_LOGFILE="$root/install.log"
export wizard_admin_username="test-admin" wizard_admin_password="test-password-123" wizard_admin_password_confirm="test-password-123"
export wizard_admin_port=16893 wizard_http_port=18080 wizard_https_port=19443
mkdir -p "$TRIM_APPDEST/ui"
printf '%s\n' '{".url":{"fonu.Application":{"port":"6893"}}}' > "$TRIM_APPDEST/ui/config"
cmd="$(cd "$(dirname "${BASH_SOURCE[0]}")/../cmd" && pwd)"
bash "$cmd/install_callback"
test "$(stat -c %a "$TRIM_PKGETC/fonu.env")" = 600
test "$(stat -c %a "$TRIM_PKGVAR/admin.bootstrap")" = 600
grep -q '^FONU_ADMIN_PORT=16893$' "$TRIM_PKGETC/fonu.env"
python3 - "$TRIM_PKGVAR/admin.bootstrap" "$TRIM_APPDEST/ui/config" <<'PY'
import json,sys
assert json.load(open(sys.argv[1])) == {"username":"test-admin","password":"test-password-123"}
assert json.load(open(sys.argv[2]))[".url"]["fonu.Application"]["port"] == "16893"
PY
printf 'database fixture' > "$TRIM_PKGVAR/fonu.db"
printf 'secret fixture' > "$TRIM_PKGETC/session_secret"
printf 'cert fixture' > "$TRIM_PKGVAR/certs/test.crt"
before="$(sha256sum "$TRIM_PKGVAR/fonu.db" "$TRIM_PKGETC/session_secret" "$TRIM_PKGETC/fonu.env" "$TRIM_PKGVAR/certs/test.crt")"
bash "$cmd/upgrade_init"
bash "$cmd/upgrade_callback"
test "$before" = "$(sha256sum "$TRIM_PKGVAR/fonu.db" "$TRIM_PKGETC/session_secret" "$TRIM_PKGETC/fonu.env" "$TRIM_PKGVAR/certs/test.crt")"
export wizard_http_port="$wizard_admin_port"
if bash "$cmd/install_callback"; then echo "duplicate port accepted" >&2; exit 1; fi
test "$before" = "$(sha256sum "$TRIM_PKGVAR/fonu.db" "$TRIM_PKGETC/session_secret" "$TRIM_PKGETC/fonu.env" "$TRIM_PKGVAR/certs/test.crt")"
echo "FPK install and upgrade callback checks passed"
