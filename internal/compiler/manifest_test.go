package compiler

import (
	"encoding/json"
	"path/filepath"
	"reflect"
	"testing"
)

func TestNewAssetManifestResolvesBasePathAndPreservesAssetOrder(t *testing.T) {
	outputDir := t.TempDir()
	result := BuildResult{
		AppName:    "Example",
		Toolkit:    "modern",
		Theme:      "theme-material",
		OutputPath: filepath.Join(outputDir, "app.js"),
		StyleSheets: []StyleSheet{
			{URL: "css/app.css"},
			{URL: "https://cdn.example/theme.css", Remote: true},
			{URL: "external/theme.css", Remote: true},
			{URL: "/shared/base.css", Remote: true},
		},
		Scripts: []ScriptAsset{
			{URL: "js/vendor.js"},
			{URL: "app.js"},
			{URL: "//cdn.example/extra.js", Remote: true},
		},
	}
	manifest := NewAssetManifest(result, "/static/example///")

	if manifest.BasePath != "/static/example/" || manifest.Resources != "/static/example/resources/" {
		t.Fatalf("manifest paths = base %q, resources %q", manifest.BasePath, manifest.Resources)
	}
	if manifest.OutputDir != outputDir {
		t.Fatalf("manifest output directory = %q, want %q", manifest.OutputDir, outputDir)
	}
	if got, want := manifest.CSS, []ManifestAsset{
		{URL: "/static/example/css/app.css", Path: "css/app.css", File: filepath.Join(outputDir, "css", "app.css")},
		{URL: "https://cdn.example/theme.css", Remote: true},
		{URL: "external/theme.css", Remote: true},
		{URL: "/shared/base.css", Remote: true},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("CSS assets = %#v, want %#v", got, want)
	}
	if got, want := manifest.JS, []ManifestAsset{
		{URL: "/static/example/js/vendor.js", Path: "js/vendor.js", File: filepath.Join(outputDir, "js", "vendor.js")},
		{URL: "/static/example/app.js", Path: "app.js", File: filepath.Join(outputDir, "app.js")},
		{URL: "//cdn.example/extra.js", Remote: true},
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("JS assets = %#v, want %#v", got, want)
	}
	if len(manifest.Files) != 3 {
		t.Fatalf("manifest files = %#v, want only local stylesheet and script outputs", manifest.Files)
	}

	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"appName", "basePath", "css", "js", "resources"} {
		if _, exists := decoded[key]; !exists {
			t.Errorf("JSON manifest has no %q field: %s", key, data)
		}
	}
}

func TestNewAssetManifestWithEmptyBasePathKeepsRelativeURLs(t *testing.T) {
	manifest := NewAssetManifest(BuildResult{
		Scripts: []ScriptAsset{{URL: "app.js"}},
	}, "")
	if manifest.BasePath != "" || manifest.Resources != "resources/" || len(manifest.CSS) != 0 || len(manifest.JS) != 1 || manifest.JS[0].URL != "app.js" {
		t.Fatalf("manifest with empty base path = %#v", manifest)
	}
}
