#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
cd "$repo_root"

version=${VERSION:?Set VERSION to the release tag, for example v1.0.0}
deb_version=${version#v}
deb_version=$(printf '%s' "$deb_version" | sed 's/-/~/')
maintainer=${DEBIAN_MAINTAINER:-tualojs contributors}
release_dir=dist/release

if ! command -v dpkg-deb >/dev/null 2>&1; then
	printf 'dpkg-deb is required to create Debian packages.\n' >&2
	exit 1
fi

mkdir -p "$release_dir"

for arch in amd64 arm64; do
	root=$(mktemp -d)
	trap 'rm -rf "$root"' EXIT HUP INT TERM
	mkdir -p "$root/usr/bin" "$root/usr/share/doc/tualojs" "$root/DEBIAN"
	install -m 0755 "dist/linux/$arch/tualojs" "$root/usr/bin/tualojs"
	install -m 0644 LICENSE "$root/usr/share/doc/tualojs/copyright"
	install -m 0644 NOTICE "$root/usr/share/doc/tualojs/NOTICE"
	install -m 0644 README.md "$root/usr/share/doc/tualojs/README.md"
	install -m 0644 README.de.md "$root/usr/share/doc/tualojs/README.de.md"
	printf 'Package: tualojs\nVersion: %s\nSection: devel\nPriority: optional\nArchitecture: %s\nMaintainer: %s\nHomepage: https://github.com/tualo/tualojs-cli\nDescription: Go-based compiler for Ext JS applications\n Builds JavaScript bundles and application assets from Ext JS projects.\n' \
		"$deb_version" "$arch" "$maintainer" > "$root/DEBIAN/control"
	dpkg-deb --build --root-owner-group "$root" "$release_dir/tualojs_${version}_linux_${arch}.deb"
	rm -rf "$root"
	trap - EXIT HUP INT TERM
done
