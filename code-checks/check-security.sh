#!/bin/sh
set -eu

for tool in govulncheck gosec; do
    if ! command -v "$tool" >/dev/null 2>&1; then
        printf '%s is required; see code-checks/README.md for installation.\n' "$tool" >&2
        exit 1
    fi
done

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

govulncheck ./...
gosec -severity high -confidence high -nosec-require-justification -nosec-require-rules ./...
