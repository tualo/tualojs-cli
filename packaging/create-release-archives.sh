#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
cd "$repo_root"

version=${VERSION:?Set VERSION to the release tag, for example v1.0.0}
release_dir=dist/release
mkdir -p "$release_dir"

create_archive() {
	platform=$1
	arch=$2
	binary=$3
	name="tualojs_${version}_${platform}_${arch}"
	package_dir="$release_dir/$name"

	mkdir -p "$package_dir"
	cp "$binary" "$package_dir/"
	cp README.md README.de.md LICENSE NOTICE "$package_dir/"
	tar -czf "$release_dir/$name.tar.gz" -C "$release_dir" "$name"
}

for platform in linux macos; do
	for arch in amd64 arm64; do
		create_archive "$platform" "$arch" "dist/$platform/$arch/tualojs"
	done
done

for arch in amd64 arm64; do
	create_archive windows "$arch" "dist/windows/$arch/tualojs.exe"
done

cd "$release_dir"
if command -v sha256sum >/dev/null 2>&1; then
	sha256sum tualojs_*.tar.gz > SHA256SUMS
elif command -v shasum >/dev/null 2>&1; then
	shasum -a 256 tualojs_*.tar.gz > SHA256SUMS
else
	printf 'sha256sum or shasum is required to write release checksums.\n' >&2
	exit 1
fi
