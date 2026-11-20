# Code checks

Run checks from the repository root. All scripts are read-only apart from Go's normal test and build caches; fuzzing may save newly discovered corpus cases in Go's fuzz cache.

The same checks run automatically in GitHub Actions on pushes and pull requests. GitHub Actions installs Dart Sass for the Go test matrix. Tagging a version with `v*` runs the checks and, if they pass, publishes binary archives, Debian packages, a Homebrew formula, and SHA-256 checksums.

## Run the full suite

Install Go 1.23 or newer, Node.js with Dart Sass, and the optional analysis tools listed below, then run:

```sh
npm install --global sass@1
```

```sh
sh code-checks/check-all.sh
```

The full suite checks formatting, `go vet`, tests, the race detector, `staticcheck`, `govulncheck`, `gosec`, and Go fuzz targets. Fuzz duration defaults to 30 seconds per target; override it as needed:

```sh
FUZZTIME=2m sh code-checks/check-fuzz.sh
```

## Run individual checks

```sh
sh code-checks/check-go.sh       # gofmt, vet, unit/integration tests, race detector
sh code-checks/check-static.sh   # staticcheck
sh code-checks/check-security.sh # govulncheck and gosec
sh code-checks/check-fuzz.sh     # parser and preprocessor fuzz targets
```

## Install optional tools

These are standalone developer tools; they do not add dependencies to `go.mod`:

```sh
go install honnef.co/go/tools/cmd/staticcheck@latest
go install golang.org/x/vuln/cmd/govulncheck@latest
go install github.com/securego/gosec/v2/cmd/gosec@latest
```

Ensure `$(go env GOPATH)/bin` is on `PATH`. The gosec gate fails on high-severity, high-confidence findings; lower-confidence reports should still be reviewed. The configured Sass executable is an intentional exception: the user selects it explicitly, and tualojs invokes it without a shell. `govulncheck` reports known vulnerabilities only when affected code is reachable in the analyzed build; use a current patched Go release when running it.

## Fuzz targets

Fuzz tests exercise JavaScript class metadata, loader-path parsing, Sencha preprocessing, and JSON comment stripping with arbitrary and malformed inputs. They are intended to expose panics and parser robustness issues; they do not prove the absence of all security defects. Run `go test ./...` to execute their seed cases as ordinary tests.
