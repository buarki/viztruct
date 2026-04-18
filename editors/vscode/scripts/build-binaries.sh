#!/usr/bin/env bash
# Cross-compile viztruct for every OS/arch combo the VSCode extension bundles.
# Called from `npm run build:binaries` (and transitively from `vsce package`
# via the vscode:prepublish hook). Outputs land in editors/vscode/bin/.
#
# Requires: Go toolchain. No CGO; viztruct is pure Go.
set -euo pipefail

SCRIPT_DIR="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")" &> /dev/null && pwd)"
EXT_DIR="$(cd -- "${SCRIPT_DIR}/.." &> /dev/null && pwd)"
REPO_ROOT="$(cd -- "${EXT_DIR}/../.." &> /dev/null && pwd)"
OUT_DIR="${EXT_DIR}/bin"

mkdir -p "${OUT_DIR}"
rm -f "${OUT_DIR}"/viztruct-*

# Each entry: "<GOOS> <GOARCH>". Windows binaries get a .exe suffix automatically.
TARGETS=(
  "darwin amd64"
  "darwin arm64"
  "linux amd64"
  "linux arm64"
  "windows amd64"
  "windows arm64"
)

cd "${REPO_ROOT}"

for target in "${TARGETS[@]}"; do
  read -r GOOS GOARCH <<< "${target}"
  ext=""
  if [[ "${GOOS}" == "windows" ]]; then
    ext=".exe"
  fi
  out="${OUT_DIR}/viztruct-${GOOS}-${GOARCH}${ext}"
  echo "  building ${GOOS}/${GOARCH} → $(basename "${out}")"
  GOOS="${GOOS}" GOARCH="${GOARCH}" CGO_ENABLED=0 \
    go build -trimpath -ldflags='-s -w' -o "${out}" ./cmd/viztruct
done

echo
echo "Built binaries:"
ls -lh "${OUT_DIR}"
