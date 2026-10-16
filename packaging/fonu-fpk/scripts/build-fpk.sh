#!/usr/bin/env bash
# Build Fonu native .fpk for fnOS (linux/amd64 or linux/arm64).
# Usage: ./scripts/build-fpk.sh <amd64|arm64> [version]
# Intended for Linux CI (GitHub Actions). Requires: curl, go, node/npm, fnpack, python3 or ar.
set -euo pipefail

ARCH="${1:?usage: build-fpk.sh amd64|arm64 [version]}"
VERSION="${2:-dev}"
FRP_VERSION="${FRP_VERSION:-0.71.0}"

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PACK_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"
REPO_ROOT="$(cd "${PACK_DIR}/../.." && pwd)"
STAGING="${PACK_DIR}/.build-${ARCH}"
DIST_DIR="${REPO_ROOT}/packaging/dist"

resolve_fnpack() {
  if [ -n "${FNPACK_BIN:-}" ] && [ -f "${FNPACK_BIN}" ]; then
    echo "${FNPACK_BIN}"
    return 0
  fi
  if command -v fnpack >/dev/null 2>&1; then
    command -v fnpack
    return 0
  fi
  local cache="${PACK_DIR}/.cache"
  if [ -d "${cache}" ]; then
    local f
    for f in "${cache}"/fnpack*; do
      [ -f "${f}" ] || continue
      chmod +x "${f}" 2>/dev/null || true
      echo "${f}"
      return 0
    done
  fi
  return 1
}

if ! FNPACK="$(resolve_fnpack)"; then
  echo "fnpack not found." >&2
  echo "  Put fnpack exe in: ${PACK_DIR}/.cache/fnpack-1.2.3-windows-amd64.exe" >&2
  echo "  Or set FNPACK_BIN, or install fnpack on PATH." >&2
  echo "  https://developer.fnnas.com/docs/cli/fnpack" >&2
  exit 1
fi
echo "==> Using fnpack: ${FNPACK}"

case "${ARCH}" in
  amd64)
    GOARCH=amd64
    FRP_ARCH=amd64
    MANIFEST_PLATFORM=x86_64
    ;;
  arm64)
    GOARCH=arm64
    FRP_ARCH=arm64
    MANIFEST_PLATFORM=aarch64
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
  find "${root}" -type f \( -name 'lib*.so*' -o -name '*.so.*' \) 2>/dev/null | while read -r so; do
    cp -n "${so}" "${LIB_DIR}/" 2>/dev/null || true
  done
}

bundle_nginx() {
  local cache="${PACK_DIR}/.cache/nginx-${ARCH}"
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
      "${PACK_DIR}/" "${STAGING}/"
    return 0
  fi
  local name
  for name in manifest ICON.PNG ICON_256.PNG LICENSE cmd config wizard app; do
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

chmod +x "${STAGING}/cmd/main"

echo "==> fnpack build"
(
  cd "${STAGING}"
  "${FNPACK}" build --directory .
)

mkdir -p "${DIST_DIR}"
FPK_NAME="fonu_${VERSION}_${ARCH}.fpk"
if [ -f "${STAGING}/fonu.fpk" ]; then
  mv -f "${STAGING}/fonu.fpk" "${DIST_DIR}/${FPK_NAME}"
elif compgen -G "${STAGING}/*.fpk" >/dev/null; then
  mv -f "${STAGING}/"*.fpk "${DIST_DIR}/${FPK_NAME}"
else
  echo "fnpack did not produce .fpk under ${STAGING}" >&2
  exit 1
fi

rm -rf "${STAGING}"
echo "==> Done: ${DIST_DIR}/${FPK_NAME}"
