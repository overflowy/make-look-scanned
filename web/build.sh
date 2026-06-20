#!/usr/bin/env bash
# Builds the wasm module and copies Go's wasm_exec.js next to the web shell.
set -euo pipefail
cd "$(dirname "$0")/.."

GOOS=js GOARCH=wasm go build -o web/main.wasm ./cmd/wasm
cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" web/wasm_exec.js

echo "built web/main.wasm ($(wc -c < web/main.wasm) bytes) + web/wasm_exec.js"
echo "serve with:  (cd web && python3 -m http.server 8080)  then open http://localhost:8080"
