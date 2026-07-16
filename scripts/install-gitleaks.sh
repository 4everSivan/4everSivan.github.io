#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "${ROOT_DIR}/config/versions.env"

LOCAL_DIR="${ROOT_DIR}/.local"
OUTPUT_DIR="${LOCAL_DIR}/bin"
OUTPUT="${OUTPUT_DIR}/gitleaks"

ensure_real_directory() {
  local directory="$1"
  if [[ -L "${directory}" ]] || { [[ -e "${directory}" ]] && [[ ! -d "${directory}" ]]; }; then
    echo "Gitleaks install directory is unsafe" >&2
    exit 1
  fi
  mkdir -p "${directory}"
  if [[ -L "${directory}" ]] || [[ ! -d "${directory}" ]]; then
    echo "Gitleaks install directory is unsafe" >&2
    exit 1
  fi
}

ensure_real_directory "${LOCAL_DIR}"
ensure_real_directory "${OUTPUT_DIR}"
if [[ -L "${OUTPUT}" ]] || { [[ -e "${OUTPUT}" ]] && [[ ! -f "${OUTPUT}" ]]; }; then
  echo "Gitleaks executable path is unsafe" >&2
  exit 1
fi

case "$(uname -s)-$(uname -m)" in
  Darwin-arm64)
    ASSET="gitleaks_${GITLEAKS_VERSION}_darwin_arm64.tar.gz"
    EXPECTED_ARCHIVE_SHA256="${GITLEAKS_DARWIN_ARM64_ARCHIVE_SHA256}"
    EXPECTED_BINARY_SHA256="${GITLEAKS_DARWIN_ARM64_BINARY_SHA256}"
    ;;
  Linux-x86_64)
    ASSET="gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz"
    EXPECTED_ARCHIVE_SHA256="${GITLEAKS_LINUX_AMD64_ARCHIVE_SHA256}"
    EXPECTED_BINARY_SHA256="${GITLEAKS_LINUX_AMD64_BINARY_SHA256}"
    ;;
  Linux-aarch64|Linux-arm64)
    ASSET="gitleaks_${GITLEAKS_VERSION}_linux_arm64.tar.gz"
    EXPECTED_ARCHIVE_SHA256="${GITLEAKS_LINUX_ARM64_ARCHIVE_SHA256}"
    EXPECTED_BINARY_SHA256="${GITLEAKS_LINUX_ARM64_BINARY_SHA256}"
    ;;
  *)
    echo "unsupported platform for pinned Gitleaks binary" >&2
    exit 1
    ;;
esac

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

if [[ -f "${OUTPUT}" ]] && [[ ! -L "${OUTPUT}" ]] && [[ -x "${OUTPUT}" ]]; then
  if [[ "$(sha256_file "${OUTPUT}")" == "${EXPECTED_BINARY_SHA256}" ]] && \
     [[ "$("${OUTPUT}" version)" == "${GITLEAKS_VERSION}" ]]; then
    exit 0
  fi
fi

TEMP_DIR="$(mktemp -d)"
STAGED_OUTPUT=""
trap 'rm -rf "${TEMP_DIR}"; if [[ -n "${STAGED_OUTPUT}" ]]; then rm -f "${STAGED_OUTPUT}"; fi' EXIT
ARCHIVE="${TEMP_DIR}/${ASSET}"
URL="https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/${ASSET}"

curl --fail --location --silent --show-error "${URL}" --output "${ARCHIVE}"
if [[ "$(sha256_file "${ARCHIVE}")" != "${EXPECTED_ARCHIVE_SHA256}" ]]; then
  echo "Gitleaks archive checksum mismatch" >&2
  exit 1
fi

tar -xzf "${ARCHIVE}" -C "${TEMP_DIR}" gitleaks
STAGED_OUTPUT="$(mktemp "${OUTPUT_DIR}/.gitleaks.XXXXXX")"
install -m 0755 "${TEMP_DIR}/gitleaks" "${STAGED_OUTPUT}"
if [[ "$(sha256_file "${STAGED_OUTPUT}")" != "${EXPECTED_BINARY_SHA256}" ]]; then
  echo "installed Gitleaks executable checksum mismatch" >&2
  exit 1
fi
if [[ "$("${STAGED_OUTPUT}" version)" != "${GITLEAKS_VERSION}" ]]; then
  echo "installed Gitleaks version does not match the pinned version" >&2
  exit 1
fi
ensure_real_directory "${OUTPUT_DIR}"
mv -f "${STAGED_OUTPUT}" "${OUTPUT}"
STAGED_OUTPUT=""
if [[ -L "${OUTPUT}" ]] || [[ "$(sha256_file "${OUTPUT}")" != "${EXPECTED_BINARY_SHA256}" ]]; then
  echo "installed Gitleaks executable integrity check failed" >&2
  exit 1
fi
