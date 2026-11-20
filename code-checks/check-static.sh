#!/bin/sh
set -eu

if ! command -v staticcheck >/dev/null 2>&1; then
    printf 'staticcheck is required; see code-checks/README.md for installation.\n' >&2
    exit 1
fi

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"
staticcheck ./...
