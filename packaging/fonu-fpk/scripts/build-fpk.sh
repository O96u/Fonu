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

create_archive() {
  local source_dir="$1" output_file="$2"
  if ! tar --version 2>/dev/null | grep -q 'GNU tar'; then
    echo "GNU tar is required to build an fnOS .fpk" >&2
    return 1
  fi
  tar --format=gnu --sort=name --mtime='UTC 2026-09-02 00:00:00' \
    --owner=0 --group=0 --numeric-owner \
    -czf "$output_file" -C "$source_dir" .
}

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
if grep -q '^version=' "${MANIFEST}"; then
  sed_inplace "s/^version=.*/version=${VERSION}/" "${MANIFEST}"
else
  echo "version=${VERSION}" >>"${MANIFEST}"
fi
if grep -q '^platform=' "${MANIFEST}"; then
  sed_inplace "s/^platform=.*/platform=${MANIFEST_PLATFORM}/" "${MANIFEST}"
else
  echo "platform=${MANIFEST_PLATFORM}" >>"${MANIFEST}"
fi

chmod +x "${STAGING}/cmd/"*
rm -f "${STAGING}/LICENSE"

# Windows checkouts are CRLF. fnOS runs these as shell scripts and parses
# manifest as text; a trailing CR fails install with "应用包不符合系统要求".
normalize_lf() {
  local f
  while IFS= read -r -d '' f; do
    if grep -q $'\r' "${f}" 2>/dev/null; then
      sed_inplace $'s/\r$//' "${f}"
    fi
  done < <(find "${STAGING}" -type f \
    ! -name '*.png' ! -name '*.PNG' \
    ! -name 'fonu' ! -name 'nginx' ! -name 'frpc' \
    ! -name '*.so' ! -name '*.so.*' \
    -print0)
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
def size(path):
    with open(path, "rb") as f:
        sig = f.read(8)
        if sig != b"\x89PNG\r\n\x1a\n":
            raise SystemExit(f"{path} is not a PNG")
        f.read(8)
        return struct.unpack(">II", f.read(8))
bad = False
for rel, want in expect.items():
    got = size(f"{root}/{rel}")
    if got != want:
        print(f"{rel} is {got[0]}x{got[1]}, need {want[0]}x{want[1]}", file=sys.stderr)
        bad = True
if bad:
    raise SystemExit(1)
print("icons ok")
PY
}
if command -v python3 >/dev/null 2>&1; then
  verify_icons
fi

echo "==> Packing app.tgz and .fpk (GNU tar, uid 0)"
create_archive "${STAGING}/app" "${STAGING}/app.tgz"
rm -rf "${STAGING}/app"
if command -v md5sum >/dev/null 2>&1; then
  APP_MD5="$(md5sum "${STAGING}/app.tgz" | awk '{print $1}')"
else
  APP_MD5="$(md5 -q "${STAGING}/app.tgz")"
fi
grep -v -E '^[[:space:]]*checksum[[:space:]]*=' "${MANIFEST}" >"${MANIFEST}.tmp" || true
mv "${MANIFEST}.tmp" "${MANIFEST}"
printf 'checksum=%s\n' "${APP_MD5}" >>"${MANIFEST}"

PACK_ROOT="$(mktemp -d)"
cp -a "${STAGING}/manifest" "${STAGING}/cmd" "${STAGING}/config" "${STAGING}/wizard" \
  "${STAGING}/ICON.PNG" "${STAGING}/ICON_256.PNG" "${STAGING}/app.tgz" "${PACK_ROOT}/"
mkdir -p "${DIST_DIR}"
FPK_NAME="fonu_${VERSION}_${ARCH}.fpk"
create_archive "${PACK_ROOT}" "${DIST_DIR}/${FPK_NAME}"
rm -rf "${PACK_ROOT}"

echo "==> Packaged manifest:"
grep -E '^(version|platform|os_min_version)=' "${STAGING}/manifest" 2>/dev/null || true
if [ -f "${DIST_DIR}/${FPK_NAME}" ]; then
  echo "==> FPK size: $(wc -c <"${DIST_DIR}/${FPK_NAME}") bytes"
  tar -tzf "${DIST_DIR}/${FPK_NAME}" 2>/dev/null | head -n 8 || true
fi

rm -rf "${STAGING}"
echo "==> Done: ${DIST_DIR}/${FPK_NAME}"
