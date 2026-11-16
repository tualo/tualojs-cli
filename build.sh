#!/bin/sh
set -eu

if [ "$#" -ne 0 ]; then
    printf 'Usage: %s\nBuilds Linux, Windows and macOS binaries for amd64 and arm64 in dist/.\n' "$0" >&2
    exit 1
fi

if ! command -v go >/dev/null 2>&1; then
    printf 'Error: Go 1.22 or newer must be installed and available in PATH.\n' >&2
    exit 1
fi

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
cd "$script_dir"

mkdir -p dist

for os in linux windows darwin; do
    for arch in amd64 arm64; do
        platform=$os
        extension=
        case "$os" in
            darwin) platform=macos ;;
            windows) extension=.exe ;;
        esac

        output="dist/extjscompiler-${platform}-${arch}${extension}"
        printf 'Building %s/%s -> %s\n' "$os" "$arch" "$output"
        CGO_ENABLED=0 GOOS="$os" GOARCH="$arch" \
            go build -mod=readonly -trimpath -o "$output" ./cmd/extjsc
    done
done

printf 'Builds completed: %s/dist\n' "$script_dir"
