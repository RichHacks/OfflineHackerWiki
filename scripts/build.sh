#!/usr/bin/env bash
# Cross-compiles the hackerwiki binary for Linux/macOS/Windows, amd64+arm64.
# Usage: scripts/build.sh [output-dir]
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")/.."

OUT_DIR="${1:-dist}"
BINARY_NAME="hackerwiki"

TARGETS=(
	"linux amd64"
	"linux arm64"
	"darwin amd64"
	"darwin arm64"
	"windows amd64"
	"windows arm64"
)

rm -rf "$OUT_DIR"
mkdir -p "$OUT_DIR"

for target in "${TARGETS[@]}"; do
	read -r os arch <<<"$target"

	ext=""
	if [ "$os" = "windows" ]; then
		ext=".exe"
	fi

	out="$OUT_DIR/${BINARY_NAME}-${os}-${arch}${ext}"
	echo "Building $out"
	GOOS="$os" GOARCH="$arch" CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o "$out" .
done

echo
echo "Built binaries:"
ls -lh "$OUT_DIR"
