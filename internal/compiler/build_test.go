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

func TestBuildPreprocessesToolkitConditionDependencies(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    // <if classic>
    requires: ['Example.ClassicOnly'],
    // </if>
    // <if modern>
    requires: ['Example.ModernOnly'],
    // </if>
});`,
		"app/ClassicOnly.js": `Ext.define('Example.ClassicOnly', {});`,
		"app/ModernOnly.js":  `Ext.define('Example.ModernOnly', {});`,
	})

	for _, test := range []struct {
		toolkit string
		present string
		absent  string
	}{
		{toolkit: "classic", present: "app/ClassicOnly.js", absent: "app/ModernOnly.js"},
		{toolkit: "modern", present: "app/ModernOnly.js", absent: "app/ClassicOnly.js"},
	} {
		result, err := BuildWithToolkit(root, "out-"+test.toolkit, test.toolkit)
		if err != nil {
			t.Fatalf("build %s: %v", test.toolkit, err)
		}
		bundle, err := os.ReadFile(result.OutputPath)
		if err != nil {
			t.Fatal(err)
		}
		content := string(bundle)
		if !strings.Contains(content, "// Source: "+test.present) || strings.Contains(content, "// Source: "+test.absent) {
			t.Errorf("%s build selected wrong conditional dependencies: %s", test.toolkit, content)
		}
		if strings.Contains(content, "<if classic>") || strings.Contains(content, "<if modern>") || strings.Contains(content, "</if>") {
			t.Errorf("%s build retained toolkit markers", test.toolkit)
		}
	}
}

func TestBuildPreprocessesPlatformConditionDependencies(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    //<if desktop>
    requires: ['Example.DesktopOnly'],
    //</if desktop>
    //<if phone>
    requires: ['Example.PhoneOnly'],
    //</if phone>
});`,
		"app/DesktopOnly.js": `Ext.define('Example.DesktopOnly', {});`,
		"app/PhoneOnly.js":   `Ext.define('Example.PhoneOnly', {});`,
	})

	for _, test := range []struct {
		platform string
		present  string
		absent   string
	}{
		{platform: "desktop", present: "app/DesktopOnly.js", absent: "app/PhoneOnly.js"},
		{platform: "phone", present: "app/PhoneOnly.js", absent: "app/DesktopOnly.js"},
	} {
		result, err := BuildWithTargetOptions(root, "out-"+test.platform, "classic", test.platform, false, BuildModeDevelopment, false, false)
		if err != nil {
			t.Fatalf("build %s: %v", test.platform, err)
		}
		bundle, err := os.ReadFile(result.OutputPath)
		if err != nil {
			t.Fatal(err)
		}
		content := string(bundle)
		if !strings.Contains(content, "// Source: "+test.present) || strings.Contains(content, "// Source: "+test.absent) {
			t.Errorf("%s build selected wrong platform dependencies: %s", test.platform, content)
		}
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
    createCard: function () { return Ext.widget('example-card'); },
    createByAlias: function () { return Ext.createByAlias('widget.example-card'); }
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

func TestBuildResolvesControllerViewModelAndPluginAliases(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.view.Main'
});`,
		"app/view/Main.js": `Ext.define('Example.view.Main', {
    extend: 'Ext.Panel',
    controller: 'main',
    viewModel: 'main',
    plugins: ['gridexporter']
});`,
		"app/view/MainController.js": `Ext.define('Example.view.MainController', { alias: 'controller.main' });`,
		"app/view/MainModel.js":      `Ext.define('Example.view.MainModel', { alias: 'viewmodel.main' });`,
		"app/plugin/GridExporter.js": `Ext.define('Example.plugin.GridExporter', { alias: 'plugin.gridexporter' });`,
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
	main := strings.Index(content, "// Source: app/view/Main.js")
	for _, dependency := range []string{"app/view/MainController.js", "app/view/MainModel.js", "app/plugin/GridExporter.js"} {
		position := strings.Index(content, "// Source: "+dependency)
		if position < 0 || position >= main {
			t.Errorf("alias dependency %s should precede Main, position = %d, Main = %d", dependency, position, main)
		}
	}
}

func TestBuildResolvesAlternateClassName(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":   `Ext.application({name: 'Example'}); Ext.create('Legacy.Button');`,
		"app/Button.js": `Ext.define('Example.Button', {
    extend: 'Ext.Button',
    alternateClassName: ['Legacy.Button']
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
	button := strings.Index(content, "// Source: app/Button.js")
	entry := strings.Index(content, "// Source: app.js")
	if button < 0 || entry < 0 || button >= entry {
		t.Fatalf("alternate class name was not resolved before entry, positions = %d, %d", button, entry)
	}
}

func TestBuildScansStaticLoaderPathDirectories(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json": `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js": `
Ext.Loader.setPath('Example.lazy', './lazy');
Ext.application({name: 'Example', extend: 'Example.Application'});
Ext.create('Example.lazy.Widget');
`,
		"app/Application.js": `Ext.define('Example.Application', { extend: 'Ext.app.Application' });`,
		"lazy/Widget.js":     `Ext.define('Example.lazy.Widget', { extend: 'Ext.Component' });`,
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
	widget := strings.Index(content, "// Source: lazy/Widget.js")
	entry := strings.Index(content, "// Source: app.js")
	if widget < 0 || entry < 0 || widget >= entry {
		t.Fatalf("loader-mapped class was not included before the entry: %d, %d", widget, entry)
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
		"app.json":             `{"name":"Example","namespace":"Example","classpath":["app"],"resources":[{"path":"resources","output":"shared"},{"path":"assets/logo.txt","output":"images/logo.txt"}],"css":[{"path":"styles/app.css"}],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":               `Ext.application({name: 'Example'});`,
		"app/Empty.js":         "",
		"resources/readme.txt": "shared resource",
		"assets/logo.txt":      "logo asset",
		"styles/app.css":       "body { color: red; }",
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceFiles != 2 {
		t.Fatalf("copied resource files = %d, want 2", result.ResourceFiles)
	}
	if len(result.StyleSheets) != 1 || result.StyleSheets[0].URL != "styles/app.css" {
		t.Fatalf("stylesheet outputs = %#v, want styles/app.css", result.StyleSheets)
	}
	stylesheet, err := os.ReadFile(result.StyleSheets[0].Path)
	if err != nil || string(stylesheet) != "body { color: red; }" {
		t.Fatalf("copied stylesheet = %q, err = %v", stylesheet, err)
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

func TestBuildCompilesSCSSPairedWithClasspathJavaScript(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":                      `{"name":"Example","namespace":"Example","toolkit":"classic","classpath":["app"],"sass":{"etc":"sass/etc/all.scss","var":"sass/var","src":["sass/src","${toolkit.name}/sass/src"]},"css":[{"path":"styles/base.css"}],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":                        `Ext.application({name: 'Example'});`,
		"app/Fashion.js":                `Ext.define('Example.Fashion', {});`,
		"app/Fashion.scss":              `@include extjs-button-small-ui($ui: 'sample');`,
		"app/Main.js":                   `Ext.define('Example.Main', {});`,
		"app/Main.scss":                 `.main { border: 0px; }`,
		"app/Orphan.scss":               `.orphan { display: none; }`,
		"sass/etc/all.scss":             "$base-color: #246;",
		"sass/var/colors.scss":          "$main-color: $base-color;",
		"sass/src/base.scss":            `.configured { color: $main-color; }`,
		"classic/sass/src/toolkit.scss": `.toolkit { color: $main-color; }`,
		"styles/base.css":               "body { color: black; }",
	})

	sassBinary := filepath.Join(t.TempDir(), "sass")
	sassScript := `#!/bin/sh
if [ "$1" != "--no-source-map" ] || [ "$2" != "--style=compressed" ] || [ "$3" != "--load-path" ] || [ "$5" != "--stdin" ]; then
    echo "unexpected Sass arguments: $*" >&2
    exit 2
fi
input=$(cat)
case "$input" in
    *'@import "sass/etc/all.scss";'*'@import "sass/var/colors.scss";'*'@import "sass/src/base.scss";'*'@import "classic/sass/src/toolkit.scss";'*'@import "app/Main.scss";'*) ;;
    *) echo "Sass entry has unexpected sources or ordering: $input" >&2; exit 3 ;;
esac
case "$input" in
    *Fashion.scss*) echo "Sass entry should omit Fashion.scss: $input" >&2; exit 4 ;;
esac
printf '.configured{color:#246}.toolkit{color:#246}.main{border:0px}\n'
`
	if err := os.WriteFile(sassBinary, []byte(sassScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SASS", sassBinary)

	result, err := BuildWithOutputOptions(root, "out", "", false, BuildModeProduction, false, true)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := len(result.StyleSheets), 2; got != want {
		t.Fatalf("stylesheet count = %d, want %d", got, want)
	}
	if result.StyleSheets[0].URL != "styles/base.css" || result.StyleSheets[1].URL != "app-sass.css" {
		t.Fatalf("stylesheet order = %#v, want configured CSS followed by generated class Sass", result.StyleSheets)
	}
	if result.StyleSheets[1].Path != filepath.Join(root, "out", "classic", "resources", "app-sass.css") {
		t.Fatalf("generated Sass path = %q, want file under classic/resources", result.StyleSheets[1].Path)
	}
	compiled, err := os.ReadFile(result.StyleSheets[1].Path)
	if err != nil {
		t.Fatal(err)
	}
	if string(compiled) != ".configured{color:#246}.toolkit{color:#246}.main{border:0px}\n" {
		t.Fatalf("generated Sass CSS = %q", compiled)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "Fashion-dependent Sass source") || !strings.Contains(result.Warnings[0], "Fashion.scss") {
		t.Fatalf("Sass warnings = %#v, want one Fashion-dependent stylesheet warning", result.Warnings)
	}
	for _, source := range result.WatchPaths {
		if source == filepath.Join(root, "app", "Main.scss") {
			return
		}
	}
	t.Fatal("class Sass source is missing from watcher paths")
}

func TestBuildLaterResourceEntryOverridesCollision(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":             `{"name":"Example","namespace":"Example","classpath":["app"],"resources":[{"path":"app-resource.txt","output":"shared/Readme.md"},{"path":"package-resource.txt","output":"shared/Readme.md"}],"js":[{"path":"app.js","bundle":true}]}`,
		"app.js":               `Ext.application({name: 'Example'});`,
		"app/Empty.js":         "",
		"app-resource.txt":     "application resource",
		"package-resource.txt": "package resource",
	})

	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	if result.ResourceFiles != 1 {
		t.Fatalf("copied resource files = %d, want 1 unique output", result.ResourceFiles)
	}
	content, err := os.ReadFile(filepath.Join(result.ResourcesPath, "shared", "Readme.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "package resource" {
		t.Fatalf("colliding resource = %q, want later entry to take precedence", content)
	}
}

func TestBuildCopiesNonBundleJavaScriptAssetsInManifestOrder(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":      `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"vendor/pre.js"},{"path":"app.js","bundle":true},{"path":"https://cdn.example/after.js","remote":true}]}`,
		"app.js":        `Ext.application({name: 'Example'});`,
		"app/Empty.js":  "",
		"vendor/pre.js": "window.preload = true;",
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	wantURLs := []string{"js/vendor/pre.js", "app.js", "https://cdn.example/after.js"}
	if len(result.Scripts) != len(wantURLs) {
		t.Fatalf("script assets = %#v, want %d entries", result.Scripts, len(wantURLs))
	}
	for i, want := range wantURLs {
		if result.Scripts[i].URL != want {
			t.Fatalf("script %d URL = %q, want %q", i, result.Scripts[i].URL, want)
		}
	}
	if !result.Scripts[2].Remote {
		t.Fatal("remote JavaScript entry lost its remote flag")
	}
	asset, err := os.ReadFile(filepath.Join(filepath.Dir(result.OutputPath), "js", "vendor", "pre.js"))
	if err != nil || string(asset) != "window.preload = true;" {
		t.Fatalf("copied JavaScript asset = %q, err = %v", asset, err)
	}
}

func TestBuildIncludesMarkedJavaScriptInBundle(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":     `{"name":"Example","namespace":"Example","classpath":["app"],"js":[{"path":"support.js","includeInBundle":true},{"path":"app.js","bundle":true}]}`,
		"app.js":       `Ext.application({name: 'Example'});`,
		"app/Empty.js": "",
		"support.js":   `window.ExampleSupport = true;`,
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
	support := strings.Index(content, "// Source: support.js")
	entry := strings.Index(content, "// Source: app.js")
	if support < 0 || entry < 0 || support >= entry || !strings.Contains(content, "window.ExampleSupport = true") {
		t.Fatalf("includeInBundle source was not bundled before app entry: %s", content)
	}
	if len(result.Scripts) != 1 || result.Scripts[0].URL != "app.js" {
		t.Fatalf("bundle script manifest = %#v, want only app.js", result.Scripts)
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
	if sourceMap.Version != 3 || sourceMap.File != "app.js" || len(sourceMap.Sources) != 2 || len(sourceMap.SourcesContent) != 2 || !strings.Contains(sourceMap.Mappings, ",") {
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

func TestBuildHonorsAppIgnorePatterns(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":                 `{"name":"Example","namespace":"Example","classpath":["app"],"ignore":["(^|/)CVS(/?$|/.*?$)"],"resources":[{"path":"resources","output":"shared"}],"css":[{"path":"styles/app.css"}],"js":[{"path":"app.js","bundle":true},{"path":"assets/CVS/ignored.js"}]}`,
		"app.js":                   `Ext.application({name: 'Example', extend: 'Example.Widget'});`,
		"app/Widget.js":            `Ext.define('Example.Widget', { extend: 'Ext.Component' });`,
		"app/CVS/Old.js":           `Ext.define('Example.Old', { extend: 'Ext.Component' });`,
		"assets/CVS/ignored.js":    `window.ignored = true;`,
		"resources/Readme.txt":     "kept",
		"resources/CVS/secret.txt": "ignored",
		"styles/app.css":           "body { color: red; }",
	})
	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(bundle), "app/CVS/Old.js") {
		t.Fatal("ignored classpath file was included")
	}
	if result.ResourceFiles != 1 {
		t.Fatalf("resource count = %d, want only non-ignored file", result.ResourceFiles)
	}
	if len(result.StyleSheets) != 1 || result.StyleSheets[0].URL != "styles/app.css" {
		t.Fatalf("ignored JS entry affected stylesheet manifest: %#v", result.StyleSheets)
	}
	if len(result.Scripts) != 1 || result.Scripts[0].URL != "app.js" {
		t.Fatalf("ignored script entry was copied: %#v", result.Scripts)
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

func TestBuildExpandsAppPathMacros(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":                  `{"name":"Example","namespace":"Example","classpath":["${app.dir}/app","${app.dir}/shared/${app.name}","${toolkit.name}/src"],"overrides":["${app.dir}/overrides/${app.name}"],"js":[{"path":"${app.dir}/app.js","bundle":true},{"path":"${app.dir}/assets/${app.name}.js"}],"css":[{"path":"${app.dir}/styles/${app.name}.css"}],"resources":[{"path":"${app.dir}/resources/${app.name}.txt","output":"${app.name}"}]}`,
		"app.js":                    `Ext.application({name: 'Example', extend: 'Example.Root'}); Ext.require(['Example.Shared', 'Example.ToolkitClass']);`,
		"app/Root.js":               `Ext.define('Example.Root', { extend: 'Ext.Base' });`,
		"shared/Example/Shared.js":  `Ext.define('Example.Shared', { extend: 'Ext.Base' });`,
		"modern/src/Toolkit.js":     `Ext.define('Example.ToolkitClass', { extend: 'Ext.Base' });`,
		"overrides/Example/Root.js": `Ext.define('Example.RootOverride', { override: 'Example.Root', marker: true });`,
		"assets/Example.js":         `window.exampleAsset = true;`,
		"styles/Example.css":        `.example { color: red; }`,
		"resources/Example.txt":     "resource content",
	})

	result, err := BuildWithToolkit(root, "out", "modern")
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{"app/Root.js", "shared/Example/Shared.js", "modern/src/Toolkit.js", "overrides/Example/Root.js"} {
		if !strings.Contains(string(bundle), "// Source: "+source) {
			t.Errorf("bundle is missing expanded path source %s", source)
		}
	}
	if len(result.Scripts) != 2 || result.Scripts[1].URL != "js/assets/Example.js" {
		t.Fatalf("scripts = %#v, want expanded JavaScript asset URL", result.Scripts)
	}
	if len(result.StyleSheets) != 1 || result.StyleSheets[0].URL != "styles/Example.css" {
		t.Fatalf("stylesheets = %#v, want expanded stylesheet URL", result.StyleSheets)
	}
	if _, err := os.Stat(filepath.Join(result.ResourcesPath, "Example", "Example.txt")); err != nil {
		t.Fatalf("expanded resource output was not copied: %v", err)
	}
}

func TestBuildAcceptsCommaSeparatedClasspath(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":           `{"name":"Example","namespace":"Example","classpath":"app, shared","js":[{"path":"app.js","bundle":true}]}`,
		"app.js":             `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js": `Ext.define('Example.Application', { mainView: 'Example.Main' });`,
		"app/Main.js":        `Ext.define('Example.Main', { extend: 'Example.SharedBase' });`,
		"shared/Base.js":     `Ext.define('Example.SharedBase', { extend: 'Ext.Component' });`,
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
	base := strings.Index(content, "// Source: shared/Base.js")
	main := strings.Index(content, "// Source: app/Main.js")
	if base < 0 || main < 0 || base >= main {
		t.Fatalf("comma-separated classpath was not loaded, positions = %d, %d", base, main)
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
		"app.json": `{"name":"Example","namespace":"Example","locale":"de","requires":["widget-alias","ext-locale"],"classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json": `{
// local workspace configuration
"frameworks": {"ext": {"path": "sdk", "version": "8.0.0.43"}}
}`,
		"sdk/package.json":                                     `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":                       `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/modern/modern/package.json":                       `{"version":"8.0.0.43","sencha":{"name":"modern","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/packages/base-lib/package.json":                   `{"version":"1.0.0","sencha":{"name":"base-lib","classpath":"${package.dir}/src","resource":{"paths":"${package.dir}/resources"}}}`,
		"sdk/packages/widget-lib/package.json":                 `{"version":"1.0.0","sencha":{"name":"widget-lib","alternateName":["widget-alias"],"requires":["base-lib"],"classpath":["${package.dir}/src","${package.dir}/${toolkit.name}/src"],"resource":{"paths":["${package.dir}/assets","${package.dir}/${toolkit.name}/resources"]},"resources":[{"path":"${package.dir}/resources","output":"shared"}]}}`,
		"sdk/modern/locale/package.json":                       `{"version":"8.0.0.43","sencha":{"name":"locale","alternateName":["ext-locale"],"classpath":"${package.dir}/src","overrides":"${package.dir}/overrides/${package.locale}"}}`,
		"sdk/packages/base-lib/src/Base.js":                    `Ext.define('Lib.Base', { extend: 'Ext.Base' });`,
		"sdk/packages/base-lib/resources/base.txt":             "base resource",
		"sdk/packages/widget-lib/src/Widget.js":                `Ext.define('Lib.Widget', { extend: 'Lib.Base' });`,
		"sdk/packages/widget-lib/assets/terms.txt":             "widget asset",
		"sdk/packages/widget-lib/modern/resources/toolkit.txt": "modern resource",
		"sdk/packages/widget-lib/src/draw/AnimationParser.js": `Ext.define('Ext.draw.sprite.AnimationParser', function () {
    return { singleton: true, requires: ['Ext.draw.Color'] };
});`,
		"sdk/packages/widget-lib/modern/src/Toolkit.js": `Ext.define('Lib.Toolkit', { extend: 'Lib.Base' });`,
		"sdk/packages/widget-lib/resources/icon.txt":    "package resource",
		"sdk/modern/locale/overrides/de/Button.js":      `Ext.define('Ext.locale.de.Button', { override: 'Ext.Button' });`,
		"app.js": `Ext.application({name: 'Example', extend: 'Example.Application'}); Ext.require('Ext.*');`,
		"app/Application.js": `Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.view.Main'
});`,
		"app/view/Main.js": `Ext.define('Example.view.Main', {
    extend: 'Ext.Panel',
    requires: ['Lib.Widget', 'Lib.Toolkit', 'Ext.Button']
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
	if got, want := result.Packages, []string{"base-lib", "widget-lib", "locale"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved packages = %#v, want %#v", got, want)
	}
	if result.ResourceFiles != 4 {
		t.Fatalf("copied package resources = %d, want 4", result.ResourceFiles)
	}
	for name, want := range map[string]string{
		"base.txt":    "base resource",
		"icon.txt":    "package resource",
		"terms.txt":   "widget asset",
		"toolkit.txt": "modern resource",
	} {
		resource, err := os.ReadFile(filepath.Join(result.ResourcesPath, "shared", name))
		if err != nil || string(resource) != want {
			t.Errorf("package resource %s = %q, err = %v; want %q", name, resource, err, want)
		}
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
	if !strings.Contains(content, "// Source: sdk/modern/locale/overrides/de/Button.js") {
		t.Fatal("locale-specific package override was not included")
	}
}

func TestBuildLoadsStandardSDKPackageManifestFields(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":                                                         `{"name":"Example","namespace":"Example","toolkit":"classic","requires":["exporter"],"classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                                                   `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                                                 `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":                                   `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/packages/exporter/package.json":                               `{"name":"exporter","requires":["core"],"classpath":["${package.dir}/src","${package.dir}/${toolkit.name}/src"],"overrides":["${package.dir}/overrides","${package.dir}/${toolkit.name}/overrides"]}`,
		"sdk/packages/core/src/Core.js":                                    `Ext.define('Lib.Core', {});`,
		"sdk/packages/exporter/src/Exporter.js":                            `Ext.define('Lib.Exporter', { extend: 'Lib.Core' });`,
		"sdk/packages/exporter/src/Widget.js":                              `Ext.define('Lib.Widget', {});`,
		"sdk/packages/exporter/classic/src/ExporterClassic.js":             `Ext.define('Lib.ExporterClassic', {});`,
		"sdk/packages/exporter/overrides/WidgetOverride.js":                `Ext.define('Lib.overrides.Widget', { override: 'Lib.Widget', packageOverride: true });`,
		"sdk/packages/exporter/classic/overrides/WidgetClassicOverride.js": `Ext.define('Lib.overrides.WidgetClassic', { override: 'Lib.Widget', toolkitOverride: 'classic' });`,
		"sdk/packages/exporter/modern/overrides/WidgetModernOverride.js":   `Ext.define('Lib.overrides.WidgetModern', { override: 'Lib.Widget', toolkitOverride: 'modern' });`,
		"sdk/classic/classic/package.json":                                 `{"version":"8.0.0.43","sencha":{"name":"classic","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"app.js":                                                           `Ext.application({name: 'Example', extend: 'Example.Application'});`,
		"app/Application.js":                                               `Ext.define('Example.Application', { extend: 'Ext.app.Application', requires: ['Lib.Core', 'Lib.Exporter', 'Lib.ExporterClassic', 'Lib.Widget'] });`,
	})

	result, err := Build(root, "out")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := result.Packages, []string{"core", "exporter"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("resolved packages = %#v, want %#v", got, want)
	}
	bundle, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []string{
		"sdk/packages/exporter/src/Exporter.js",
		"sdk/packages/exporter/classic/src/ExporterClassic.js",
		"sdk/packages/exporter/src/Widget.js",
		"sdk/packages/exporter/overrides/WidgetOverride.js",
		"sdk/packages/exporter/classic/overrides/WidgetClassicOverride.js",
	} {
		if !strings.Contains(string(bundle), "// Source: "+source) {
			t.Errorf("bundle does not include package source %s", source)
		}
	}
	if strings.Contains(string(bundle), "WidgetModernOverride.js") {
		t.Fatal("modern package override was included in a classic build")
	}
	widget := strings.Index(string(bundle), "// Source: sdk/packages/exporter/src/Widget.js")
	for _, overrideName := range []string{
		"sdk/packages/exporter/overrides/WidgetOverride.js",
		"sdk/packages/exporter/classic/overrides/WidgetClassicOverride.js",
	} {
		override := strings.Index(string(bundle), "// Source: "+overrideName)
		if widget < 0 || override < 0 || widget >= override {
			t.Errorf("package override %s should follow its target class: widget=%d override=%d", overrideName, widget, override)
		}
	}
}

func TestBuildResolvesToolkitSpecificPackageRequirements(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":                                          `{"name":"Example","namespace":"Example","toolkit":"modern","requires":["common"],"classic":{"requires":["classic-addon"]},"modern":{"requires":["modern-addon"]},"builds":{"classic":{"toolkit":"classic","requires":["classic-profile-addon"]}},"classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                                    `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                                  `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":                    `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/classic/classic/package.json":                  `{"version":"8.0.0.43","sencha":{"name":"classic","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/modern/modern/package.json":                    `{"version":"8.0.0.43","sencha":{"name":"modern","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/packages/common/package.json":                  `{"version":"1.0.0","sencha":{"name":"common","classpath":"${package.dir}/src"}}`,
		"sdk/packages/classic-addon/package.json":           `{"version":"1.0.0","sencha":{"name":"classic-addon","classpath":"${package.dir}/src"}}`,
		"sdk/packages/modern-addon/package.json":            `{"version":"1.0.0","sencha":{"name":"modern-addon","classpath":"${package.dir}/src"}}`,
		"sdk/packages/classic-profile-addon/package.json":   `{"version":"1.0.0","sencha":{"name":"classic-profile-addon","classpath":"${package.dir}/src"}}`,
		"sdk/packages/common/src/Common.js":                 `Ext.define('Lib.Common', {});`,
		"sdk/packages/classic-addon/src/Classic.js":         `Ext.define('Lib.Classic', {});`,
		"sdk/packages/modern-addon/src/Modern.js":           `Ext.define('Lib.Modern', {});`,
		"sdk/packages/classic-profile-addon/src/Profile.js": `Ext.define('Lib.Profile', {});`,
		"app.js":       `Ext.application({name: 'Example'}); Ext.require(['Lib.Common', 'Lib.Classic', 'Lib.Modern', 'Lib.Profile']);`,
		"app/Empty.js": "",
	})

	classic, err := BuildWithToolkit(root, "out-classic", "classic")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := classic.Packages, []string{"common", "classic-addon", "classic-profile-addon"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("classic packages = %#v, want %#v", got, want)
	}
	classicBundle, err := os.ReadFile(classic.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(classicBundle), "sdk/packages/classic-profile-addon/src/Profile.js") || strings.Contains(string(classicBundle), "sdk/packages/modern-addon/src/Modern.js") {
		t.Fatalf("classic package selection failed: %s", classicBundle)
	}

	modern, err := BuildWithToolkit(root, "out-modern", "modern")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := modern.Packages, []string{"common", "modern-addon"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("modern packages = %#v, want %#v", got, want)
	}
}

func TestBuildFiltersPackagesForOtherToolkit(t *testing.T) {
	root := makeFixture(t, map[string]string{
		"app.json":                                `{"name":"Example","namespace":"Example","toolkit":"modern","requires":["classic-addon"],"classpath":["app"],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                          `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                        `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/classic/classic/package.json":        `{"version":"8.0.0.43","sencha":{"name":"classic","classpath":"${package.dir}/src"}}`,
		"sdk/modern/modern/package.json":          `{"version":"8.0.0.43","sencha":{"name":"modern","classpath":"${package.dir}/src"}}`,
		"sdk/packages/classic-addon/package.json": `{"version":"1.0.0","sencha":{"name":"classic-addon","toolkit":"classic","classpath":"${package.dir}/src"}}`,
		"sdk/packages/classic-addon/src/Addon.js": `Ext.define('Lib.Addon', {});`,
		"app.js":       `Ext.application({name: 'Example'}); Ext.require('Lib.Addon');`,
		"app/Empty.js": "",
	})

	_, err := BuildWithToolkit(root, "out-modern", "modern")
	if err == nil || !strings.Contains(err.Error(), `required Ext JS package "classic-addon" was not found`) {
		t.Fatalf("modern build error = %v, want classic-only package to be filtered", err)
	}

	classic, err := BuildWithToolkit(root, "out-classic", "classic")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(classic.Packages, []string{"classic-addon"}) {
		t.Fatalf("classic packages = %#v, want classic-addon", classic.Packages)
	}
	bundle, err := os.ReadFile(classic.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bundle), "sdk/packages/classic-addon/src/Addon.js") {
		t.Fatalf("classic-only package source was not bundled: %s", bundle)
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
