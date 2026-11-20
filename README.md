# tualojs

[English](README.md) · [Deutsch](README.de.md)

**tualojs is an independent Go-based compiler for Ext JS applications.** It builds JavaScript bundles and related application assets from an Ext JS project, with compatibility implemented incrementally against documented Sencha Cmd behavior.

> **Compatibility scope:** The current target is Ext JS 8.0.0 (SDK compatibility version `8.0.0`). tualojs is not a complete replacement for Sencha Cmd; supported behavior and known limitations are documented in [`docs/`](docs/).

## Features

- Builds Ext JS **Classic** and **Modern** applications.
- Resolves application classes and dependencies, including `requires`, `uses`, overrides, aliases, and xtype references.
- Reads `app.json`, `workspace.json`, Ext JS SDK manifests, and supported SDK packages.
- Processes JavaScript, selected Sencha preprocessor constructs, stylesheets, Sass, and static resources.
- Supports development and production modes, source maps, optional JavaScript minification, and a JSON asset manifest.
- Provides a local watch and preview server.
- Can optionally prepend the SDK's Ext JS runtime to the generated `app.js` bundle.
- Does not require Sencha Cmd at runtime and has no external Go module dependencies.

## Requirements

- **Go 1.23 or newer** to build from source.
- An Ext JS 8 SDK configured in the application workspace when SDK/package resolution or the Ext JS runtime is needed. The SDK must report compatibility version `8.0.0`.
- **Dart Sass** available as `sass` on `PATH` when the application build contains Sass sources. Set the `SASS` environment variable to use a different executable path.

The compiler does not include or grant a license to Ext JS, Sencha Cmd, SDK packages, or other Sencha assets. See [License and third-party software](#license-and-third-party-software).

## Build tualojs

Build a binary for the current platform:

```sh
go build -o tualojs ./cmd/tualojs
```

To build binaries for Linux, Windows, and macOS on amd64 and arm64:

```sh
./build.sh
```

The binaries are written to `dist/`. To run the test suite and static checks:

```sh
go test ./...
go vet ./...
```

## Configure an Ext JS SDK

For SDK and package resolution, the application's `workspace.json` must point to a locally available Ext JS SDK. For example, if the SDK is next to the application directory:

```json
{
  "frameworks": {
    "ext": {
      "path": "../ext-8.0.0",
      "version": "8.0.0.43"
    }
  }
}
```

The SDK's `package.json` must identify the framework as `ext` and declare `compatVersion` `8.0.0`. The SDK is supplied and licensed separately; it is not bundled with tualojs.

## Usage

Build an application from its project directory or pass its path with `-app`:

```sh
./tualojs build \
  -app ./my-ext-app \
  -out ./my-ext-app/build-tualojs \
  -toolkit modern \
  -mode production \
  -source-map \
  -minify
```

If the toolkit is already defined in `app.json`, `-toolkit` can be omitted. Use `classic` or `modern` for universal applications. The build writes `app.js` and any generated or copied assets to the output directory.

Start a local watch and preview server during development:

```sh
./tualojs watch \
  -app ./my-ext-app \
  -toolkit modern \
  -addr 127.0.0.1:1841
```

The server prints the local preview URL. It watches project inputs, rebuilds on changes, and refreshes the preview after a successful build.

### Useful options

| Option | Command | Description |
| --- | --- | --- |
| `-app <directory>` | `build`, `watch` | Application directory containing `app.json` (default: `.`). |
| `-out <directory>` | `build`, `watch` | Build output directory (default: `build-go`). |
| `-toolkit <toolkit>` | `build`, `watch` | Select `classic` or `modern`, including for universal apps. |
| `-platform <platform>` | `build`, `watch` | Select `desktop`, `phone`, or `tablet` Sencha conditions. |
| `-mode <mode>` | `build`, `watch` | Select `development` or `production` (default: `development`). |
| `-source-map` | `build`, `watch` | Write an `app.js.map` source map. |
| `-minify` | `build`, `watch` | Remove JavaScript comments and unnecessary whitespace. |
| `-normalize-let-const` | `build`, `watch` | Lower supported function-scope `let`/`const` declarations to `var`. |
| `-upgrade-es5` | `build`, `watch` | Upgrade supported function-local ES5 `var` declarations to `const`/`let`. |
| `-json` | `build` | Write an asset manifest to standard output. |
| `-base-path <url-prefix>` | `build` | Set the public URL prefix in the JSON manifest. |
| `-include-extjs` | `build` | Prepend the resolved Ext JS runtime to `app.js`. Requires an SDK and toolkit. |
| `-addr <address>` | `watch` | Preview server address (default: `127.0.0.1:1841`). |
| `-interval <duration>` | `watch` | Project polling interval (default: `500ms`). |

Run `./tualojs --help`, `./tualojs build --help`, or `./tualojs watch --help` for the full command-line usage.

### JSON asset manifest

The manifest can be used by a deployment tool to locate generated CSS, JavaScript, and resources:

```sh
./tualojs build \
  -app ./my-ext-app \
  -out ./my-ext-app/build-tualojs \
  -toolkit modern \
  -json \
  -base-path /assets/my-ext-app/
```

The JSON is written to standard output; build warnings are written to standard error. See [`docs/43-json-asset-manifest.md`](docs/43-json-asset-manifest.md) for the manifest fields.

## Compatibility and limitations

tualojs implements a documented subset of Ext JS and Sencha Cmd behavior. In particular:

- Dynamic JavaScript class names and other runtime-computed references cannot generally be resolved at build time.
- Sencha Fashion is only partially supported. Sass compatibility helpers cover documented cases, not the complete Fashion runtime or theme system.
- JavaScript minification removes comments and unnecessary whitespace; it does not rename identifiers or perform the full optimizations of Sencha Cmd's production toolchain.
- Other unsupported cases and known differences are recorded in the relevant [`docs/`](docs/) pages.

Use the documented scope to determine whether tualojs covers the build features required by your application.

## Documentation

- [`goals.md`](goals.md) — project goals, implemented features, and current scope.
- [`docs/00-roadmap.md`](docs/00-roadmap.md) — development roadmap and architecture guidelines.
- [`docs/`](docs/) — per-feature behavior, examples, tests, and limitations.

## License and third-party software

tualojs source code is licensed under the [Apache License 2.0](LICENSE). The license applies to tualojs only and **does not grant any rights to Sencha Ext JS, Sencha Cmd, or any Sencha SDK, package, or asset**. Those are governed by their respective license terms. Builds may include or copy materials from an application's SDK; users are responsible for observing the terms applicable to those materials. See [`NOTICE`](NOTICE).
