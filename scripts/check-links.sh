#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "${ROOT_DIR}"

export GOCACHE="${GOCACHE:-${ROOT_DIR}/.local/cache/go-build}"
mkdir -p .local/bin
go build -trimpath -o .local/bin/linkcheck ./cmd/linkcheck
./.local/bin/linkcheck "${1:-public}"
