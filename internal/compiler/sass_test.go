package compiler

import (
	"bytes"
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

func TestThemeBaseFunctionSourcesQualifyAsSassHelpers(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	appRoot, err := filepath.Abs(filepath.Join("..", "..", "testdata", "ext-build", "default", "build", "production", "Tualo"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(appRoot, "ext", "modern", "theme-base", "sass", "etc", "functions.scss")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !sassSourceIsSassHelper(data) {
		t.Fatal("theme-base Sass functions should be preloaded as helpers")
	}
	stripped := stripFashionThemeBackgroundImageFunction(data)
	if !bytes.Contains(stripped, []byte("@function vertical")) {
		t.Fatal("removing theme-background-image should preserve the remaining theme functions")
	}
	entry := filepath.Join(appRoot, "ext", "modern", "theme-base", "sass", "etc", "all.scss")
	helperRoots := packageSassEtcHelperRoots(filepath.Join(appRoot, "ext"), []string{entry})
	helpers, err := resolveThemeSassHelperSources(appRoot, []string{entry}, helperRoots, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, helper := range helpers {
		if helper == path {
			calcPath := filepath.Join(filepath.Dir(path), "functions", "calc-math.scss")
			calcData, err := os.ReadFile(calcPath)
			if err != nil {
				t.Fatal(err)
			}
			fixture := makeFixture(t, map[string]string{
				"sdk/sass/etc/functions.scss":           string(data),
				"sdk/sass/etc/functions/calc-math.scss": string(calcData),
				"app/Theme.scss":                        `.theme { margin-top: vertical(4px 2px); }`,
			})
			functions := filepath.Join(fixture, "sdk", "sass", "etc", "functions.scss")
			style := filepath.Join(fixture, "app", "Theme.scss")
			css, warnings, err := compileClassSassWithInlineImports(fixture, []string{functions, style}, false, map[string]bool{functions: true}, nil, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if len(warnings) != 0 || !strings.Contains(string(css), "margin-top: 8px") {
				t.Fatalf("compiled theme function CSS = %s, warnings = %#v", css, warnings)
			}
			return
		}
	}
	t.Fatalf("theme-base functions source %s was not discovered in %#v", path, helpers)
}

func TestNormalizeFashionDynamicSassExpressions(t *testing.T) {
	source := []byte(`$color: dynamic(#ddd);
$spacing: dynamic(12px);
$enabled: dynamic(true);
$derived: dynamic($color);
$mixed: dynamic(mix(#fff, $color, 50%));
$sum: dynamic(1px + 2px);
$rounded: dynamic(round(15px * 1.2));
$implicit-semicolon: dynamic(#123)
$forward: dynamic($later);
$later: dynamic(#456);
$palette: dynamic((
    'primary': #abc,
    'nested': ('light': #eee, 'dark': #111)
));
`)
	got, changed, reason := normalizeFashionDynamic(source)
	if reason != "" || !changed {
		t.Fatalf("normalizeFashionDynamic changed=%t, reason=%q", changed, reason)
	}
	if want := "$color: #ddd;\n$spacing: 12px;\n$enabled: true;\n$derived: $color;\n$mixed: mix(#fff, $color, 50%);\n$sum: 1px + 2px;\n$rounded: round(15px * 1.2);\n$implicit-semicolon: #123;\n$later: #456;\n$forward: $later;\n$palette: (\n    'primary': #abc,\n    'nested': ('light': #eee, 'dark': #111)\n);\n"; string(got) != want {
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

func TestNormalizeFashionDuplicateMixinArgument(t *testing.T) {
	source := []byte(`@include border-management(
    $border-radius: $border-radius,
    $border-radius-big $border-radius-big
);
`)
	want := `@include border-management(
    $border-radius: $border-radius,
    $border-radius-big: $border-radius-big
);
`
	if got := string(normalizeFashionDuplicateMixinArguments(source)); got != want {
		t.Fatalf("normalized Sass include = %q, want %q", got, want)
	}
}

func TestNormalizeFashionScopedFunctions(t *testing.T) {
	source := []byte(`@mixin datepanel-ui {
    .navigation-tools {
        @function calcPos($margin) {
            $sum: ceil($cell-width / 2) + $margin;
            @if $sum != null {
                @return calc(#{$sum} - 1em);
            } @else {
                @return 1em;
            }
        }
        right: calcPos(right($pane-margin));
        left: calcPos(left($pane-margin));
    }
    .cell {
        @function calcOffset($padding) {
            $ret: 0;
            $offset: $focused-cell-outline-offset;
            @if $offset != null {
                @if $padding == null { $padding: 0; }
                $ret: (-$padding - 1) + $offset;
            }
            @return $ret;
        }
        outline-offset: calcOffset($cell-spacing);
    }
}
`)
	got := string(normalizeFashionScopedFunctions(source))
	for _, want := range []string{
		"@function fashion-calc-pos($margin, $cell-width)",
		"right: fashion-calc-pos(right($pane-margin), $cell-width);",
		"left: fashion-calc-pos(left($pane-margin), $cell-width);",
		"@function fashion-calc-offset($padding, $focused-cell-outline-offset)",
		"outline-offset: fashion-calc-offset($cell-spacing, $focused-cell-outline-offset);",
		"@mixin datepanel-ui {",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("normalized Sass is missing %q: %s", want, got)
		}
	}
	if strings.Contains(got, "@function calcPos") || strings.Contains(got, "@function calcOffset") {
		t.Fatalf("nested Fashion function remained in Sass: %s", got)
	}
}

func TestNormalizeFashionButtonColorFunctionDefaults(t *testing.T) {
	source := []byte(`@function button-disabled-color($background-color, $color) {
    @return button-color($background-color);
}
`)
	want := `@function button-disabled-color($background-color, $color: null) {
    @return button-color($background-color);
}
`
	if got := string(normalizeFashionButtonColorFunctionDefaults(source)); got != want {
		t.Fatalf("normalized button color function = %q, want %q", got, want)
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

func TestResolvePackageSassVariablesSkipsExpressionsDependingOnNull(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"widget/sass/var/derived.scss": `$lighter-color: dynamic(darken($border-color, 10%));`,
		"widget/sass/var/alias.scss":   `$border-alias: dynamic($border-color);`,
		"theme/sass/var/theme.scss":    `$border-color: dynamic(null);`,
	})

	sources, _, variables, warnings, err := resolvePackageSassVariables(root, []string{
		filepath.Join(root, "widget", "sass", "var"),
		filepath.Join(root, "theme", "sass", "var"),
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantSources := []string{
		filepath.Join(root, "theme", "sass", "var", "theme.scss"),
		filepath.Join(root, "widget", "sass", "var", "alias.scss"),
	}
	if !reflect.DeepEqual(sources, wantSources) {
		t.Fatalf("package Sass sources = %#v, want null source only %#v", sources, wantSources)
	}
	if !variables["border-color"] || !variables["border-alias"] || variables["lighter-color"] {
		t.Fatalf("available variables = %#v, want border-color alias but no derived lighter-color", variables)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "Fashion value is null") {
		t.Fatalf("package Sass warnings = %#v, want null dependency warning", warnings)
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
		"theme-child/sass/var/child.scss": `$accent: dynamic(mix(#fff, $base-color, 50%));
$spacing: dynamic(1px + 2px);`,
		"theme-base/sass/var/base.scss": `$base-color: dynamic(#246);`,
		"app/Main.scss":                 `.main { color: $accent; margin: $spacing; }`,
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
	if !strings.Contains(string(css), "color:") || !strings.Contains(string(css), "margin: 3px") {
		t.Fatalf("compiled CSS = %s, want dynamic Sass expressions evaluated", css)
	}
}

func TestDartSassResolvesDynamicAssignmentsInImportedPartial(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}

	root := makeFixture(t, map[string]string{
		"app/_theme.scss": `$theme-color: dynamic(#abc);
$theme-palette: dynamic((
    'primary': #abc,
    'secondary': (light: #eee, dark: #111)
));`,
		"app/Main.scss": `@import "theme"; .main {
    color: $theme-color;
    border-color: map-get($theme-palette, 'primary');
}`,
	})
	css, warnings, err := compileClassSass(root, []string{filepath.Join(root, "app", "Main.scss")}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Sass warnings = %#v", warnings)
	}
	if !strings.Contains(string(css), "color: #abc") || !strings.Contains(string(css), "border-color: #abc") {
		t.Fatalf("compiled CSS = %s, want imported dynamic variable resolved", css)
	}
}

func TestDartSassCompilesFashionBoxFunctionsWithFallbackDefaults(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}

	root := makeFixture(t, map[string]string{
		"sass/functions.scss": `@function top($box, $fallbackBox) {
    @if $box == null { $box: $fallbackBox; }
    @return parsebox($box, 1);
}`,
		"sass/Main.scss": `.main { height: top(12px); }`,
	})
	functions := filepath.Join(root, "sass", "functions.scss")
	css, warnings, err := compileClassSassWithInlineImports(root, []string{
		functions,
		filepath.Join(root, "sass", "Main.scss"),
	}, false, map[string]bool{functions: true}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Sass warnings = %#v", warnings)
	}
	if !strings.Contains(string(css), "height: 12px") {
		t.Fatalf("compiled CSS = %s, want fallback argument default", css)
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
		"app.json":                               `{"name":"Example","namespace":"Example","toolkit":"classic","theme":"theme-custom","requires":["widget-lib"],"classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                         `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                       `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0","sass":{"etc":"${package.dir}/sass/etc/all.scss"}}`,
		"sdk/packages/core/package.json":         `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src","sass":{"etc":"${package.dir}/sass/etc/all.scss"}}}`,
		"sdk/packages/widget-lib/package.json":   `{"version":"1.0.0","sencha":{"name":"widget-lib","requires":["core"],"classpath":"${package.dir}/src","sass":{"src":"${package.dir}/sass/src"}}}`,
		"sdk/packages/theme-custom/package.json": `{"version":"1.0.0","sencha":{"name":"theme-custom","type":"theme","extend":"core","classpath":"${package.dir}/src","sass":{"etc":"${package.dir}/sass/etc/all.scss","var":"${package.dir}/sass/var"},"resources":[{"path":"${package.dir}/resources","output":"shared"}]}}`,
		"sdk/classic/classic/package.json":       `{"version":"8.0.0.43","sencha":{"name":"classic","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/sass/etc/all.scss":                  "$sdk-prefix: dynamic('sdk-');\n@import \"globals\";\n",
		"sdk/sass/etc/_globals.scss":             "$sdk-global-color: #789;",
		"sdk/packages/core/sass/etc/all.scss":    "$prefix: dynamic('x-');\n@import \"mixins\";\n",
		"sdk/packages/core/sass/etc/_mixins.scss": `$shared-color: #789;
@mixin unused-package-mixin { color: red; }`,
		"sdk/packages/theme-custom/sass/etc/all.scss":    "$theme-color: #abc;\n@import \"theme\";\n",
		"sdk/packages/theme-custom/sass/etc/_theme.scss": `@mixin theme-package-mixin { border-color: $theme-color; }`,
		"sdk/packages/theme-custom/sass/var/Button.scss": `$button-radius: 3px;
@mixin extjs-button-small-ui($ui) { .#{$prefix}btn-#{$ui} { border-radius: $button-radius; } }`,
		"sdk/packages/theme-custom/resources/images/theme-icon.png": "theme image",
		"sdk/packages/widget-lib/src/Widget.js":                     `Ext.define('Lib.Widget', {});`,
		"sdk/packages/widget-lib/sass/src/Widget.scss":              `.#{$prefix}widget { @include unused-package-mixin; @include theme-package-mixin; @include extjs-button-small-ui('test'); @include user-select(none); @include box-shadow(0 1px 2px #000); color: #246; background-color: $shared-color; outline-color: $sdk-global-color; background-image: theme-background-image('images/theme-icon', 'png'); }`,
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
	if !strings.Contains(string(css), ".x-widget") || !strings.Contains(string(css), ".x-btn-test") || !strings.Contains(string(css), "border-radius: 3px") || !strings.Contains(string(css), "-webkit-user-select: none") || !strings.Contains(string(css), "box-shadow: 0 1px 2px #000") || !strings.Contains(string(css), "color: red") || !strings.Contains(string(css), "border-color: #abc") || !strings.Contains(string(css), "color: #246") || !strings.Contains(string(css), "background-color: #789") || !strings.Contains(string(css), "outline-color: #789") || !strings.Contains(string(css), "../../shared/images/theme-icon.png") {
		t.Fatalf("compiled package Sass = %s", css)
	}
	imagePath := filepath.Join(root, "out", "resources", "shared", "images", "theme-icon.png")
	image, err := os.ReadFile(imagePath)
	if err != nil || string(image) != "theme image" {
		t.Fatalf("copied theme image = %q, err=%v", image, err)
	}
	for _, path := range []string{
		filepath.Join(root, "sdk", "sass", "etc", "all.scss"),
		filepath.Join(root, "sdk", "sass", "etc", "_globals.scss"),
		filepath.Join(root, "sdk", "packages", "core", "sass", "etc", "all.scss"),
		filepath.Join(root, "sdk", "packages", "core", "sass", "etc", "_mixins.scss"),
		filepath.Join(root, "sdk", "packages", "theme-custom", "sass", "etc", "all.scss"),
		filepath.Join(root, "sdk", "packages", "theme-custom", "sass", "etc", "_theme.scss"),
		filepath.Join(root, "sdk", "packages", "theme-custom", "sass", "var", "Button.scss"),
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
		"sdk/packages/core/sass/src/Panel.scss": `.#{$prefix}framework-panel { @include rotate(15deg); @include scale(1.1); color: #246; }`,
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
	if !strings.Contains(string(css), ".x-framework-panel") || !strings.Contains(string(css), "-webkit-transform: rotate(15deg)") || !strings.Contains(string(css), "transform: scale(1.1)") || !strings.Contains(string(css), "color: #246") {
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

func TestFilterSassSourcesByMixinsSkipsUnknownMixin(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app/Known.scss":           `.known { @include color-mixin; }`,
		"app/Unknown.scss":         `.unknown { @include extjs-button-small-ui; }`,
		"app/MixinDefinition.scss": `@mixin wrapper { .inner { @include generated-fashion-mixin; } }`,
	})
	known := filepath.Join(root, "app", "Known.scss")
	unknown := filepath.Join(root, "app", "Unknown.scss")
	definition := filepath.Join(root, "app", "MixinDefinition.scss")

	sources, warnings, err := filterSassSourcesByMixins([]string{known, unknown, definition}, map[string]bool{"color-mixin": true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sources, []string{known, definition}) {
		t.Fatalf("Sass sources = %#v, want known-mixin source and unused mixin definition", sources)
	}
	if len(warnings) != 1 || !strings.Contains(warnings[0], "extjs-button-small-ui") {
		t.Fatalf("Sass warnings = %#v, want unknown Fashion mixin warning", warnings)
	}
}

func TestResolveSassIncludeDefaultsForUndeclaredFeatures(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app/Main.scss": `$include-ext-button-segmented: true;
@if $include-ext-button-segmented { .button { display: inline-flex; } }
@if $include-slicer-border-radius { .sliced { border-radius: 4px; } }`,
	})
	defaults, err := resolveSassIncludeDefaults([]string{filepath.Join(root, "app", "Main.scss")}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(defaults) != 1 || defaults["include-slicer-border-radius"] != "false" {
		t.Fatalf("Sass include defaults = %#v, want slicer flag false and declared flag untouched", defaults)
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
