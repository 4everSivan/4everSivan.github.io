#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

if ! git diff --quiet --; then
  echo "工作树与 Git 暂存区不一致; 请先完整暂存后再验证" >&2
  exit 1
fi
UNTRACKED_FILES="$(git ls-files --others --exclude-standard)"
if [[ -n "${UNTRACKED_FILES}" ]]; then
  echo "存在未忽略且未暂存的文件; 请先完整暂存后再验证" >&2
  exit 1
fi

# shellcheck source=/dev/null
source config/versions.env
export GOCACHE="${GOCACHE:-${ROOT_DIR}/.local/cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-${ROOT_DIR}/.local/cache/go-mod}"
export GOTOOLCHAIN="go${GO_VERSION}"

./scripts/install-gitleaks.sh
./scripts/install-frontend-assets.sh
if [[ "$(go env GOVERSION)" != "go${GO_VERSION}" ]]; then
  echo "本地 Go 工具链与 config/versions.env 不一致" >&2
  exit 1
fi
if ! hugo version | grep -F "v${HUGO_VERSION}" >/dev/null; then
  echo "本地 Hugo 版本与 config/versions.env 不一致" >&2
  exit 1
fi
if ! grep -F "github.com/imfing/hextra ${HEXTRA_VERSION}" go.mod >/dev/null; then
  echo "go.mod 中的 Hextra 版本与 config/versions.env 不一致" >&2
  exit 1
fi

go mod verify
go test -race -count=1 ./...
go vet ./...

mkdir -p .local/bin
go build -trimpath -o .local/bin/contentctl ./cmd/contentctl
go build -trimpath -o .local/bin/linkcheck ./cmd/linkcheck

./.local/bin/contentctl verify
TRACKED_TREE="$(mktemp -d "${ROOT_DIR}/.local/tracked-tree.XXXXXX")"
trap 'rm -rf "${TRACKED_TREE}"' EXIT
git checkout-index --all --prefix="${TRACKED_TREE}/"
./.local/bin/gitleaks dir \
  --config .gitleaks.toml \
  --no-banner \
  --no-color \
  --redact=100 \
  --exit-code=1 \
  "${TRACKED_TREE}"

hugo mod verify
mkdir -p .local/cache/hugo
hugo --gc --minify --cleanDestinationDir \
  --cacheDir "${ROOT_DIR}/.local/cache/hugo" \
  --baseURL "https://4everSivan.github.io/"
./.local/bin/linkcheck public
