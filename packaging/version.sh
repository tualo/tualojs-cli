#!/bin/bash
version=$1
commit_message=$2
if [ -z "$version" ] || [ -z "$commit_message" ]; then
    echo "Usage: $0 <version> <commit_message>"
    exit 1
fi
VERSION=$version ./build.sh
VERSION=$version sh packaging/create-deb-packages.sh
VERSION=$version sh packaging/create-release-archives.sh
git tag -a v$version -m "Release version $version"
git push origin --tags
git commit -am "$commit_message"
git push origin HEAD
