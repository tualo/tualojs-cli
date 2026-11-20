package main

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"tualojs/internal/compiler"
)

func TestVersionCommand(t *testing.T) {
	originalVersion := version
	version = "v1.2.3"
	t.Cleanup(func() { version = originalVersion })

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStdout := os.Stdout
	os.Stdout = writer
	runErr := run([]string{"version"})
	_ = writer.Close()
	os.Stdout = originalStdout
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !bytes.HasPrefix(output, []byte("tualojs v1.2.3 (")) || !bytes.HasSuffix(output, []byte(")\n")) {
		t.Fatalf("version output = %q", output)
	}
}

func TestBuildJSONWritesAssetManifestToStdout(t *testing.T) {
	appDir := t.TempDir()
	for name, content := range map[string]string{
		"app.json":                          `{"name":"Example","namespace":"Example","toolkit":"modern","classpath":["app"],"js":[{"path":"vendor.js"},{"path":"app.js","bundle":true}],"css":["site.css"],"resources":["resources"]}`,
		"workspace.json":                    `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                  `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":    `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/packages/core/src/Base.js":     `Ext.define('Ext.Base', {});`,
		"sdk/modern/modern/package.json":    `{"version":"8.0.0.43","sencha":{"name":"modern","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/modern/modern/src/Panel.js":    `Ext.define('Ext.Panel', { extend: 'Ext.Base' });`,
		"sdk/build/ext-modern-all-debug.js": `window.Ext = window.Ext || {}; window.frameworkIncluded = true;`,
		"app.js":                            `Ext.application({name: 'Example', extend: 'Example.Entry'});`,
		"vendor.js":                         `window.vendorReady = true;`,
		"site.css":                          `.example { color: red; }`,
		"app/Entry.js":                      `Ext.define('Example.Entry', {});`,
		"resources/logo.svg":                `<svg xmlns="http://www.w3.org/2000/svg"/>`,
	} {
		path := filepath.Join(appDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	originalStdout := os.Stdout
	os.Stdout = writer
	defer func() { os.Stdout = originalStdout }()
	runErr := run([]string{"build", "-app", appDir, "-out", filepath.Join(appDir, "out"), "-include-extjs", "-json", "-base-path", "/static/example/"})
	_ = writer.Close()
	os.Stdout = originalStdout
	output, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if runErr != nil {
		t.Fatal(runErr)
	}
	if readErr != nil {
		t.Fatal(readErr)
	}

	var manifest compiler.AssetManifest
	if err := json.Unmarshal(output, &manifest); err != nil {
		t.Fatalf("build output is not JSON: %v\n%s", err, output)
	}
	if manifest.AppName != "Example" || manifest.BasePath != "/static/example/" || manifest.Resources != "/static/example/resources/" || !manifest.FrameworkIncluded {
		t.Fatalf("manifest metadata = %#v", manifest)
	}
	if got, want := manifest.CSS, []compiler.ManifestAsset{{
		URL:  "/static/example/site.css",
		Path: "site.css",
		File: filepath.Join(appDir, "out", "site.css"),
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest CSS = %#v, want %#v", got, want)
	}
	if got, want := manifest.JS, []compiler.ManifestAsset{
		{URL: "/static/example/js/vendor.js", Path: "js/vendor.js", File: filepath.Join(appDir, "out", "js", "vendor.js")},
		{URL: "/static/example/app.js", Path: "app.js", File: filepath.Join(appDir, "out", "app.js")},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest JS = %#v, want %#v", got, want)
	}
	if manifest.OutputDir != filepath.Join(appDir, "out") {
		t.Fatalf("manifest outputDir = %q", manifest.OutputDir)
	}
	if got, want := manifest.ResourceFiles, []compiler.ManifestFile{{
		Kind: "resource",
		Path: "resources/logo.svg",
		File: filepath.Join(appDir, "out", "resources", "logo.svg"),
	}}; !reflect.DeepEqual(got, want) {
		t.Fatalf("manifest resource files = %#v, want %#v", got, want)
	}
	if len(manifest.Files) != 4 {
		t.Fatalf("manifest file list = %#v, want JS, CSS, and resource outputs", manifest.Files)
	}
	bundle, err := os.ReadFile(filepath.Join(appDir, "out", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(bundle), "\n// Source: framework/ext-modern-all-debug.js\nwindow.Ext = window.Ext || {}; window.frameworkIncluded = true;") {
		t.Fatalf("CLI did not prepend the Ext JS runtime to app.js: %s", bundle)
	}
}
