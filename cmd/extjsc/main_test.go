package main

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"extjs-compiler/internal/compiler"
)

func TestBuildJSONWritesAssetManifestToStdout(t *testing.T) {
	appDir := t.TempDir()
	for name, content := range map[string]string{
		"app.json":           `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"vendor.js"},{"path":"app.js","bundle":true}],"css":["site.css"],"resources":["resources"]}`,
		"app.js":             `Ext.application({name: 'Example', extend: 'Example.Entry'});`,
		"vendor.js":          `window.vendorReady = true;`,
		"site.css":           `.example { color: red; }`,
		"app/Entry.js":       `Ext.define('Example.Entry', {});`,
		"resources/logo.svg": `<svg xmlns="http://www.w3.org/2000/svg"/>`,
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
	runErr := run([]string{"build", "-app", appDir, "-out", filepath.Join(appDir, "out"), "-json", "-base-path", "/static/example/"})
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
	if manifest.AppName != "Example" || manifest.BasePath != "/static/example/" || manifest.Resources != "/static/example/resources/" {
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
}
