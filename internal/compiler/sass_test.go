package compiler

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestCompileClassSassReportsMissingDartSass(t *testing.T) {
	root := t.TempDir()
	source := filepath.Join(root, "Main.scss")
	if err := os.WriteFile(source, []byte(".main { border: 0px; }"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SASS", filepath.Join(root, "missing-sass"))
	_, _, err := compileClassSass(root, []string{source}, false)
	if err == nil || !strings.Contains(err.Error(), "Dart Sass executable") {
		t.Fatalf("compileClassSass error = %v, want actionable missing-executable error", err)
	}
}

func TestResolveSassImportWatchPaths(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app/Main.scss": `@use "theme/colors";
@forward "shared";
@import "theme.css";
@import url("https://cdn.example/theme.css");
`,
		"app/theme/_colors.scss":  `@use "palette";`,
		"app/theme/_palette.scss": `$palette: blue;`,
		"app/_shared.scss":        `$shared: true;`,
	})

	dependencies, err := resolveSassImportWatchPaths(root, []string{filepath.Join(root, "app", "Main.scss")})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		filepath.Join(root, "app", "theme", "_colors.scss"),
		filepath.Join(root, "app", "theme", "_palette.scss"),
		filepath.Join(root, "app", "_shared.scss"),
	}
	if !reflect.DeepEqual(dependencies, want) {
		t.Fatalf("Sass import watch paths = %#v, want %#v", dependencies, want)
	}
}

func TestDartSassCompilesConfiguredAndClassSourcesTogether(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}

	root := makeFixture(t, map[string]string{
		"app.json":             `{"name":"Example","namespace":"Example","toolkit":"classic","classpath":["app"],"sass":{"etc":"sass/etc/all.scss","var":"sass/var","src":"sass/src"},"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":               `Ext.application({name: 'Example'});`,
		"app/Main.js":          `Ext.define('Example.Main', {});`,
		"app/Main.scss":        `@import "palette"; .main { color: $main-color; }`,
		"app/_palette.scss":    `$main-color: #456;`,
		"sass/etc/all.scss":    `$base-color: #246;`,
		"sass/var/colors.scss": `$main-color: $base-color;`,
		"sass/src/base.scss":   `.configured { border: 0px; }`,
	})

	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.StyleSheets) != 1 || result.StyleSheets[0].URL != "app-sass.css" {
		t.Fatalf("generated stylesheets = %#v, want app-sass.css", result.StyleSheets)
	}
	css, err := os.ReadFile(result.StyleSheets[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{".configured", "border: 0px", ".main", "color: #456"} {
		if !strings.Contains(string(css), expected) {
			t.Errorf("compiled Sass CSS is missing %q: %s", expected, css)
		}
	}
}
