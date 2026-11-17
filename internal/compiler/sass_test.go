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

func TestNormalizeFashionDynamicLiteralAssignments(t *testing.T) {
	source := []byte(`$color: dynamic(#ddd);
$spacing: dynamic(12px);
$enabled: dynamic(true);
$derived: dynamic($color);
`)
	got, changed, reason := normalizeFashionDynamic(source)
	if reason != "" || !changed {
		t.Fatalf("normalizeFashionDynamic changed=%t, reason=%q", changed, reason)
	}
	if want := "$color: #ddd;\n$spacing: 12px;\n$enabled: true;\n$derived: $color;\n"; string(got) != want {
		t.Fatalf("normalized Sass = %q, want %q", got, want)
	}
	for _, unsupported := range [][]byte{
		[]byte(".widget { color: dynamic(#fff); }\n"),
	} {
		if _, _, reason := normalizeFashionDynamic(unsupported); reason == "" {
			t.Errorf("normalizeFashionDynamic(%q) should require Fashion", unsupported)
		}
	}
}

func TestResolvePackageSassVariablesOrdersForwardReferences(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"theme-child/sass/var/child.scss": `$accent: dynamic($base-color);`,
		"theme-base/sass/var/base.scss":   `$base-color: dynamic(#246);`,
	})

	sources, watchPaths, variables, warnings, err := resolvePackageSassVariables(root, []string{
		filepath.Join(root, "theme-child", "sass", "var"),
		filepath.Join(root, "theme-base", "sass", "var"),
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantSources := []string{
		filepath.Join(root, "theme-base", "sass", "var", "base.scss"),
		filepath.Join(root, "theme-child", "sass", "var", "child.scss"),
	}
	if !reflect.DeepEqual(sources, wantSources) {
		t.Fatalf("package Sass sources = %#v, want dependency order %#v", sources, wantSources)
	}
	if len(warnings) != 0 {
		t.Fatalf("package Sass warnings = %#v", warnings)
	}
	if !variables["base-color"] || !variables["accent"] {
		t.Fatalf("available Sass variables = %#v, want base-color and accent", variables)
	}
	wantWatchPaths := []string{
		filepath.Join(root, "theme-child", "sass", "var", "child.scss"),
		filepath.Join(root, "theme-base", "sass", "var", "base.scss"),
	}
	if !reflect.DeepEqual(watchPaths, wantWatchPaths) {
		t.Fatalf("package Sass watch paths = %#v, want %#v", watchPaths, wantWatchPaths)
	}
}

func TestDartSassCompilesDynamicPackageVariableReferences(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}

	root := makeFixture(t, map[string]string{
		"theme-child/sass/var/child.scss": `$accent: dynamic($base-color);`,
		"theme-base/sass/var/base.scss":   `$base-color: dynamic(#246);`,
		"app/Main.scss":                   `.main { color: $accent; }`,
	})
	variables, _, _, _, err := resolvePackageSassVariables(root, []string{
		filepath.Join(root, "theme-child", "sass", "var"),
		filepath.Join(root, "theme-base", "sass", "var"),
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	sources := append(variables, filepath.Join(root, "app", "Main.scss"))
	css, warnings, err := compileClassSass(root, sources, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Sass warnings = %#v", warnings)
	}
	if !strings.Contains(string(css), "color: #246") {
		t.Fatalf("compiled CSS = %s, want dynamic variable reference resolved", css)
	}
}

func TestBuildLoadsPackageSassEtcAndLocalImports(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}

	root := makeFixture(t, map[string]string{
		"app.json":                                     `{"name":"Example","namespace":"Example","toolkit":"classic","requires":["widget-lib"],"classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                               `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                             `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":               `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src","sass":{"etc":"${package.dir}/sass/etc/all.scss"}}}`,
		"sdk/packages/widget-lib/package.json":         `{"version":"1.0.0","sencha":{"name":"widget-lib","requires":["core"],"classpath":"${package.dir}/src","sass":{"src":"${package.dir}/sass/src"}}}`,
		"sdk/classic/classic/package.json":             `{"version":"8.0.0.43","sencha":{"name":"classic","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/packages/core/sass/etc/all.scss":          "$prefix: dynamic('x-');\n@import \"mixins\";\n",
		"sdk/packages/core/sass/etc/_mixins.scss":      `@mixin unused-package-mixin { color: red; }`,
		"sdk/packages/widget-lib/src/Widget.js":        `Ext.define('Lib.Widget', {});`,
		"sdk/packages/widget-lib/sass/src/Widget.scss": `.#{$prefix}widget { color: #246; }`,
		"app.js":       `Ext.application({name: 'Example'}); Ext.require('Lib.Widget');`,
		"app/Empty.js": "",
	})

	result, err := BuildWithToolkit(root, "out", "classic")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.StyleSheets) != 1 {
		t.Fatalf("generated stylesheets = %#v, want one package Sass stylesheet", result.StyleSheets)
	}
	css, err := os.ReadFile(result.StyleSheets[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), ".x-widget") || !strings.Contains(string(css), "color: #246") {
		t.Fatalf("compiled package Sass = %s", css)
	}
	for _, path := range []string{
		filepath.Join(root, "sdk", "packages", "core", "sass", "etc", "all.scss"),
		filepath.Join(root, "sdk", "packages", "core", "sass", "etc", "_mixins.scss"),
	} {
		found := false
		for _, watchPath := range result.WatchPaths {
			if watchPath == path {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("package Sass etc path %s is missing from watch paths", path)
		}
	}
}

func TestBuildLoadsFrameworkPackageClassSass(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}

	root := makeFixture(t, map[string]string{
		"app.json":                              `{"name":"Example","namespace":"Example","toolkit":"classic","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                        `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                      `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":        `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src","sass":{"etc":"${package.dir}/sass/etc/all.scss","src":"${package.dir}/sass/src"}}}`,
		"sdk/classic/classic/package.json":      `{"version":"8.0.0.43","sencha":{"name":"classic","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/packages/core/sass/etc/all.scss":   "$prefix: dynamic('x-');",
		"sdk/packages/core/src/Panel.js":        `Ext.define('Ext.Panel', {});`,
		"sdk/packages/core/sass/src/Panel.scss": `.#{$prefix}framework-panel { color: #246; }`,
		"app.js":                                `Ext.application({name: 'Example'});`,
		"app/Empty.js":                          "",
	})

	result, err := BuildWithToolkit(root, "out", "classic")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.StyleSheets) != 1 {
		t.Fatalf("generated stylesheets = %#v, want framework package Sass stylesheet", result.StyleSheets)
	}
	css, err := os.ReadFile(result.StyleSheets[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), ".x-framework-panel") || !strings.Contains(string(css), "color: #246") {
		t.Fatalf("compiled framework Sass = %s", css)
	}
	for _, path := range []string{
		filepath.Join(root, "sdk", "packages", "core", "sass", "etc", "all.scss"),
		filepath.Join(root, "sdk", "packages", "core", "sass", "src", "Panel.scss"),
	} {
		found := false
		for _, watchPath := range result.WatchPaths {
			if watchPath == path {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("framework Sass path %s is missing from watch paths", path)
		}
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
		"sass/etc/all.scss":    `$base-color: dynamic(#246);`,
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

func TestDartSassUsesPackageVariablesForPackageClassStyles(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}

	root := makeFixture(t, map[string]string{
		"app.json":                                     `{"name":"Example","namespace":"Example","toolkit":"classic","requires":["widget-lib"],"classpath":["app"],"sass":{"var":"sass/var"},"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                               `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                             `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":               `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/classic/classic/package.json":             `{"version":"8.0.0.43","sencha":{"name":"classic","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/packages/widget-lib/package.json":         `{"version":"1.0.0","sencha":{"name":"widget-lib","requires":["core"],"classpath":"${package.dir}/src","sass":{"var":"${package.dir}/sass/var","src":"${package.dir}/sass/src"}}}`,
		"sdk/packages/widget-lib/src/Widget.js":        `Ext.define('Lib.Widget', {});`,
		"sdk/packages/widget-lib/sass/var/colors.scss": `$widget-color: dynamic(#456);`,
		"sdk/packages/widget-lib/sass/src/Widget.scss": `.package-widget { color: $widget-color; }`,
		"sass/var/app.scss":                            `$widget-color: #789;`,
		"app.js":                                       `Ext.application({name: 'Example'}); Ext.require('Lib.Widget');`,
		"app/Empty.js":                                 "",
	})

	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	css, err := os.ReadFile(result.StyleSheets[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(css), ".package-widget") || !strings.Contains(string(css), "color: #789") {
		t.Fatalf("package Sass variable was not applied to package class CSS: %s", css)
	}
}
