package compiler

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBuildOrdersApplicationDependencies(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'}); Ext.require('Example.view.Required');`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.view.Main'
});`,
		"app/view/Main.js": `Ext.define('Example.view.Main', {
    extend: 'Ext.Panel',
    requires: ['Example.view.Greeting'],
    initialize: function () { this.created = Ext.create('Example.view.Created'); }
});`,
		"app/view/Greeting.js": `Ext.define('Example.view.Greeting', { extend: 'Ext.Component' });`,
		"app/view/Created.js":  `Ext.define('Example.view.Created', { extend: 'Ext.Component' });`,
		"app/view/Required.js": `Ext.define('Example.view.Required', { extend: 'Ext.Component' });`,
	})

	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceCount != 6 {
		t.Fatalf("source count = %d, want 6", result.SourceCount)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	positions := []int{
		strings.Index(content, "// Source: app/view/Greeting.js"),
		strings.Index(content, "// Source: app/view/Created.js"),
		strings.Index(content, "// Source: app/view/Main.js"),
		strings.Index(content, "// Source: app/Application.js"),
		strings.Index(content, "// Source: app/view/Required.js"),
		strings.Index(content, "// Source: app.js"),
	}
	for i, position := range positions {
		if position < 0 {
			t.Fatalf("bundle is missing source at position %d: %s", i, content)
		}
		if i > 0 && positions[i-1] >= position {
			t.Fatalf("sources are not dependency ordered: %v", positions)
		}
	}
}

func TestBuildPreservesLexicalDeclarationsByDefault(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":     `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":       `Ext.application({name: 'Example', extend: 'Example.Entry'});`,
		"app/Entry.js": `Ext.define('Example.Entry', { run: function (value) { let value = 'local'; return value; } });`,
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "let value = 'local'") {
		t.Fatalf("default build unexpectedly transformed lexical declaration: %s", bundle)
	}
}

func TestBuildModePreprocessesDebugDependencies(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js": `Ext.application({name: 'Example'});
//<debug>
Ext.require('Example.DebugOnly');
//</debug>
`,
		"app/DebugOnly.js": `Ext.define('Example.DebugOnly', { debugOnly: true });`,
	})

	development, err := BuildWithMode(root, "out-development", "", false, BuildModeDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	developmentBundle, err := os.ReadFile(development.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(developmentBundle), "Example.DebugOnly") {
		t.Fatal("development build should retain debug-only class reference")
	}

	production, err := BuildWithMode(root, "out-production", "", false, BuildModeProduction)
	if err != nil {
		t.Fatal(err)
	}
	productionBundle, err := os.ReadFile(production.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(productionBundle), "DebugOnly") {
		t.Fatalf("production build should remove debug-only dependency: %s", productionBundle)
	}
}

func TestBuildRejectsDependencyCycle(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.A'});`,
		"app/A.js": `Ext.define('Example.A', { requires: ['Example.B'] });`,
		"app/B.js": `Ext.define('Example.B', { requires: ['Example.A'] });`,
	})

	if _, err := Build(root, "out"); err == nil || !strings.Contains(err.Error(), "dependency cycle") {
		t.Fatalf("Build error = %v, want dependency cycle", err)
	}
}

func TestBuildRejectsPathOutsideApp(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"../outside.js","bundle":true}]}`,
	})
	if _, err := Build(root, "out"); err == nil || !strings.Contains(err.Error(), "escapes application directory") {
		t.Fatalf("Build error = %v, want path escape error", err)
	}
}

func TestBuildRefusesToOverwriteEntry(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":     `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":       `Ext.application({name: 'Example'});`,
		"app/Empty.js": "",
	})
	if _, err := Build(root, "."); err == nil || !strings.Contains(err.Error(), "overwrite JavaScript entry") {
		t.Fatalf("Build error = %v, want overwrite protection error", err)
	}
}

func TestBuildNormalizesShadowedFunctionParameter(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Route'});`,
		"app/Route.js": `Ext.define('Example.Route', {
    run: function (tablename) {
        let tablename = 'scss';
        return tablename;
    }
});`,
	})
	result, err := BuildWithOptions(root, "out", "", true)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bundle), "let tablename") || !strings.Contains(string(bundle), "var tablename = 'scss'") {
		t.Fatalf("bundle did not normalize parameter redeclaration: %s", bundle)
	}
}

func TestBuildIncludesOverridesForLoadedTargetsOnly(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"overrides":["overrides"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.Base'
});`,
		"app/Base.js": `Ext.define('Example.Base', { extend: 'Ext.Panel' });`,
		"overrides/Base.js": `Ext.define('Example.overrides.Base', {
    override: 'Example.Base',
    config: { userCls: 'overridden' }
});`,
		"overrides/Panel.js": `Ext.define('Example.overrides.Panel', {
    override: 'Ext.Panel'
});`,
		"overrides/Unused.js": `Ext.define('Example.overrides.Unused', {
    override: 'Example.Unused'
});`,
	})

	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	base := strings.Index(content, "// Source: app/Base.js")
	override := strings.Index(content, "// Source: overrides/Base.js")
	if base < 0 || override < 0 || base >= override {
		t.Fatalf("target should precede its override, positions = %d, %d", base, override)
	}
	if !strings.Contains(content, "// Source: overrides/Panel.js") {
		t.Fatal("override for referenced framework class Ext.Panel was not included")
	}
	if strings.Contains(content, "overrides/Unused.js") {
		t.Fatal("override for an unloaded target was included")
	}
}

func TestBuildExpandsNamespaceWildcard(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', requires: ['Example.*', 'Ext.*']});`,
		"app/A.js": `Ext.define('Example.A', { extend: 'Ext.Base' });`,
		"app/B.js": `Ext.define('Example.sub.B', { extend: 'Ext.Base' });`,
		"app/C.js": `Ext.define('Other.C', { extend: 'Ext.Base' });`,
	})

	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	for _, expected := range []string{"app/A.js", "app/B.js"} {
		if !strings.Contains(content, "// Source: "+expected) {
			t.Errorf("wildcard bundle is missing %s", expected)
		}
	}
	if strings.Contains(content, "app/C.js") {
		t.Fatal("wildcard escaped the application namespace")
	}
}

func TestBuildResolvesXtypeAndWidgetAliases(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.view.Main'
});`,
		"app/view/Main.js": `Ext.define('Example.view.Main', {
    extend: 'Ext.Panel',
    items: [{ xtype: 'example-card' }],
    createCard: function () { return Ext.widget('example-card'); }
});`,
		"app/view/Card.js": `Ext.define('Example.view.Card', {
    extend: 'Ext.Component',
    alias: 'widget.example-card'
});`,
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	card := strings.Index(content, "// Source: app/view/Card.js")
	main := strings.Index(content, "// Source: app/view/Main.js")
	if card < 0 || main < 0 || card >= main {
		t.Fatalf("xtype dependency should precede Main, positions = %d, %d", card, main)
	}
}

func TestBuildLoadsMixinBeforeConsumer(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.Controller'
});`,
		"app/Controller.js": `Ext.define('Example.Controller', {
    extend: 'Ext.app.ViewController',
    mixins: { util: 'Example.mixins.ControllerTools' }
});`,
		"app/mixins/ControllerTools.js": `Ext.define('Example.mixins.ControllerTools', {
    refresh: function () { return true; }
});`,
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	mixin := strings.Index(content, "// Source: app/mixins/ControllerTools.js")
	consumer := strings.Index(content, "// Source: app/Controller.js")
	if mixin < 0 || consumer < 0 || mixin >= consumer {
		t.Fatalf("mixin should precede consumer, positions = %d, %d", mixin, consumer)
	}
}

func TestBuildLoadsUsesAfterDeclaringClass(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.Main'
});`,
		"app/Main.js": `Ext.define('Example.Main', {
    extend: 'Ext.Panel',
    uses: ['Example.Optional']
});`,
		"app/Optional.js": `Ext.define('Example.Optional', {
    requires: ['Example.Main']
});`,
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	main := strings.Index(content, "// Source: app/Main.js")
	optional := strings.Index(content, "// Source: app/Optional.js")
	if main < 0 || optional < 0 || main >= optional {
		t.Fatalf("uses class should follow its declaring class, positions = %d, %d", main, optional)
	}
}

func TestBuildCopiesConfiguredResources(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":             `{"name":"Example","namespace":"Example","classpath":["app"],"resources":[{"path":"resources","output":"shared"},{"path":"assets/logo.txt","output":"images/logo.txt"}],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":               `Ext.application({name: 'Example'});`,
		"app/Empty.js":         "",
		"resources/readme.txt": "shared resource",
		"assets/logo.txt":      "logo asset",
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceFiles != 2 {
		t.Fatalf("copied resource files = %d, want 2", result.ResourceFiles)
	}
	for relative, expected := range map[string]string{
		"shared/readme.txt": "shared resource",
		"images/logo.txt":   "logo asset",
	} {
		content, err := os.ReadFile(filepath.Join(result.ResourcesPath, relative))
		if err != nil {
			t.Fatalf("read copied resource %s: %v", relative, err)
		}
		if string(content) != expected {
			t.Errorf("resource %s = %q, want %q", relative, content, expected)
		}
	}
}

func TestBuildWritesLineLevelSourceMap(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Entry'});`,
		"app/Entry.js": `Ext.define('Example.Entry', {
    extend: 'Ext.Component',
    value: 'source mapped'
});`,
	})
	result, err := BuildWithSourceMap(root, "out", "", false, BuildModeDevelopment, true)
	if err != nil {
		t.Fatal(err)
	}
	if result.SourceMapPath != result.OutputPath+".map" {
		t.Fatalf("source map path = %q, want %q", result.SourceMapPath, result.OutputPath+".map")
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "//# sourceMappingURL=app.js.map") {
		t.Fatalf("bundle is missing sourceMappingURL: %s", bundle)
	}
	mapData, err := os.ReadFile(result.SourceMapPath)
	if err != nil {
		t.Fatal(err)
	}
	var sourceMap struct {
		Version        int      `json:"version"`
		File           string   `json:"file"`
		Sources        []string `json:"sources"`
		SourcesContent []string `json:"sourcesContent"`
		Mappings       string   `json:"mappings"`
	}
	if err := json.Unmarshal(mapData, &sourceMap); err != nil {
		t.Fatal(err)
	}
	if sourceMap.Version != 3 || sourceMap.File != "app.js" || len(sourceMap.Sources) != 2 || len(sourceMap.SourcesContent) != 2 || sourceMap.Mappings == "" {
		t.Fatalf("unexpected source map: %#v", sourceMap)
	}
	if !strings.Contains(strings.Join(sourceMap.SourcesContent, "\n"), "source mapped") {
		t.Fatalf("sourcesContent did not retain source text: %#v", sourceMap.SourcesContent)
	}
}

func TestBuildMinifiesAndKeepsSourceMap(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({ name: 'Example', extend: 'Example.Entry' });`,
		"app/Entry.js": `Ext.define('Example.Entry', {
    extend: 'Ext.Component', // remove this comment
    value: 'kept string'
});`,
	})
	result, err := BuildWithOutputOptions(root, "out", "", false, BuildModeDevelopment, true, true)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bundle), "remove this comment") || !strings.Contains(string(bundle), "value:'kept string'") || !strings.Contains(string(bundle), "sourceMappingURL=app.js.map") {
		t.Fatalf("unexpected minified bundle: %s", bundle)
	}
	if _, err := os.Stat(result.SourceMapPath); err != nil {
		t.Fatalf("missing source map: %v", err)
	}
}

func TestBuildRejectsResourceOutputOutsideBuildResources(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":           `{"name":"Example","namespace":"Example","classpath":["app"],"resources":[{"path":"resources","output":"../outside"}],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":             `Ext.application({name: 'Example'});`,
		"app/Empty.js":       "",
		"resources/file.txt": "file",
	})
	if _, err := Build(root, "out"); err == nil || !strings.Contains(err.Error(), "escapes resources directory") {
		t.Fatalf("Build error = %v, want resource output traversal error", err)
	}
}

func TestBuildExpandsSenchaBuildIDResourcePath(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":     `{"name":"Example","namespace":"Example","classpath":["app"],"resources":[{"path":"${build.id}/resources"}],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":       `Ext.application({name: 'Example'});`,
		"app/Empty.js": "",
		"build/production/Example/resources/generated.txt": "from prior Sencha build",
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(filepath.Join(result.ResourcesPath, "generated.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "from prior Sencha build" {
		t.Fatalf("copied resource = %q, want prior-build content", content)
	}
}

func TestBuildExpandsToolkitClasspath(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":           `{"name":"Example","namespace":"Example","classpath":["app","${toolkit.name}/src","both"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":             `Ext.application({name: 'Example', extend: 'Example.Root'}); Ext.require('Example.ToolkitClass');`,
		"app/Root.js":        `Ext.define('Example.Root', { extend: 'Ext.Base' });`,
		"modern/src/M.js":    `Ext.define('Example.ToolkitClass', { source: 'modern' });`,
		"classic/src/C.js":   `Ext.define('Example.ToolkitClass', { source: 'classic' });`,
		"both/src/Shared.js": `Ext.define('Example.ToolkitClass', { source: 'shared' });`,
	})

	if _, err := Build(root, "out-without-toolkit"); err == nil || !strings.Contains(err.Error(), "requires a toolkit") {
		t.Fatalf("Build error = %v, want toolkit selection error", err)
	}
	result, err := BuildWithToolkit(root, "out-modern", "modern")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "// Source: modern/src/M.js") || strings.Contains(string(bundle), "both/src/Shared.js") || strings.Contains(string(bundle), "classic/src/C.js") {
		t.Fatalf("toolkit classpath selection failed: %s", bundle)
	}
	classicResult, err := BuildWithToolkit(root, "out-classic", "classic")
	if err != nil {
		t.Fatal(err)
	}
	classicBundle, err := os.ReadFile(classicResult.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(classicBundle), "// Source: classic/src/C.js") || strings.Contains(string(classicBundle), "modern/src/M.js") {
		t.Fatalf("classic toolkit classpath selection failed: %s", classicBundle)
	}
}

func TestBuildPrefersCanonicalFilenameForDuplicateClass(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":            `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":              `Ext.application({name: 'Example', extend: 'Example.Store'});`,
		"app/Store.js":        `Ext.define('Example.Store', { source: 'canonical' });`,
		"app/Store.backup.js": `Ext.define('Example.Store', { source: 'backup' });`,
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	if !strings.Contains(content, "// Source: app/Store.js") || strings.Contains(content, "// Source: app/Store.backup.js") {
		t.Fatalf("canonical class source was not selected: %s", content)
	}
}

func TestBuildWarnsForUnresolvedApplicationClass(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":           `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":             `Ext.application({name: 'Example', extend: 'Example.Application'}); Ext.create('Example.DynamicClass');`,
		"app/Application.js": `Ext.define('Example.Application', { extend: 'Ext.app.Application' });`,
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "Example.DynamicClass") {
		t.Fatalf("warnings = %#v, want unresolved Example.DynamicClass", result.Warnings)
	}
}

func TestBuildLoadsTransitiveSDKPackageClasspaths(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","requires":["widget-alias"],"classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json": `{
// local workspace configuration
"frameworks": {"ext": {"path": "sdk", "version": "8.0.0.43"}}
}`,
		"sdk/package.json":                      `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":        `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/modern/modern/package.json":        `{"version":"8.0.0.43","sencha":{"name":"modern","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/packages/base-lib/package.json":    `{"version":"1.0.0","sencha":{"name":"base-lib","classpath":"${package.dir}/src"}}`,
		"sdk/packages/widget-lib/package.json":  `{"version":"1.0.0","sencha":{"name":"widget-lib","alternateName":["widget-alias"],"requires":["base-lib"],"classpath":["${package.dir}/src","${package.dir}/${toolkit.name}/src"],"resources":[{"path":"${package.dir}/resources","output":"shared"}]}}`,
		"sdk/packages/base-lib/src/Base.js":     `Ext.define('Lib.Base', { extend: 'Ext.Base' });`,
		"sdk/packages/widget-lib/src/Widget.js": `Ext.define('Lib.Widget', { extend: 'Lib.Base' });`,
		"sdk/packages/widget-lib/src/draw/AnimationParser.js": `Ext.define('Ext.draw.sprite.AnimationParser', function () {
    return { singleton: true, requires: ['Ext.draw.Color'] };
});`,
		"sdk/packages/widget-lib/modern/src/Toolkit.js": `Ext.define('Lib.Toolkit', { extend: 'Lib.Base' });`,
		"sdk/packages/widget-lib/resources/icon.txt":    "package resource",
		"app.js": `Ext.application({name: 'Example', extend: 'Example.Application'}); Ext.require('Ext.*');`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.view.Main'
});`,
		"app/view/Main.js": `Ext.define('Example.view.Main', {
    extend: 'Ext.Panel',
    requires: ['Lib.Widget', 'Lib.Toolkit']
});`,
	})

	if _, err := Build(root, "out-without-toolkit"); err == nil || !strings.Contains(err.Error(), "requires a toolkit") {
		t.Fatalf("Build error = %v, want package toolkit selection error", err)
	}
	result, err := BuildWithToolkit(root, "out", "modern")
	if err != nil {
		t.Fatal(err)
	}
	if result.FrameworkVersion != "8.0.0.43" {
		t.Fatalf("framework version = %q, want 8.0.0.43", result.FrameworkVersion)
	}
	if got, want := result.Packages, []string{"base-lib", "widget-lib"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved packages = %#v, want %#v", got, want)
	}
	if result.ResourceFiles != 1 {
		t.Fatalf("copied package resources = %d, want 1", result.ResourceFiles)
	}
	resource, err := os.ReadFile(filepath.Join(result.ResourcesPath, "shared", "icon.txt"))
	if err != nil || string(resource) != "package resource" {
		t.Fatalf("package resource = %q, err = %v", resource, err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(bundle)
	base := strings.Index(content, "// Source: sdk/packages/base-lib/src/Base.js")
	widget := strings.Index(content, "// Source: sdk/packages/widget-lib/src/Widget.js")
	main := strings.Index(content, "// Source: app/view/Main.js")
	if base < 0 || widget < 0 || main < 0 || base >= widget || widget >= main {
		t.Fatalf("package classes were not ordered before the consuming app class: %v", []int{base, widget, main})
	}
	if !strings.Contains(content, "// Source: sdk/packages/widget-lib/modern/src/Toolkit.js") {
		t.Fatal("toolkit-specific package classpath was not included")
	}
	if !strings.Contains(content, "// Source: sdk/packages/widget-lib/src/draw/AnimationParser.js") {
		t.Fatal("factory-defined class was not included by Ext.* wildcard")
	}
}

func TestBuildRejectsIncompatibleSDK(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":         `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":   `{"frameworks":{"ext":{"path":"sdk","version":"7.6.0.41"}}}`,
		"sdk/package.json": `{"name":"ext","version":"7.6.0.41","compatVersion":"7.6.0"}`,
		"app.js":           `Ext.application({name: 'Example'});`,
		"app/Empty.js":     "",
	})
	if _, err := Build(root, "out"); err == nil || !strings.Contains(err.Error(), "targets 8.0.0") {
		t.Fatalf("Build error = %v, want Ext JS 8.0.0 compatibility error", err)
	}
}

func makeFixture(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}
