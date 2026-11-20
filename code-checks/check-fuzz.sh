#!/bin/sh
set -eu

repo_root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
cd "$repo_root"

fuzz_time=${FUZZTIME:-30s}
for target in \
    FuzzParseExtMetadata \
    FuzzParseLoaderPaths \
    FuzzPreprocessSenchaForTarget \
    FuzzStripJSONComments
do
    printf 'Fuzzing %s for %s\n' "$target" "$fuzz_time"
    go test ./internal/compiler -run='^$' -fuzz="^${target}$" -fuzztime="$fuzz_time"
done
