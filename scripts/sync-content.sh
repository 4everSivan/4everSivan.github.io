#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

export GOCACHE="${GOCACHE:-${ROOT_DIR}/.local/cache/go-build}"
export GOMODCACHE="${GOMODCACHE:-${ROOT_DIR}/.local/cache/go-mod}"
# shellcheck source=/dev/null
source config/versions.env
export GOTOOLCHAIN="go${GO_VERSION}"

./scripts/install-gitleaks.sh
mkdir -p .local/bin
go build -trimpath -o .local/bin/contentctl ./cmd/contentctl
./.local/bin/contentctl sync
./.local/bin/contentctl verify
