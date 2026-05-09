#!/usr/bin/env bash
set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
BACKEND_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
mkdir -p "$BACKEND_DIR/dist"
go build -buildmode=plugin -o "$BACKEND_DIR/dist/plugin.so" "$SCRIPT_DIR/plugin.go"
echo "built: $BACKEND_DIR/dist/plugin.so"
