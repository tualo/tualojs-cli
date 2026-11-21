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
	if err == nil || !strings.Contains(err.Error(), "sass executable") {
		t.Fatalf("compileClassSass error = %v, want actionable missing-executable error", err)
	}
}

func TestCompileClassSassProvidesCornerRadiusMixins(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "Main.scss")
	if err := os.WriteFile(source, []byte(`.rounded {
    @include border-top-left-radius(4px);
    @include border-top-right-radius(4px);
    @include border-bottom-right-radius(4px);
    @include border-bottom-left-radius(4px);
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	css, _, err := compileClassSass(root, []string{source}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, property := range []string{
		"border-top-left-radius: 4px",
		"border-top-right-radius: 4px",
		"border-bottom-right-radius: 4px",
		"border-bottom-left-radius: 4px",
	} {
		if !bytes.Contains(css, []byte(property)) {
			t.Errorf("compiled CSS is missing %q: %s", property, css)
		}
	}
}

func TestCompileClassSassProvidesNullSafeBoxRotations(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "Main.scss")
	if err := os.WriteFile(source, []byte(`@if rotate90(null) == null { .null-rotation { display: none; } }
.box {
    padding: rotate90(1px 2px 3px 4px);
    margin: rotate180(1px 2px 3px 4px);
    border-width: rotate270(1px 2px 3px 4px);
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	css, _, err := compileClassSass(root, []string{source}, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{
		".null-rotation",
		"padding: 4px 1px 2px 3px",
		"margin: 3px 4px 1px 2px",
		"border-width: 2px 3px 4px 1px",
	} {
		if !bytes.Contains(css, []byte(want)) {
			t.Errorf("compiled rotation CSS is missing %q: %s", want, css)
		}
	}
}

func TestCompileClassSassPropagatesNullThroughDarkenAssignments(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := makeFixture(t, map[string]string{
		"theme/sass/var/Slider.scss": `$slider-track-border-color: dynamic(null);`,
		"ux/sass/var/Gauge.scss": `$gauge-track-stroke: dynamic($slider-track-border-color);
$gauge-needle-fill: dynamic(darken($slider-track-border-color, 40%));`,
		"app/Gauge.scss": `.gauge { width: 5px; stroke: $gauge-track-stroke; fill: $gauge-needle-fill; }`,
	})
	sources := []string{
		filepath.Join(root, "theme", "sass", "var", "Slider.scss"),
		filepath.Join(root, "ux", "sass", "var", "Gauge.scss"),
		filepath.Join(root, "app", "Gauge.scss"),
	}
	css, _, err := compileClassSass(root, sources, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(css, []byte(".gauge")) || !bytes.Contains(css, []byte("width: 5px")) || bytes.Contains(css, []byte("darken(")) {
		t.Fatalf("compiled CSS did not propagate Fashion null through darken: %s", css)
	}
}

func TestNormalizeFashionNullDarkenAssignments(t *testing.T) {
	source := []byte(`$slider-track-border-color: null;
$gauge-needle-fill: darken($slider-track-border-color, 40%);
`)
	got, changed := normalizeFashionNullDarkenAssignments(source, map[string]bool{"slider-track-border-color": true})
	if !changed || !bytes.Contains(got, []byte("$gauge-needle-fill: null;")) {
		t.Fatalf("normalized null darken assignment changed=%t: %s", changed, got)
	}
}

func TestCompileClassSassRecognizesCompassCompatibilityImports(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "Main.scss")
	if err := os.WriteFile(source, []byte(`@import "compass";
@import "compass/css3";
@import "compass/css3/images";
.rounded { @include border-radius(4px); @include opacity(.5); }
`), 0o644); err != nil {
		t.Fatal(err)
	}
	css, warnings, err := compileClassSassWithInlineImports(root, []string{source}, false, map[string]bool{source: true}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Sass warnings = %#v, want recognized Compass compatibility imports", warnings)
	}
	if !bytes.Contains(css, []byte("border-radius: 4px")) || !bytes.Contains(css, []byte("opacity: 0.5")) {
		t.Fatalf("compiled Compass-compatible CSS = %s", css)
	}
}

func TestCompileClassSassPreservesNestedCSSURLImports(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "Main.scss")
	if err := os.WriteFile(source, []byte(`@mixin google-font {
    @import url('https://fonts.example.test/css?family=Exo');
}
@include google-font;
`), 0o644); err != nil {
		t.Fatal(err)
	}
	css, warnings, err := compileClassSassWithInlineImports(root, []string{source}, false, map[string]bool{source: true}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Sass warnings = %#v, want nested CSS URL import preserved", warnings)
	}
	if !bytes.Contains(css, []byte(`@import url("https://fonts.example.test/css?family=Exo")`)) {
		t.Fatalf("compiled CSS lost nested import = %s", css)
	}
}

func TestCompileClassSassUsesKnownFashionJavaScriptShims(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "Main.scss")
	if err := os.WriteFile(source, []byte(`@import "theme/mixins/font-icon.js";
@import "ext/sass/etc/extensions.js";
@import "modern/theme-base/mixins/icon.js";
@import "core/functions.js";
$icon-args: parseFontIconArgs('A' 16px);
$simple-icon-args: parseIconArgs('B' FontAwesome 90);
.icon {
    font-size: nth($icon-args, 2);
    font-family: nth($simple-icon-args, 2);
    transform: rotate(#{nth($simple-icon-args, 3)}deg);
    background-image: url(file_join("assets", "fonts/icon.woff"));
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	css, warnings, err := compileClassSassWithInlineImports(root, []string{source}, false, map[string]bool{source: true}, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Fashion Sass shim warnings = %#v, want none", warnings)
	}
	if !bytes.Contains(css, []byte("font-size: 16px")) || !bytes.Contains(css, []byte("font-family: FontAwesome")) || !bytes.Contains(css, []byte("transform: rotate(90deg)")) || !bytes.Contains(css, []byte(`url("assets/fonts/icon.woff")`)) {
		t.Fatalf("compiled Fashion shim CSS = %s", css)
	}
}

func TestCompileClassSassEvaluatesEscapedFontAwesomeIconInterpolations(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "all.scss")
	icons := filepath.Join(root, "_icons.scss")
	if err := os.WriteFile(source, []byte(`@import "icons";`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(icons, []byte(`$fa-icons: ("0": '\30', "check": '\f00c');
$fa-brand-icons: ("brand-check": '\f00c');
@each $name, $icon in $fa-icons {
  .fa-#{$name}::before { content: unquote("\"\#{ $icon }\""); }
}
@each $name, $icon in $fa-brand-icons {
  .fab-#{$name}:before { content: unquote("\"\#{ $icon }\""); }
}`), 0o644); err != nil {
		t.Fatal(err)
	}
	css, warnings, err := compileClassSassWithInlineImports(root, []string{source}, false, nil, nil, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(warnings) != 0 {
		t.Fatalf("Font Awesome Sass warnings = %#v, want none", warnings)
	}
	content := string(css)
	if strings.Contains(content, `#{ $icon }`) || !strings.Contains(content, `.fa-0::before`) || !strings.Contains(content, `.fa-check::before`) || !strings.Contains(content, `.fab-brand-check:before`) || !strings.Contains(content, `content: "\f00c"`) {
		t.Fatalf("Font Awesome icon interpolation was not compiled: %s", content)
	}
}

func TestCollectFashionJavaScriptFunctionUses(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"unused.scss": `$sass-map: map-merge((a: 1), (b: 2));`,
		"used.scss": `$merged: map_merge((a: 1), (b: 2));
$arguments: intersect-arguments(first-ui, second-ui);`,
		"alias.scss": `$arguments: intersect_arguments(first-ui, second-ui);`,
	})
	used, err := collectFashionJavaScriptFunctionUses([]string{
		filepath.Join(root, "unused.scss"),
		filepath.Join(root, "used.scss"),
		filepath.Join(root, "alias.scss"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !used["map_merge"] || !used["intersect_arguments"] {
		t.Fatalf("Fashion JavaScript function uses = %#v, want map_merge and intersect_arguments", used)
	}
	if isFashionJavaScriptImportShim("functions.js", used) {
		t.Fatal("functions.js should remain unsupported when one of its functions is used")
	}
	if !isFashionJavaScriptImportShim("functions.js", nil) {
		t.Fatal("unused functions.js should use the empty-import shim")
	}
}

func TestPackageSassSrcIncludesExplicitEntryFiles(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := makeFixture(t, map[string]string{
		"pkg/package.json": "{}",
		"pkg/sass/src/all.scss": `$font-path: get-resource-path('fonts', 'shared');
@import "../etc/colors";
.package-entry { color: $package-color; src: url('#{$font-path}/sample.woff?v1#icon'); }`,
		"pkg/sass/etc/_colors.scss": `$package-color: #246;`,
		"pkg/sass/src/Unused.scss":  `.unused { color: red; }`,
	})
	pkg := &sdkPackage{root: filepath.Join(root, "pkg")}
	pkg.manifest.Sencha.Name = "sample-package"
	pkg.manifest.Sencha.Sass.Src = pathList{
		"${package.dir}/sass/src",
		"${package.dir}/sass/src/all.scss",
	}
	roots, rootWarnings, err := expandPackageClassSassRoots(pkg, "classic", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(rootWarnings) != 0 || len(roots) != 2 || !roots[1].EntryFile {
		t.Fatalf("package Sass roots = %#v, warnings = %#v, want directory plus explicit entry", roots, rootWarnings)
	}
	sources, watchPaths, warnings, err := resolvePackageClassSassSources(root, roots, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantEntry, err := filepath.Abs(filepath.Join(root, "pkg", "sass", "src", "all.scss"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(sources, []string{wantEntry}) {
		t.Fatalf("package Sass sources = %#v, want explicit entry %s", sources, wantEntry)
	}
	if !reflect.DeepEqual(watchPaths, []string{wantEntry}) || len(warnings) != 0 {
		t.Fatalf("package Sass watch paths = %#v, warnings = %#v", watchPaths, warnings)
	}
	css, _, err := compileClassSass(root, sources, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(css, []byte(".package-entry")) || !bytes.Contains(css, []byte("color: #246")) || !bytes.Contains(css, []byte("../../shared/fonts/sample.woff?v1#icon")) {
		t.Fatalf("compiled package Sass entry = %s", css)
	}
}

func TestUnsupportedPackageSassSourceAllowsLocalCallableParameters(t *testing.T) {
	localParameters := []byte(`@function add-if ($value1, $value2) {
    @if $value1 == null or $value2 == null { @return null; }
    @return $value1 + $value2;
}
@mixin spacing($gap, $color: red) {
    margin: $gap;
    color: $color;
}
`)
	if reason := unsupportedPackageSassSource(localParameters, nil); reason != "" {
		t.Fatalf("local Sass parameters were treated as missing package variables: %s", reason)
	}
	withMissingGlobal := []byte(`@function add-if ($value1) {
    @return $value1 + $not-loaded;
}
`)
	if reason := unsupportedPackageSassSource(withMissingGlobal, nil); !strings.Contains(reason, "$not-loaded") {
		t.Fatalf("missing global Sass variable reason = %q, want $not-loaded", reason)
	}
	withMissingDefault := []byte(`@mixin spacing($gap: $not-loaded) {
    margin: $gap;
}
`)
	if reason := unsupportedPackageSassSource(withMissingDefault, nil); !strings.Contains(reason, "$not-loaded") {
		t.Fatalf("missing Sass parameter default reason = %q, want $not-loaded", reason)
	}
}

func TestNormalizeFashionResourcePaths(t *testing.T) {
	source := []byte(`$font-path: get-resource-path('fonts');
$shared-path: get-resource-path("icons", "shared");
$package-path: get-resource-path('fonts', $packageName: 'ext');
`)
	got, changed, reason := normalizeFashionResourcePaths(source)
	if reason != "" || !changed {
		t.Fatalf("normalizeFashionResourcePaths changed=%t, reason=%q", changed, reason)
	}
	for _, want := range []string{
		`$font-path: "../../shared/fonts";`,
		`$shared-path: "../../shared/icons";`,
		`$package-path: get-resource-path('fonts', $packageName: 'ext');`,
	} {
		if !strings.Contains(string(got), want) {
			t.Errorf("normalized resource paths missing %q: %s", want, got)
		}
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
		if os.IsNotExist(err) {
			t.Skip("local Ext JS SDK fixture is not available; skipping SDK-specific Sass integration test")
		}
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

func TestNormalizeFashionDynamicOrdersVariablesBeforeTrailingMixins(t *testing.T) {
	source := []byte(`$offset: dynamic(round($height / 2));
$height: dynamic(6px);

@mixin menu-arrow {
    top: $offset;
}
`)
	got, changed, reason := normalizeFashionDynamic(source)
	if reason != "" || !changed {
		t.Fatalf("normalizeFashionDynamic changed=%t, reason=%q", changed, reason)
	}
	heightIndex := strings.Index(string(got), "$height: 6px;")
	offsetIndex := strings.Index(string(got), "$offset: round($height / 2);")
	mixinIndex := strings.Index(string(got), "@mixin menu-arrow")
	if heightIndex < 0 || offsetIndex < 0 || mixinIndex < 0 || heightIndex > offsetIndex || offsetIndex > mixinIndex {
		t.Fatalf("Sass assignments were not ordered ahead of the trailing mixin: %s", got)
	}
}

func TestCompileClassSassNormalizesNestedDynamicMapValues(t *testing.T) {
	binary := os.Getenv("SASS")
	if binary == "" {
		binary = "sass"
	}
	if _, err := exec.LookPath(binary); err != nil {
		t.Skip("Dart Sass is not installed")
	}
	root := t.TempDir()
	source := filepath.Join(root, "Main.scss")
	if err := os.WriteFile(source, []byte(`$summary-ui: dynamic((
    font-weight: dynamic(bold),
    padding: dynamic(8px 0)
));
.summary {
    font-weight: map-get($summary-ui, font-weight);
    padding: map-get($summary-ui, padding);
}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	css, _, err := compileClassSass(root, []string{source}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(css, []byte("font-weight: bold")) || !bytes.Contains(css, []byte("padding: 8px 0")) {
		t.Fatalf("compiled Sass with nested dynamic map values = %s", css)
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

func TestNormalizeFashionNestedMixins(t *testing.T) {
	source := []byte(`@mixin tabs($opacity) {
    @if $legacy {
        @mixin ie-tab-opacity() {
            -ms-filter: "alpha(opacity=#{$opacity * 100})";
        }
    }
    .tab {
        @if $opacity != 1 {
            @include ie-tab-opacity;
        }
    }
}
`)
	normalized, changed := normalizeFashionNestedMixins(source)
	if !changed || sassSourceHasNestedMixin(normalized) {
		t.Fatalf("nested Sass mixin was not normalized: %s", normalized)
	}
	for _, want := range []string{
		"@mixin ie-tab-opacity($opacity)",
		"@include ie-tab-opacity($opacity);",
		"@mixin tabs($opacity)",
	} {
		if !strings.Contains(string(normalized), want) {
			t.Errorf("normalized Sass is missing %q: %s", want, normalized)
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

func TestResolvePackageSassVariablesSupportsLocalImports(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"theme/sass/var/Breadcrumb.scss": `@import "Toolbar";
$breadcrumb-width: dynamic($toolbar-button-width);`,
		"theme/sass/var/Toolbar.scss": `$toolbar-button-width: dynamic(8px);
$toolbar-icon-color: dynamic(adjust-color($base-color, $hue: 0.8deg, $saturation: 7%));`,
		"theme/sass/var/Base.scss": `$base-color: dynamic(#246);`,
		"app/Main.scss":            `.breadcrumb { width: $breadcrumb-width; }`,
	})

	sources, watchPaths, variables, warnings, err := resolvePackageSassVariables(root, []string{
		filepath.Join(root, "theme", "sass", "var"),
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantSources := []string{
		filepath.Join(root, "theme", "sass", "var", "Base.scss"),
		filepath.Join(root, "theme", "sass", "var", "Toolbar.scss"),
		filepath.Join(root, "theme", "sass", "var", "Breadcrumb.scss"),
	}
	if !reflect.DeepEqual(sources, wantSources) {
		t.Fatalf("package Sass sources = %#v, want dependency order %#v", sources, wantSources)
	}
	if !variables["base-color"] || !variables["toolbar-button-width"] || !variables["toolbar-icon-color"] || !variables["breadcrumb-width"] || len(warnings) != 0 {
		t.Fatalf("available variables = %#v, warnings = %#v", variables, warnings)
	}
	wantWatchPaths := []string{
		filepath.Join(root, "theme", "sass", "var", "Base.scss"),
		filepath.Join(root, "theme", "sass", "var", "Breadcrumb.scss"),
		filepath.Join(root, "theme", "sass", "var", "Toolbar.scss"),
	}
	if !reflect.DeepEqual(watchPaths, wantWatchPaths) {
		t.Fatalf("package Sass watch paths = %#v, want both imported sources %#v", watchPaths, wantWatchPaths)
	}

	style := filepath.Join(root, "app", "Main.scss")
	css, _, err := compileClassSass(root, append(sources, style), false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(css, []byte("width: 8px")) {
		t.Fatalf("compiled Sass with local variable import = %s", css)
	}
}

func TestResolvePackageSassVariablesSkipsExpressionsDependingOnNull(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"widget/sass/var/derived.scss": `$lighter-color: dynamic(mix($border-color, #fff, 50%));`,
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

func TestResolvePackageSassVariablesKeepsNullAliasesAndOrFallbacks(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"widget/sass/var/colors.scss": `$border-alias: dynamic($border-color);
$fallback-color: dynamic($border-color or #246);
$rotated-border: dynamic(rotate90($border-color));`,
		"theme/sass/var/base.scss": `$border-color: dynamic(null);`,
		"app/Main.scss":            `.widget { color: $fallback-color; }`,
	})

	sources, _, variables, warnings, err := resolvePackageSassVariables(root, []string{
		filepath.Join(root, "widget", "sass", "var"),
		filepath.Join(root, "theme", "sass", "var"),
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantSources := []string{
		filepath.Join(root, "theme", "sass", "var", "base.scss"),
		filepath.Join(root, "widget", "sass", "var", "colors.scss"),
	}
	if !reflect.DeepEqual(sources, wantSources) {
		t.Fatalf("package Sass sources = %#v, want alias and fallback source after base", sources)
	}
	if !variables["border-alias"] || !variables["fallback-color"] || !variables["rotated-border"] || len(warnings) != 0 {
		t.Fatalf("available variables = %#v, warnings = %#v", variables, warnings)
	}
	css, _, err := compileClassSass(root, append(sources, filepath.Join(root, "app", "Main.scss")), false)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(css, []byte("color: #246")) {
		t.Fatalf("compiled Sass with null fallback = %s", css)
	}
}

func TestResolvePackageSassVariablesUsesLocalOverrideBeforeNullCheck(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"widget/sass/var/tab.scss": `$border-color: dynamic(#246);
$tab-border-width: dynamic(1px);
$tab-border-width-focus: dynamic($tab-border-width);`,
		"theme/sass/var/base.scss": `$border-color: dynamic(null);`,
	})

	sources, _, variables, warnings, err := resolvePackageSassVariables(root, []string{
		filepath.Join(root, "widget", "sass", "var"),
		filepath.Join(root, "theme", "sass", "var"),
	}, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	wantSources := []string{
		filepath.Join(root, "widget", "sass", "var", "tab.scss"),
		filepath.Join(root, "theme", "sass", "var", "base.scss"),
	}
	if !reflect.DeepEqual(sources, wantSources) {
		t.Fatalf("package Sass sources = %#v, want sources including local override %#v", sources, wantSources)
	}
	if !variables["tab-border-width-focus"] {
		t.Fatalf("available variables = %#v, want tab-border-width-focus", variables)
	}
	if len(warnings) != 0 {
		t.Fatalf("package Sass warnings = %#v, want none", warnings)
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
