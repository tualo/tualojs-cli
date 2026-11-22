#!/bin/sh
set -eu

script_dir=$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)
repo_root=$(CDPATH= cd -- "$script_dir/.." && pwd)
cd "$repo_root"

usage() {
	printf 'Usage: %s <version> [commit-or-tag-message]\n' "$0" >&2
	printf 'Example: %s 1.0.3 "Release 1.0.3"\n' "$0" >&2
}

if [ "$#" -lt 1 ] || [ "$#" -gt 2 ]; then
	usage
	exit 2
fi

version=${1#v}
message=${2:-}
if ! printf '%s\n' "$version" | grep -Eq '^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(-[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?(\+[0-9A-Za-z]+([.-][0-9A-Za-z]+)*)?$'; then
	printf 'Invalid version %s; use semantic versioning such as 1.0.3 or v1.0.3.\n' "$1" >&2
	exit 2
fi
tag="v$version"
tag_message=${message:-"Release $tag"}

branch=$(git symbolic-ref --quiet --short HEAD) || {
	printf 'A release must be created from a named branch, not detached HEAD.\n' >&2
	exit 1
}
git remote get-url origin >/dev/null 2>&1 || {
	printf 'Git remote "origin" is required to publish a release.\n' >&2
	exit 1
}

git fetch origin
remote_branch="refs/remotes/origin/$branch"
if ! git show-ref --verify --quiet "$remote_branch"; then
	printf 'Remote branch origin/%s does not exist; publish the branch before releasing.\n' "$branch" >&2
	exit 1
fi
behind=$(git rev-list --count "HEAD..$remote_branch")
if [ "$behind" -ne 0 ]; then
	printf 'Branch %s is behind origin/%s by %s commit(s); update it before releasing.\n' "$branch" "$branch" "$behind" >&2
	exit 1
fi

remote_tag=$(git ls-remote --tags --refs origin "refs/tags/$tag")
if [ -n "$remote_tag" ]; then
	printf 'Release tag %s already exists on origin.\n' "$tag" >&2
	exit 1
fi

tag_exists=0
head_commit=$(git rev-parse HEAD)
if git show-ref --verify --quiet "refs/tags/$tag"; then
	tag_commit=$(git rev-parse "$tag^{commit}")
	if [ "$tag_commit" != "$head_commit" ]; then
		printf 'Local tag %s already points to a different commit.\n' "$tag" >&2
		exit 1
	fi
	tag_exists=1
fi

untracked=$(git ls-files --others --exclude-standard)
if [ -n "$untracked" ]; then
	printf 'Untracked files are present; stage intended release files or remove them before releasing:\n%s\n' "$untracked" >&2
	exit 1
fi

tracked_changes=0
if ! git diff --quiet || ! git diff --cached --quiet; then
	tracked_changes=1
fi
if [ "$tag_exists" -eq 1 ] && [ "$tracked_changes" -eq 1 ]; then
	printf 'Local tag %s already exists; commit or discard working changes before retrying.\n' "$tag" >&2
	exit 1
fi
if [ "$tracked_changes" -eq 1 ] && [ -z "$message" ]; then
	printf 'Tracked changes are present; provide a commit message as the second argument.\n' >&2
	usage
	exit 2
fi

printf 'Running Go tests and vet checks...\n'
go test ./...
go vet ./...

printf 'Building release binaries for %s...\n' "$tag"
VERSION="$tag" ./build.sh

if command -v dpkg-deb >/dev/null 2>&1; then
	VERSION="$tag" sh packaging/create-deb-packages.sh
else
	printf 'dpkg-deb is unavailable; skipping local Debian packages (the tagged GitHub workflow will create them).\n' >&2
fi
VERSION="$tag" sh packaging/create-release-archives.sh

if [ "$tracked_changes" -eq 1 ]; then
	git add -u
	git commit -m "$message"
fi

git push origin "HEAD:refs/heads/$branch"
if [ "$tag_exists" -eq 0 ]; then
	git tag -a "$tag" -m "$tag_message"
fi
git push origin "refs/tags/$tag"

printf 'Published tag %s. The GitHub release workflow will build and publish its release assets.\n' "$tag"
