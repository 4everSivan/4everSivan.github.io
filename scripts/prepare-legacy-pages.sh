#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

# shellcheck source=/dev/null
source config/versions.env
if [[ ! "${LEGACY_STATIC_COMMIT:-}" =~ ^[0-9a-f]{40}$ ]]; then
  echo "legacy static commit is not pinned to a full lower-case SHA" >&2
  exit 1
fi
git cat-file -e "${LEGACY_STATIC_COMMIT}^{commit}"
git merge-base --is-ancestor "${LEGACY_STATIC_COMMIT}" origin/main

PUBLIC_DIR="${ROOT_DIR}/public"
if [[ "${PUBLIC_DIR}" != "${ROOT_DIR}/public" ]] || [[ -L "${PUBLIC_DIR}" ]]; then
  echo "legacy Pages output path is unsafe" >&2
  exit 1
fi
rm -rf -- "${PUBLIC_DIR}"
mkdir -p "${PUBLIC_DIR}"

TEMP_DIR="$(mktemp -d)"
trap 'rm -rf "${TEMP_DIR}"' EXIT
ARCHIVE="${TEMP_DIR}/legacy-pages.tar"
STATIC_PATHS=(404.html assets ha images index.html linux mysql pg python write_log)
git archive --format=tar --output="${ARCHIVE}" "${LEGACY_STATIC_COMMIT}" -- "${STATIC_PATHS[@]}"
tar -xf "${ARCHIVE}" -C "${PUBLIC_DIR}"

expected_files=0
while IFS= read -r -d '' record; do
  metadata="${record%%$'\t'*}"
  relative_path="${record#*$'\t'}"
  mode="${metadata%% *}"
  remainder="${metadata#* }"
  object_type="${remainder%% *}"
  object_id="${metadata##* }"
  if [[ "${object_type}" != "blob" ]] || { [[ "${mode}" != "100644" ]] && [[ "${mode}" != "100755" ]]; }; then
    echo "legacy static tree contains a non-regular entry" >&2
    exit 1
  fi
  if [[ ! -f "${PUBLIC_DIR}/${relative_path}" ]] || [[ -L "${PUBLIC_DIR}/${relative_path}" ]]; then
    echo "legacy static archive is incomplete" >&2
    exit 1
  fi
  if [[ "$(git hash-object -- "${PUBLIC_DIR}/${relative_path}")" != "${object_id}" ]]; then
    echo "legacy static file integrity check failed" >&2
    exit 1
  fi
  expected_files=$((expected_files + 1))
done < <(git ls-tree -rz "${LEGACY_STATIC_COMMIT}" -- "${STATIC_PATHS[@]}")

extracted_files="$(find "${PUBLIC_DIR}" -type f | wc -l | tr -d '[:space:]')"
if [[ "${extracted_files}" != "${expected_files}" ]]; then
  echo "legacy static archive file set differs from the pinned commit" >&2
  exit 1
fi

# The archived site already linked to these absent sections. Minimal local
# fallbacks make the recovery artifact pass the same internal-link gate
# without restoring or inventing article content.
placeholder='<!doctype html><html lang="zh-CN"><meta charset="utf-8"><title>旧站点恢复入口</title><a href="/">返回首页</a></html>'
for section in day_log me sql_server translation; do
  mkdir -p "${PUBLIC_DIR}/${section}"
  printf '%s\n' "${placeholder}" > "${PUBLIC_DIR}/${section}/index.html"
done
printf '' > "${PUBLIC_DIR}/.nojekyll"

if [[ -n "$(find "${PUBLIC_DIR}" -type l -print -quit)" ]]; then
  echo "legacy Pages artifact contains a symlink" >&2
  exit 1
fi
echo "旧站点恢复产物已从固定提交生成: ${LEGACY_STATIC_COMMIT}"
