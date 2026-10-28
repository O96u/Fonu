#!/usr/bin/env bash
# Build Fonu native .fpk for fnOS (linux/amd64 or linux/arm64).
# Usage: ./scripts/build-fpk.sh <amd64|arm64> [version]
# Intended for Linux CI (GitHub Actions). Requires: curl, go, node/npm, gcc, make, python3, fnpack.
#
# Bundled nginx is compiled from source (stable 1.30.x) with fnOS native install
# paths, matching /var/apps/<appname>/{target,var,shares} used by official Nginx FPK.
set -euo pipefail

ARCH="${1:?usage: build-fpk.sh amd64|arm64 [version]}"
VERSION="${2:-dev}"
FRP_VERSION="${FRP_VERSION:-0.71.0}"
# Official Nginx FPK tracked 1.28/1.29; those branches are EOL. Fonu bundles
# current stable 1.30.x. Override with NGINX_VERSION=1.30.6 if GitHub is unreachable.
NGINX_VERSION="${NGINX_VERSION:-}"
NGINX_VERSION_FALLBACK="1.30.6"
APP_INSTALL_ROOT="/var/apps/fonu"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACK_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${PACK_DIR}/../.." && pwd)"
STAGING="${PACK_DIR}/.build-${ARCH}"
DIST_DIR="${REPO_ROOT}/packaging/dist"
CACHE_DIR="${PACK_DIR}/.cache"

case "${ARCH}" in
  amd64)
    GOARCH=amd64
    FRP_ARCH=amd64
    MANIFEST_PLATFORM=x86
    ;;
  arm64)
    GOARCH=arm64
    FRP_ARCH=arm64
    MANIFEST_PLATFORM=arm
    ;;
  *)
    echo "unsupported arch: ${ARCH}" >&2
    exit 1
    ;;
esac

sed_inplace() {
  local expr="$1"
  local file="$2"
  if sed --version 2>/dev/null | grep -qi gnu; then
    sed -i "$expr" "$file"
  else
    sed -i '' "$expr" "$file"
  fi
}

resolve_nginx_version() {
  if [ -n "${NGINX_VERSION}" ]; then
    echo "==> Using NGINX_VERSION=${NGINX_VERSION}"
    return 0
  fi
  echo "==> Resolving latest nginx 1.30.x (stable)"
  local releases="" latest=""
  if releases="$(curl -fsSL --retry 3 "https://api.github.com/repos/nginx/nginx/releases?per_page=80")"; then
    if command -v jq >/dev/null 2>&1; then
      latest="$(printf '%s' "${releases}" | jq -r '.[].tag_name' | sed -n 's/^release-\(1\.30\.[0-9][0-9]*\)$/\1/p' | sort -V | tail -n1)"
    else
      latest="$(printf '%s' "${releases}" | grep -oE 'release-1\.30\.[0-9]+' | sed 's/^release-//' | sort -V | tail -n1)"
    fi
  fi
  if [ -z "${latest}" ]; then
    NGINX_VERSION="${NGINX_VERSION_FALLBACK}"
    echo "==> Fallback nginx ${NGINX_VERSION}"
  else
    NGINX_VERSION="${latest}"
    echo "==> Latest nginx 1.30.x is ${NGINX_VERSION}"
  fi
}

compile_nginx() {
  local src="$1"
  local out="$2"
  (
    cd "${src}"
    ./auto/configure \
      --prefix="${APP_INSTALL_ROOT}/var" \
      --sbin-path="${APP_INSTALL_ROOT}/target/Nginx" \
      --modules-path="${APP_INSTALL_ROOT}/shares/etc/modules" \
      --conf-path="${APP_INSTALL_ROOT}/shares/etc/nginx.conf" \
      --pid-path="${APP_INSTALL_ROOT}/var/nginx.pid" \
      --lock-path="${APP_INSTALL_ROOT}/var/nginx.lock" \
      --http-client-body-temp-path="${APP_INSTALL_ROOT}/var/cache/client_temp" \
      --http-proxy-temp-path="${APP_INSTALL_ROOT}/var/cache/proxy_temp" \
      --http-fastcgi-temp-path="${APP_INSTALL_ROOT}/var/cache/fastcgi_temp" \
      --http-uwsgi-temp-path="${APP_INSTALL_ROOT}/var/cache/uwsgi_temp" \
      --http-scgi-temp-path="${APP_INSTALL_ROOT}/var/cache/scgi_temp" \
      --with-http_ssl_module \
      --with-http_v2_module \
      --with-http_realip_module \
      --with-http_sub_module \
      --with-http_dav_module \
      --with-http_mp4_module \
      --with-http_gunzip_module \
      --with-http_gzip_static_module \
      --with-http_auth_request_module \
      --with-http_random_index_module \
      --with-http_secure_link_module \
      --with-http_slice_module \
      --with-http_stub_status_module \
      --with-threads \
      --with-file-aio \
      --with-pcre-jit \
      --with-stream \
      --with-stream_ssl_module \
      --with-mail \
      --with-mail_ssl_module
    make -j"$(nproc)"
  )
  install -m 755 "${src}/objs/nginx" "${out}"
}

bundle_nginx() {
  local cache="${CACHE_DIR}/nginx-${ARCH}-${NGINX_VERSION}"
  if [ -f "${cache}/Nginx" ]; then
    echo "==> Using cached nginx ${NGINX_VERSION}: ${cache}"
    install -m 755 "${cache}/Nginx" "${STAGING}/app/Nginx"
    return 0
  fi
  if ! command -v gcc >/dev/null 2>&1 || ! command -v make >/dev/null 2>&1; then
    echo "need gcc and make to compile nginx from source" >&2
    return 1
  fi
  echo "==> Compiling nginx ${NGINX_VERSION} (${ARCH}) for ${APP_INSTALL_ROOT}"
  local work tarball
  work="$(mktemp -d)"
  tarball="${work}/nginx-${NGINX_VERSION}.tar.gz"
  curl -fsSL --retry 3 "https://github.com/nginx/nginx/archive/release-${NGINX_VERSION}.tar.gz" -o "${tarball}"
  mkdir -p "${work}/src"
  tar -xzf "${tarball}" -C "${work}/src" --strip-components=1
  compile_nginx "${work}/src" "${STAGING}/app/Nginx"
  mkdir -p "${cache}"
  install -m 755 "${STAGING}/app/Nginx" "${cache}/Nginx"
  rm -rf "${work}"
  echo "==> Cached nginx: ${cache}"
}

verify_nginx_binary() {
  if [ ! -x "${STAGING}/app/Nginx" ]; then
    echo "nginx binary missing: ${STAGING}/app/Nginx" >&2
    return 1
  fi
  if command -v file >/dev/null 2>&1; then
    file "${STAGING}/app/Nginx"
  fi
  if ! command -v readelf >/dev/null 2>&1; then
    echo "==> readelf not found; skipped nginx ELF check"
    return 0
  fi
  readelf -d "${STAGING}/app/Nginx" | sed -n 's/.*Shared library: \[\([^]]*\)\].*/\1/p' || true
  echo "==> nginx ${NGINX_VERSION} ready"
}

resolve_fnpack_bin() {
  if [ -n "${FNPACK_BIN:-}" ]; then
    if command -v "${FNPACK_BIN}" >/dev/null 2>&1 || [ -f "${FNPACK_BIN}" ]; then
      return 0
    fi
    echo "FNPACK_BIN is set but not executable: ${FNPACK_BIN}" >&2
    return 1
  fi
  if command -v fnpack >/dev/null 2>&1; then
    FNPACK_BIN=fnpack
    return 0
  fi
  local candidate
  for candidate in \
    "${CACHE_DIR}/fnpack" \
    "${CACHE_DIR}/fnpack-1.2.3-linux-amd64" \
    "${CACHE_DIR}/fnpack-1.2.3-linux-arm64" \
    "${CACHE_DIR}/fnpack-1.2.3-windows-amd64.exe"; do
    if [ -f "${candidate}" ]; then
      FNPACK_BIN="${candidate}"
      return 0
    fi
  done
  echo "fnpack missing: set FNPACK_BIN to a trusted fnpack executable" >&2
  return 1
}

echo "==> Fonu FPK build (${ARCH}, v${VERSION})"
# Keep compilation and packaging independently runnable in CI.
MODE="${BUILD_MODE:-all}"
case "$MODE" in all|payload|package) ;; *) echo "invalid BUILD_MODE: $MODE" >&2; exit 1 ;; esac
resolve_fnpack_bin
FNPACK_BIN="$(command -v "$FNPACK_BIN" || printf '%s' "$FNPACK_BIN")"
if [[ "$FNPACK_BIN" != /* ]]; then FNPACK_BIN="$(cd "$(dirname "$FNPACK_BIN")" && pwd)/$(basename "$FNPACK_BIN")"; fi
if [ "$MODE" != package ]; then
resolve_nginx_version
fi

echo "==> Staging FPK tree"
rm -rf "${STAGING}"
mkdir -p "${STAGING}"
stage_pack_tree() {
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --exclude='.build-*' --exclude='*.fpk' --exclude='scripts' --exclude='.cache' \
      --exclude='LICENSE' --exclude='ICON_64.PNG' \
      "${PACK_DIR}/" "${STAGING}/"
    return 0
  fi
  local name
  for name in manifest ICON.PNG ICON_256.PNG cmd config wizard app; do
    if [ -e "${PACK_DIR}/${name}" ]; then
      cp -R "${PACK_DIR}/${name}" "${STAGING}/${name}"
    fi
  done
}

stage_pack_tree
mkdir -p "${STAGING}/app"

if [ "$MODE" != package ]; then
echo "==> Building web (Vite)"
(
  cd "${REPO_ROOT}/web"
  if [ ! -d node_modules ]; then
    npm ci
  fi
  VITE_APP_VERSION="${VERSION}" npm run build
)

echo "==> Building fonu (GOOS=linux GOARCH=${GOARCH})"
(
  cd "${REPO_ROOT}"
  mkdir -p cmd/fonu/web/dist
  rm -rf cmd/fonu/web/dist
  mkdir -p cmd/fonu/web/dist
  cp -R "${REPO_ROOT}/web/dist/." cmd/fonu/web/dist/
  CGO_ENABLED=0 GOOS=linux GOARCH="${GOARCH}" go build \
    -ldflags "-X github.com/fonu/fonu/internal/version.Version=${VERSION}" \
    -o "${STAGING}/app/fonu" ./cmd/fonu
)

echo "==> Downloading frpc v${FRP_VERSION} (${FRP_ARCH})"
FRP_TMP="${TMPDIR:-/tmp}"
FRP_TGZ="${FRP_TMP}/frp_${FRP_VERSION}_linux_${FRP_ARCH}.tar.gz"
FRP_CACHE="${CACHE_DIR}/frpc-${FRP_VERSION}-linux-${FRP_ARCH}"
if [ -f "${FRP_CACHE}" ]; then
  echo "==> Using cached frpc: ${FRP_CACHE}"
  install -m 755 "${FRP_CACHE}" "${STAGING}/app/frpc"
else
  curl -fsSL --retry 3 "https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_linux_${FRP_ARCH}.tar.gz" \
    -o "${FRP_TGZ}"
  tar -xzf "${FRP_TGZ}" -C "${FRP_TMP}"
  install -m 755 "${FRP_TMP}/frp_${FRP_VERSION}_linux_${FRP_ARCH}/frpc" "${STAGING}/app/frpc"
  mkdir -p "${CACHE_DIR}"
  install -m 755 "${STAGING}/app/frpc" "${FRP_CACHE}"
  rm -rf "${FRP_TMP}/frp_${FRP_VERSION}_linux_${FRP_ARCH}" "${FRP_TGZ}"
fi

bundle_nginx
verify_nginx_binary

cp "${REPO_ROOT}/deploy/nginx/mime.types" "${STAGING}/app/mime.types"
mkdir -p "${STAGING}/app/migrations"
cp -R "${REPO_ROOT}/migrations/." "${STAGING}/app/migrations/"

mkdir -p "${DIST_DIR}/payload-${ARCH}"
cp "${STAGING}/app/"{fonu,Nginx,frpc} "${DIST_DIR}/payload-${ARCH}/"
if [ "$MODE" = payload ]; then
  echo "==> Payload ready: ${DIST_DIR}/payload-${ARCH}"
  exit 0
fi
else
  cp "${DIST_DIR}/payload-${ARCH}/"{fonu,Nginx,frpc} "${STAGING}/app/"
  cp "${REPO_ROOT}/deploy/nginx/mime.types" "${STAGING}/app/mime.types"
  mkdir -p "${STAGING}/app/migrations"
  cp -R "${REPO_ROOT}/migrations/." "${STAGING}/app/migrations/"
fi

# Reject wrong-architecture payloads before producing an installable package.
python3 - "${STAGING}/app" "$ARCH" <<'PYELF'
import pathlib, struct, sys
root, arch = sys.argv[1:]
for name in ('fonu', 'Nginx', 'frpc'):
    header = (pathlib.Path(root) / name).read_bytes()[:20]
    if header[:6] != b'\x7fELF\x02\x01' or struct.unpack('<H', header[18:20])[0] != {'amd64':62, 'arm64':183}[arch]:
        raise SystemExit(f'{name}: expected Linux ELF64 {arch}')
print('All payload architectures verified')
PYELF

echo "==> Updating manifest (version=${VERSION}, platform=${MANIFEST_PLATFORM})"
MANIFEST="${STAGING}/manifest"
sed_inplace "s/@VERSION@/${VERSION}/g" "${MANIFEST}"
sed_inplace "s/@PLATFORM@/${MANIFEST_PLATFORM}/g" "${MANIFEST}"

chmod +x "${STAGING}/cmd/"* "${STAGING}/app/fonu" "${STAGING}/app/frpc" "${STAGING}/app/Nginx"
find "${STAGING}/config" "${STAGING}/wizard" "${STAGING}/app/ui" -type f -exec chmod 644 {} +
chmod 644 "${STAGING}/manifest" "${STAGING}/ICON.PNG" "${STAGING}/ICON_256.PNG"
rm -f "${STAGING}/LICENSE"

# Windows checkouts are CRLF. fnOS runs these as shell scripts and parses
# manifest as text; normalize line endings for Linux tooling.
normalize_lf() {
  local f
  while IFS= read -r -d '' f; do
    if grep -q $'\r' "${f}" 2>/dev/null; then
      sed_inplace $'s/\r$//' "${f}"
    fi
  done < <(
    printf '%s\0' "${STAGING}/manifest"
    find "${STAGING}/cmd" -maxdepth 1 -type f -print0
  )
}
normalize_lf

verify_icons() {
  python3 - "${STAGING}" <<'PY'
import struct, sys
root = sys.argv[1]
expect = {
    # fnpack only ships ICON.PNG (64) and ICON_256.PNG at the FPK root.
    "ICON.PNG": (64, 64),
    "ICON_256.PNG": (256, 256),
    "app/ui/images/ICON.PNG": (64, 64),
    "app/ui/images/icon_64.png": (64, 64),
    "app/ui/images/icon_256.png": (256, 256),
}
def png_info(path):
    with open(path, "rb") as f:
        sig = f.read(8)
        if sig != b"\x89PNG\r\n\x1a\n":
            raise SystemExit(f"{path} is not a PNG")
        f.read(8)  # IHDR length + type
        w, h = struct.unpack(">II", f.read(8))
        f.read(1)  # bit depth
        color_type = f.read(1)[0]
        return w, h, color_type
bad = False
for rel, want in expect.items():
    w, h, color_type = png_info(f"{root}/{rel}")
    if (w, h) != want:
        print(f"{rel} is {w}x{h}, need {want[0]}x{want[1]}", file=sys.stderr)
        bad = True
    if color_type != 6:
        print(f"{rel} must be RGBA PNG (color type 6), got {color_type}", file=sys.stderr)
        bad = True
if bad:
    raise SystemExit(1)
print("icons ok")
PY
}
if command -v python3 >/dev/null 2>&1; then
  verify_icons
fi

echo "==> Packing with official fnpack"
resolve_fnpack_bin
chmod +x "${FNPACK_BIN}" 2>/dev/null || true
mkdir -p "${DIST_DIR}"
FPK_NAME="fonu_${VERSION}_${ARCH}.fpk"
rm -f "${DIST_DIR}/fonu.fpk" "${DIST_DIR}/${FPK_NAME}" "${STAGING}/fonu.fpk"
(
  cd "${DIST_DIR}"
  "${FNPACK_BIN}" build --directory "${STAGING}"
)
GENERATED=""
for candidate in "${DIST_DIR}/fonu.fpk" "${STAGING}/fonu.fpk" ; do
  if [ -s "${candidate}" ]; then GENERATED="${candidate}"; break; fi
done
if [ -z "${GENERATED}" ]; then
  echo "fnpack completed but fonu.fpk not found; check fnpack output location" >&2
  exit 1
fi
mv "${GENERATED}" "${DIST_DIR}/${FPK_NAME}"

python3 - "${DIST_DIR}/${FPK_NAME}" "${MANIFEST_PLATFORM}" "${VERSION}" <<'PYVERIFY'
import hashlib, io, pathlib, tarfile, sys
fpk, platform, version = sys.argv[1:]
with tarfile.open(fpk, 'r:gz') as outer:
    members = {m.name.lstrip('./'): m for m in outer.getmembers()}
    needed = ['manifest','app.tgz','ICON.PNG','ICON_256.PNG',
              'cmd/main','config/privilege','config/resource','wizard/install','wizard/config','wizard/uninstall']
    for name in needed:
        assert name in members, f'missing FPK member: {name}'
    manifest = outer.extractfile(members['manifest']).read().decode('utf-8')
    fields = dict(x.split('=',1) for x in manifest.splitlines() if '=' in x)
    fields = {k.strip():v.strip() for k,v in fields.items()}
    assert fields.get('platform') == platform, fields
    assert fields.get('version') == version, fields
    assert fields.get('desktop_applaunchname') == 'fonu.Application', fields
    app = outer.extractfile(members['app.tgz']).read()
    assert hashlib.md5(app).hexdigest() == fields.get('checksum'), 'app.tgz checksum mismatch'
    assert members['cmd/main'].mode & 0o111, 'cmd/main not executable'
    with tarfile.open(fileobj=io.BytesIO(app),mode='r:gz') as inner:
        names = {m.name.lstrip('./') for m in inner.getmembers()}
        for need in ('fonu', 'Nginx', 'frpc', 'mime.types', 'ui/config'):
            assert need in names, f'{need} missing from app.tgz'
print('FPK archive, metadata, checksum, executable and payload: OK')
PYVERIFY
if command -v sha256sum >/dev/null 2>&1; then
  sha256sum "${DIST_DIR}/${FPK_NAME}"
fi
rm -rf "${STAGING}"
echo "==> Done: ${DIST_DIR}/${FPK_NAME}"
