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
	binary_name=${binary##*/}
	name="tualojs_${version}_${platform}_${arch}"
	package_dir="$release_dir/$name"

	mkdir -p "$package_dir"
	cp "$binary" "$package_dir/"
	cp README.md README.de.md LICENSE NOTICE "$package_dir/"
	tar -czf "$release_dir/$name.tar.gz" -C "$package_dir" "$binary_name" LICENSE NOTICE README.md README.de.md
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
	checksum_command=sha256sum
elif command -v shasum >/dev/null 2>&1; then
	checksum_command=shasum
else
	printf 'sha256sum or shasum is required to write release checksums.\n' >&2
	exit 1
fi

archive_sha() {
	if [ "$checksum_command" = sha256sum ]; then
		sha256sum "$1" | awk '{print $1}'
	else
		shasum -a 256 "$1" | awk '{print $1}'
	fi
}

formula_version=${version#v}
sed \
	-e "s/@VERSION@/$formula_version/g" \
	-e "s/@MACOS_ARM64_SHA256@/$(archive_sha "tualojs_${version}_macos_arm64.tar.gz")/g" \
	-e "s/@MACOS_AMD64_SHA256@/$(archive_sha "tualojs_${version}_macos_amd64.tar.gz")/g" \
	-e "s/@LINUX_ARM64_SHA256@/$(archive_sha "tualojs_${version}_linux_arm64.tar.gz")/g" \
	-e "s/@LINUX_AMD64_SHA256@/$(archive_sha "tualojs_${version}_linux_amd64.tar.gz")/g" \
	"$script_dir/tualojs.rb.in" > tualojs.rb

if [ "$checksum_command" = sha256sum ]; then
	set -- "tualojs_${version}_"*.tar.gz
	for package in "tualojs_${version}_"*.deb; do
		if [ -f "$package" ]; then
			set -- "$@" "$package"
		fi
	done
	sha256sum "$@" tualojs.rb > SHA256SUMS
else
	set -- "tualojs_${version}_"*.tar.gz
	for package in "tualojs_${version}_"*.deb; do
		if [ -f "$package" ]; then
			set -- "$@" "$package"
		fi
	done
	shasum -a 256 "$@" tualojs.rb > SHA256SUMS
fi
