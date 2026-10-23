#!/usr/bin/env bash
# Build Fonu native .fpk for fnOS (linux/amd64 or linux/arm64).
# Usage: ./scripts/build-fpk.sh <amd64|arm64> [version]
# Intended for Linux CI (GitHub Actions). Requires: curl, go, node/npm, GNU tar, python3 or ar.
# The .fpk is a GNU tar (uid 0), same layout as nginx-web-fnos. fnpack emits a
# PAX archive owned by the CI user, which fnOS rejects as a format mismatch.
set -euo pipefail

ARCH="${1:?usage: build-fpk.sh amd64|arm64 [version]}"
VERSION="${2:-dev}"
FRP_VERSION="${FRP_VERSION:-0.71.0}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACK_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${PACK_DIR}/../.." && pwd)"
STAGING="${PACK_DIR}/.build-${ARCH}"
DIST_DIR="${REPO_ROOT}/packaging/dist"

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

nginx_deb_urls() {
  # Debian bookworm 池内固定版本（仅用于解出 Linux nginx + .so，与 fnOS glibc 接近）
  case "${ARCH}" in
    amd64)
      cat <<'EOF'
https://deb.debian.org/debian/pool/main/n/nginx/nginx_1.22.1-9+deb12u9_amd64.deb
https://deb.debian.org/debian/pool/main/p/pcre2/libpcre2-8-0_10.42-1_amd64.deb
https://deb.debian.org/debian/pool/main/o/openssl/libssl3_3.0.17-1~deb12u2_amd64.deb
https://deb.debian.org/debian/pool/main/z/zlib/zlib1g_1.2.13.dfsg-1_amd64.deb
https://deb.debian.org/debian/pool/main/libx/libxcrypt/libcrypt1_4.4.33-2_amd64.deb
EOF
      ;;
    arm64)
      cat <<'EOF'
https://deb.debian.org/debian/pool/main/n/nginx/nginx_1.22.1-9+deb12u9_arm64.deb
https://deb.debian.org/debian/pool/main/p/pcre2/libpcre2-8-0_10.42-1_arm64.deb
https://deb.debian.org/debian/pool/main/o/openssl/libssl3_3.0.17-1~deb12u2_arm64.deb
https://deb.debian.org/debian/pool/main/z/zlib/zlib1g_1.2.13.dfsg-1_arm64.deb
https://deb.debian.org/debian/pool/main/libx/libxcrypt/libcrypt1_4.4.33-2_arm64.deb
EOF
      ;;
  esac
}

extract_deb_into_libs() {
  local url="$1"
  local work="$2"
  local idx="$3"
  local deb="${work}/pkg-${idx}.deb"
  local root="${work}/root-${idx}"
  local extractor="${SCRIPT_DIR}/extract-deb.py"
  if ! curl -fsSL "${url}" -o "${deb}"; then
    echo "failed to download Debian package: ${url}" >&2
    return 1
  fi
  mkdir -p "${root}"
  if command -v ar >/dev/null 2>&1 && ar t "${deb}" >/dev/null 2>&1; then
    (cd "${root}" && ar x "${deb}" && tar -xf data.tar.*)
  elif [ -f "${extractor}" ] && command -v python3 >/dev/null 2>&1; then
    python3 "${extractor}" "${deb}" "${root}"
  elif [ -f "${extractor}" ] && command -v python >/dev/null 2>&1; then
    python "${extractor}" "${deb}" "${root}"
  else
    echo "need ar, or python3 + ${extractor}, to extract .deb packages" >&2
    return 1
  fi
  if [ -f "${root}/usr/sbin/nginx" ]; then
    install -m 755 "${root}/usr/sbin/nginx" "${BIN_DIR}/nginx"
  fi
  # Debian ships SONAMEs as symlinks (libssl.so.3 -> libssl.so.3.*). Copy the
  # link target under the SONAME so nginx can start via LD_LIBRARY_PATH.
  find "${root}" \( -type f -o -type l \) \( -name 'lib*.so*' -o -name '*.so.*' \) -print0 2>/dev/null |
    while IFS= read -r -d '' so; do
      cp -L "${so}" "${LIB_DIR}/" 2>/dev/null || true
    done
}

verify_nginx_libs() {
  if ! command -v readelf >/dev/null 2>&1; then
    echo "==> readelf not found; skipped nginx library check"
    return 0
  fi
  local missing=0 name
  while IFS= read -r name; do
    [ -z "${name}" ] && continue
    case "${name}" in
      libc.so.*|libm.so.*|libdl.so.*|libpthread.so.*|librt.so.*|ld-linux*)
        continue
        ;;
    esac
    if [ ! -e "${LIB_DIR}/${name}" ]; then
      echo "nginx needs ${name}, missing from ${LIB_DIR}" >&2
      missing=1
    fi
  done < <(readelf -d "${BIN_DIR}/nginx" | sed -n 's/.*Shared library: \[\([^]]*\)\].*/\1/p')
  if [ "${missing}" -ne 0 ]; then
    return 1
  fi
  echo "==> nginx shared libraries present"
}

bundle_nginx() {
  local cache="${PACK_DIR}/.cache/nginx-${ARCH}-2"
  if [ -f "${cache}/nginx" ]; then
    echo "==> Using cached nginx: ${cache}"
    install -m 755 "${cache}/nginx" "${BIN_DIR}/nginx"
    if [ -d "${cache}/lib" ]; then
      cp -n "${cache}/lib/"* "${LIB_DIR}/" 2>/dev/null || true
    fi
    return 0
  fi
  if ! command -v curl >/dev/null 2>&1; then
    echo "need curl to download Debian nginx packages" >&2
    return 1
  fi
  echo "==> Downloading Linux nginx (${ARCH}) from Debian packages (no Docker)"
  local work
  work="$(mktemp -d)"
  local idx=0
  local url
  while IFS= read -r url; do
    [ -z "${url}" ] && continue
    extract_deb_into_libs "${url}" "${work}" "${idx}"
    idx=$((idx + 1))
  done < <(nginx_deb_urls)
  if [ ! -f "${BIN_DIR}/nginx" ]; then
    echo "failed to extract nginx binary" >&2
    rm -rf "${work}"
    return 1
  fi
  mkdir -p "${cache}/lib"
  install -m 755 "${BIN_DIR}/nginx" "${cache}/nginx"
  cp -n "${LIB_DIR}/"* "${cache}/lib/" 2>/dev/null || true
  rm -rf "${work}"
  echo "==> Cached nginx: ${cache}"
}

echo "==> Fonu FPK build (${ARCH}, v${VERSION})"
echo "==> Staging FPK tree"
rm -rf "${STAGING}"
mkdir -p "${STAGING}"
stage_pack_tree() {
  if command -v rsync >/dev/null 2>&1; then
    rsync -a --exclude='.build-*' --exclude='*.fpk' --exclude='scripts' --exclude='.cache' \
      --exclude='LICENSE' \
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

BIN_DIR="${STAGING}/app/bin"
LIB_DIR="${STAGING}/app/lib"
mkdir -p "${BIN_DIR}" "${LIB_DIR}"

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
    -o "${BIN_DIR}/fonu" ./cmd/fonu
)

echo "==> Downloading frpc v${FRP_VERSION} (${FRP_ARCH})"
FRP_TMP="${TMPDIR:-/tmp}"
FRP_TGZ="${FRP_TMP}/frp_${FRP_VERSION}_linux_${FRP_ARCH}.tar.gz"
curl -fsSL "https://github.com/fatedier/frp/releases/download/v${FRP_VERSION}/frp_${FRP_VERSION}_linux_${FRP_ARCH}.tar.gz" \
  -o "${FRP_TGZ}"
tar -xzf "${FRP_TGZ}" -C "${FRP_TMP}"
install -m 755 "${FRP_TMP}/frp_${FRP_VERSION}_linux_${FRP_ARCH}/frpc" "${BIN_DIR}/frpc"
rm -rf "${FRP_TMP}/frp_${FRP_VERSION}_linux_${FRP_ARCH}" "${FRP_TGZ}"

bundle_nginx
verify_nginx_libs

cp "${REPO_ROOT}/deploy/nginx/mime.types" "${BIN_DIR}/mime.types"
mkdir -p "${STAGING}/app/migrations"
cp -R "${REPO_ROOT}/migrations/." "${STAGING}/app/migrations/"

echo "==> Updating manifest (version=${VERSION}, platform=${MANIFEST_PLATFORM})"
MANIFEST="${STAGING}/manifest"
sed_inplace() {
  local expr="$1"
  local file="$2"
  if sed --version 2>/dev/null | grep -qi gnu; then
    sed -i "$expr" "$file"
  else
    sed -i '' "$expr" "$file"
  fi
}
# fnOS appcenter expects manifest lines like "key = value" (see gately/nginx-web-fnos).
sed_inplace "s/@VERSION@/${VERSION}/g" "${MANIFEST}"
sed_inplace "s/@PLATFORM@/${MANIFEST_PLATFORM}/g" "${MANIFEST}"

chmod +x "${STAGING}/cmd/"*
rm -f "${STAGING}/LICENSE"

# Windows checkouts are CRLF. fnOS runs these as shell scripts and parses
# manifest as text; a trailing CR fails install with "应用包不符合系统要求".
normalize_lf() {
  local f
  # Only normalize known text metadata and shell scripts, never bundled binary assets.
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
    "ICON.PNG": (64, 64),
    "ICON_256.PNG": (256, 256),
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

# fnpack performs its own app archive and checksum generation.
# Keep the app/ directory intact and pass a complete native fnOS package tree.
echo "==> Packing with official fnpack"
: "${FNPACK_BIN:=fnpack}"
if ! command -v "${FNPACK_BIN}" >/dev/null 2>&1 && [ ! -x "${FNPACK_BIN}" ]; then
  echo "fnpack missing: set FNPACK_BIN to a trusted fnpack executable" >&2
  exit 1
fi
mkdir -p "${DIST_DIR}"
FPK_NAME="fonu_${VERSION}_${ARCH}.fpk"
# Avoid accidentally publishing old build artifacts.
rm -f "${DIST_DIR}/fonu.fpk" "${DIST_DIR}/${FPK_NAME}" "${STAGING}/fonu.fpk"
(
  cd "${DIST_DIR}"
  "${FNPACK_BIN}" build --directory "${STAGING}"
)
# fnpack versions differ in where the output is written.
GENERATED=""
for candidate in "${DIST_DIR}/fonu.fpk" "${STAGING}/fonu.fpk" "${PACK_DIR}/fonu.fpk" "${REPO_ROOT}/fonu.fpk"; do
  if [ -s "${candidate}" ]; then GENERATED="${candidate}"; break; fi
done
if [ -z "${GENERATED}" ]; then
  echo "fnpack completed but fonu.fpk not found; check fnpack output location" >&2
  exit 1
fi
mv "${GENERATED}" "${DIST_DIR}/${FPK_NAME}"

# Verify the actual release file, not only staging metadata.
python3 - "${DIST_DIR}/${FPK_NAME}" "${MANIFEST_PLATFORM}" "${VERSION}" <<'PYVERIFY'
import hashlib, io, pathlib, tarfile, sys
fpk, platform, version = sys.argv[1:]
with tarfile.open(fpk, 'r:gz') as outer:
    members = {m.name.lstrip('./'): m for m in outer.getmembers()}
    needed = ['manifest','app.tgz','ICON.PNG','ICON_256.PNG',
              'cmd/main','config/privilege','config/resource','wizard/install']
    for name in needed:
        assert name in members, f'missing FPK member: {name}'
    manifest = outer.extractfile(members['manifest']).read().decode('utf-8')
    fields = dict(x.split('=',1) for x in manifest.splitlines() if '=' in x)
    fields = {k.strip():v.strip() for k,v in fields.items()}
    assert fields.get('platform') == platform, fields
    assert fields.get('version') == version, fields
    app = outer.extractfile(members['app.tgz']).read()
    assert hashlib.md5(app).hexdigest() == fields.get('checksum'), 'app.tgz checksum mismatch'
    assert members['cmd/main'].mode & 0o111, 'cmd/main not executable'
    with tarfile.open(fileobj=io.BytesIO(app),mode='r:gz') as inner:
        names = {m.name.lstrip('./') for m in inner.getmembers()}
        assert 'bin/fonu' in names, 'bin/fonu missing'
        assert 'ui/config' in names, 'ui/config missing'
print('FPK archive, metadata, checksum, executable and payload: OK')
PYVERIFY
sha256sum "${DIST_DIR}/${FPK_NAME}"
rm -rf "${STAGING}"
echo "==> Done: ${DIST_DIR}/${FPK_NAME}"
