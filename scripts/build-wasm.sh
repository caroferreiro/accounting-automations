#!/usr/bin/env bash
set -euo pipefail

echo "Building comparador.wasm..."
GOOS=js GOARCH=wasm go build -o web/comparador.wasm ./pkg/cmd/wasm/

echo "Copying wasm_exec.js..."
GOROOT=$(go env GOROOT)
if [ -f "$GOROOT/misc/wasm/wasm_exec.js" ]; then
    cp "$GOROOT/misc/wasm/wasm_exec.js" web/wasm_exec.js
elif [ -f "$GOROOT/lib/wasm/wasm_exec.js" ]; then
    cp "$GOROOT/lib/wasm/wasm_exec.js" web/wasm_exec.js
else
    echo "Error: wasm_exec.js not found in $GOROOT" && exit 1
fi

echo "Done."
echo "  web/comparador.wasm  $(du -sh web/comparador.wasm | cut -f1)"
echo "  web/wasm_exec.js     $(du -sh web/wasm_exec.js | cut -f1)"
