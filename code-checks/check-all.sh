#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)

sh "$script_dir/check-go.sh"
sh "$script_dir/check-static.sh"
sh "$script_dir/check-security.sh"
sh "$script_dir/check-fuzz.sh"
