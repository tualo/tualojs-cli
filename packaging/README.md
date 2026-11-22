# Release packages

The tag release workflow publishes:

- Linux, macOS, and Windows binary archives for amd64 and arm64.
- Debian `.deb` packages for Linux amd64 and arm64.
- A version-specific Homebrew formula (`tualojs.rb`) with checksums for the macOS and Linux archives.
- `SHA256SUMS` covering the release archives, Debian packages, and formula.

## Publish a release

From the repository root, run:

```sh
./packaging/version.sh 1.0.3 "Release 1.0.3"
```

The script accepts either `1.0.3` or `v1.0.3`, runs `go test ./...` and `go vet ./...`, builds all six platform binaries with that version embedded, and creates local release archives. If tracked source changes are present, the supplied message is used to commit them before pushing the branch and annotated `v1.0.3` tag. With a clean tree, the message is optional and is used for the tag. The tag triggers the GitHub Actions workflow, which builds and publishes the official release assets.

Untracked files must be staged or removed before release. The script checks that the branch is not behind `origin` and refuses to reuse an already published version tag. On systems without `dpkg-deb`, local Debian packages are skipped; the tagged GitHub workflow creates them on Ubuntu.

To create the packages locally, first run `VERSION=v1.0.0 ./build.sh`, then:

```sh
VERSION=v1.0.0 sh packaging/create-deb-packages.sh
VERSION=v1.0.0 sh packaging/create-release-archives.sh
```

The Debian package uses `DEBIAN_MAINTAINER` for its `Maintainer` field. The release workflow defaults to `tualojs contributors`; set a repository Actions variable named `DEBIAN_MAINTAINER` to the project's real contact before publishing packages to a Debian repository.

The generated Homebrew formula is included in the GitHub Release and can be installed directly from its release URL. A shorter `brew install owner/tap/tualojs` command requires a separate Homebrew tap repository and is not configured here.

Only the tualojs binary and its project documents are packaged. Ext JS SDK files and Dart Sass are not included. A user may choose to bundle licensed SDK files into their own application separately; those files retain their own licensing terms.
