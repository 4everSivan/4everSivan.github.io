#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
# shellcheck source=/dev/null
source "${ROOT_DIR}/config/versions.env"

ASSETS_DIR="${ROOT_DIR}/assets"
OUTPUT_DIR="${ASSETS_DIR}/vendor"

ensure_real_directory() {
  local directory="$1"
  if [[ -L "${directory}" ]] || { [[ -e "${directory}" ]] && [[ ! -d "${directory}" ]]; }; then
    echo "frontend asset directory is unsafe" >&2
    exit 1
  fi
  mkdir -p "${directory}"
  if [[ -L "${directory}" ]] || [[ ! -d "${directory}" ]]; then
    echo "frontend asset directory is unsafe" >&2
    exit 1
  fi
}

ensure_real_directory "${ASSETS_DIR}"
ensure_real_directory "${OUTPUT_DIR}"

sha256_file() {
  if command -v sha256sum >/dev/null 2>&1; then
    sha256sum "$1" | awk '{print $1}'
  else
    shasum -a 256 "$1" | awk '{print $1}'
  fi
}

install_asset() {
  local url="$1"
  local expected="$2"
  local output="$3"
  if [[ ! "${expected}" =~ ^[0-9a-f]{64}$ ]]; then
    echo "frontend asset checksum is invalid" >&2
    exit 1
  fi
  if [[ -L "${output}" ]] || { [[ -e "${output}" ]] && [[ ! -f "${output}" ]]; }; then
    echo "frontend asset output path is unsafe" >&2
    exit 1
  fi
  if [[ -f "${output}" ]] && [[ ! -L "${output}" ]] && [[ "$(sha256_file "${output}")" == "${expected}" ]]; then
    return
  fi
  local temporary
  temporary="$(mktemp "${OUTPUT_DIR}/.asset.XXXXXX")"
  curl --fail --location --silent --show-error "${url}" --output "${temporary}"
  if [[ "$(sha256_file "${temporary}")" != "${expected}" ]]; then
    rm -f "${temporary}"
    echo "frontend asset checksum mismatch" >&2
    exit 1
  fi
  chmod 0644 "${temporary}"
  ensure_real_directory "${OUTPUT_DIR}"
  mv -f "${temporary}" "${output}"
  if [[ -L "${output}" ]] || [[ "$(sha256_file "${output}")" != "${expected}" ]]; then
    echo "installed frontend asset integrity check failed" >&2
    exit 1
  fi
}

install_asset \
  "https://cdn.jsdelivr.net/npm/flexsearch@${FLEXSEARCH_VERSION}/dist/flexsearch.bundle.min.js" \
  "${FLEXSEARCH_SHA256}" \
  "${OUTPUT_DIR}/flexsearch.bundle.min.js"
install_asset \
  "https://cdn.jsdelivr.net/npm/mermaid@${MERMAID_VERSION}/dist/mermaid.min.js" \
  "${MERMAID_SHA256}" \
  "${OUTPUT_DIR}/mermaid.min.js"
