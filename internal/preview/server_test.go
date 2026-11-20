package preview

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"tualojs/internal/compiler"
)

func TestPreviewServesShellFrameworkThemeAndBundle(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"app.json":                       `{"name":"PreviewApp","namespace":"PreviewApp","toolkit":"modern","theme":"theme-material","classpath":["app"],"resources":[{"path":"app-assets","output":"shared"}],"css":[{"path":"styles/app.css"}],"js":[{"path":"vendor/pre.js"},{"path":"app.js","bundle":true},{"path":"https://cdn.example/after.js","remote":true}]}`,
		"workspace.json":                 `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":               `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json": `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/packages/core/src/mixin/Factoryable.js":                       `Ext.define('Ext.mixin.Factoryable', {});`,
		"sdk/modern/modern/package.json":                                   `{"version":"8.0.0.43","sencha":{"name":"modern","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/modern/modern/src/scroll/Scroller.js":                         `Ext.define('Ext.scroll.Scroller', { mixins: ['Ext.mixin.Factoryable'] });`,
		"sdk/build/ext-modern-all-debug.js":                                "window.Ext = window.Ext || {};",
		"sdk/build/modern/theme-material/resources/theme-material-all.css": "body { color: black; }",
		"app.js":              `Ext.application({name: 'PreviewApp', extend: 'PreviewApp.Main'});`,
		"app/Main.js":         `Ext.define('PreviewApp.Main', { extend: 'Ext.Panel', requires: ['Ext.scroll.Scroller'] });`,
		"app/Main.scss":       `.preview-sass { color: blue; }`,
		"app-assets/logo.txt": "preview logo",
		"styles/app.css":      "body { background: red; }",
		"vendor/pre.js":       "window.preload = true;",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sassBinary := filepath.Join(t.TempDir(), "sass")
	if err := os.WriteFile(sassBinary, []byte("#!/bin/sh\nprintf '.preview-sass{color:blue}\\n'\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SASS", sassBinary)

	result, err := compiler.BuildWithSourceMap(root, "build-go", "modern", false, compiler.BuildModeDevelopment, true)
	if err != nil {
		t.Fatal(err)
	}
	if !contains(result.FrameworkClasses, "Ext.scroll.Scroller") {
		t.Fatalf("framework preload classes = %#v, want Ext.scroll.Scroller", result.FrameworkClasses)
	}
	if _, exists := result.FrameworkFiles["Ext.scroll.Scroller"]; !exists {
		t.Fatal("framework source map is missing Ext.scroll.Scroller")
	}
	state, err := makePreviewState(result)
	if err != nil {
		t.Fatal(err)
	}
	var current atomic.Pointer[previewState]
	current.Store(state)
	server := httptest.NewServer(newHandler(root, &current))
	defer server.Close()

	checks := []struct {
		path    string
		content string
	}{
		{path: "/", content: "/sdk/ext.js"},
		{path: "/", content: "/sdk/theme/theme-material-all.css"},
		{path: "/", content: "/app.js"},
		{path: "/", content: "Ext.scroll.Scroller"},
		{path: "/app.js", content: "PreviewApp.Main"},
		{path: "/", content: "/js/vendor/pre.js"},
		{path: "/", content: "https://cdn.example/after.js"},
		{path: "/js/vendor/pre.js", content: "window.preload"},
		{path: "/app.js", content: "sourceMappingURL=app.js.map"},
		{path: "/app.js.map", content: `"version":3`},
		{path: "/sources/app/Main.js", content: "PreviewApp.Main"},
		{path: "/resources/shared/logo.txt", content: "preview logo"},
		{path: "/", content: "/css/styles/app.css"},
		{path: "/css/styles/app.css", content: "background: red"},
		{path: "/", content: "/css/app-sass.css"},
		{path: "/css/app-sass.css", content: ".preview-sass{color:blue}"},
		{path: "/sdk/ext.js", content: "window.Ext"},
		{path: "/", content: "Ext.scroll.Scroller"},
		{path: "/sdk/classes/mixin/Factoryable.js", content: "Ext.mixin.Factoryable"},
		{path: "/sdk/classes/scroll/Scroller.js", content: "Ext.scroll.Scroller"},
		{path: "/sdk/theme/theme-material-all.css", content: "color: black"},
		{path: "/__build", content: state.buildID},
	}
	for _, check := range checks {
		response, err := http.Get(server.URL + check.path)
		if err != nil {
			t.Fatalf("GET %s: %v", check.path, err)
		}
		body, readErr := io.ReadAll(response.Body)
		response.Body.Close()
		if readErr != nil {
			t.Fatalf("read %s: %v", check.path, readErr)
		}
		if response.StatusCode != http.StatusOK {
			t.Errorf("GET %s status = %d, want 200", check.path, response.StatusCode)
		}
		if !strings.Contains(string(body), check.content) {
			t.Errorf("GET %s body does not contain %q", check.path, check.content)
		}
	}
}

func contains(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func TestPreviewFallsBackToSenchaGeneratedThemeCSS(t *testing.T) {
	root := t.TempDir()
	sdkRoot := filepath.Join(root, "ext")
	frameworkJS := filepath.Join(sdkRoot, "build", "ext-all-debug.js")
	generatedResources := filepath.Join(root, "build", "development", "SampleForm", "resources")
	generatedCSS := filepath.Join(generatedResources, "SampleForm-all.css")
	for path, content := range map[string]string{
		frameworkJS:  "window.Ext = window.Ext || {};",
		generatedCSS: "body { color: blue; }",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	state, err := makePreviewState(compiler.BuildResult{
		AppName:              "SampleForm",
		AppDir:               root,
		FrameworkSDKPath:     sdkRoot,
		FrameworkRuntimePath: frameworkJS,
		Toolkit:              "classic",
		Theme:                "theme-triton",
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.themeResources != generatedResources || state.themeCSS != generatedCSS || state.themeCSSName != "SampleForm-all.css" {
		t.Fatalf("preview theme fallback = %#v", state)
	}
}

func TestPreviewPrefersCombinedSenchaAppCSS(t *testing.T) {
	root := t.TempDir()
	sdkRoot := filepath.Join(root, "ext")
	frameworkJS := filepath.Join(sdkRoot, "build", "ext-all-debug.js")
	sdkResources := filepath.Join(sdkRoot, "build", "classic", "theme-triton", "resources")
	combinedResources := filepath.Join(root, "build", "production", "SampleForm", "classic", "resources")
	for path, content := range map[string]string{
		frameworkJS: "window.Ext = window.Ext || {};",
		filepath.Join(sdkResources, "theme-triton-all.css"):    "body { color: gray; }",
		filepath.Join(combinedResources, "SampleForm-all.css"): "body { color: blue; }",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	state, err := makePreviewState(compiler.BuildResult{
		AppName:              "SampleForm",
		AppDir:               root,
		FrameworkSDKPath:     sdkRoot,
		FrameworkRuntimePath: frameworkJS,
		Toolkit:              "classic",
		Theme:                "theme-triton",
		Mode:                 "production",
	})
	if err != nil {
		t.Fatal(err)
	}
	if state.themeResources != combinedResources || state.themeCSSName != "SampleForm-all.css" {
		t.Fatalf("preview did not choose generated app CSS: %#v", state)
	}
}

func TestWatcherRebuildsAfterSourceChanges(t *testing.T) {
	root := t.TempDir()
	files := map[string]string{
		"app.json":                          `{"name":"PreviewApp","namespace":"PreviewApp","toolkit":"modern","theme":"theme-material","classpath":["app"],"resources":[{"path":"assets","output":"shared"}],"js":[{"path":"app.js","bundle":true}]}`,
		"workspace.json":                    `{"frameworks":{"ext":{"path":"sdk","version":"8.0.0.43"}}}`,
		"sdk/package.json":                  `{"name":"ext","version":"8.0.0.43","compatVersion":"8.0.0"}`,
		"sdk/packages/core/package.json":    `{"version":"8.0.0.43","sencha":{"name":"core","classpath":"${package.dir}/src"}}`,
		"sdk/modern/modern/package.json":    `{"version":"8.0.0.43","sencha":{"name":"modern","requires":["core"],"classpath":"${package.dir}/src"}}`,
		"sdk/build/ext-modern-all-debug.js": "window.Ext = window.Ext || {};",
		"sdk/build/modern/theme-material/resources/theme-material-all.css": "body {}",
		"app.js":           `Ext.application({name: 'PreviewApp', extend: 'PreviewApp.Main'});`,
		"app/Main.scss":    `@import "colors"; .preview-main { color: $preview-color; }`,
		"app/_colors.scss": `$preview-color: before;`,
		"app/Main.js": `Ext.define('PreviewApp.Main', {
    extend: 'Ext.Panel',
    title: 'before',
    //<if phone>
    requires: ['PreviewApp.PhoneOnly'],
    //</if phone>
    //<if desktop>
    requires: ['PreviewApp.DesktopOnly'],
    //</if desktop>
});`,
		"app/PhoneOnly.js":   `Ext.define('PreviewApp.PhoneOnly', {});`,
		"app/DesktopOnly.js": `Ext.define('PreviewApp.DesktopOnly', {});`,
		"assets/logo.txt":    "before",
	}
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	sassBinary := filepath.Join(t.TempDir(), "sass")
	sassScript := `#!/bin/sh
input=$(cat "$SASS_SOURCE" "${SASS_SOURCE%/*}/_colors.scss")
case "$input" in
    *after*) printf '.preview-main{color:after}\n' ;;
    *) printf '.preview-main{color:before}\n' ;;
esac
`
	if err := os.WriteFile(sassBinary, []byte(sassScript), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("SASS", sassBinary)
	t.Setenv("SASS_SOURCE", filepath.Join(root, "app", "Main.scss"))
	result, err := compiler.BuildWithTargetOptions(root, "build-go", "modern", "phone", false, compiler.BuildModeDevelopment, false, false)
	if err != nil {
		t.Fatal(err)
	}
	initial, err := os.ReadFile(result.OutputPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(initial), "app/PhoneOnly.js") || strings.Contains(string(initial), "app/DesktopOnly.js") {
		t.Fatalf("initial preview build ignored its phone platform: %s", initial)
	}
	initialCSS, err := os.ReadFile(result.StyleSheets[len(result.StyleSheets)-1].Path)
	if err != nil || !strings.Contains(string(initialCSS), "color:before") {
		t.Fatalf("initial Sass output = %q, err=%v", initialCSS, err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var current atomic.Pointer[previewState]
	current.Store(&previewState{result: result})
	done := make(chan struct{})
	go func() {
		defer close(done)
		watch(ctx, Config{AppDir: root, OutDir: "build-go", Toolkit: "modern", Platform: "phone", Interval: 20 * time.Millisecond}, &current)
	}()
	time.Sleep(40 * time.Millisecond)
	mainPath := filepath.Join(root, "app", "Main.js")
	if err := os.WriteFile(mainPath, []byte(`Ext.define('PreviewApp.Main', {
    extend: 'Ext.Panel',
    title: 'after',
    //<if phone>
    requires: ['PreviewApp.PhoneOnly'],
    //</if phone>
    //<if desktop>
    requires: ['PreviewApp.DesktopOnly'],
    //</if desktop>
});`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "app", "_colors.scss"), []byte(`$preview-color: after;`), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	built := false
	for time.Now().Before(deadline) {
		updated, err := os.ReadFile(result.OutputPath)
		updatedCSS, cssErr := os.ReadFile(result.StyleSheets[len(result.StyleSheets)-1].Path)
		if err == nil && cssErr == nil && strings.Contains(string(updated), "title: 'after'") && strings.Contains(string(updated), "app/PhoneOnly.js") && !strings.Contains(string(updated), "app/DesktopOnly.js") && strings.Contains(string(updatedCSS), "color:after") {
			built = true
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	if !built {
		updated, _ := os.ReadFile(result.OutputPath)
		t.Fatalf("watcher did not rebuild bundle; before=%q after=%q", initial, updated)
	}
	assetPath := filepath.Join(root, "assets", "logo.txt")
	if err := os.WriteFile(assetPath, []byte("after"), 0o644); err != nil {
		t.Fatal(err)
	}
	deadline = time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		updated, err := os.ReadFile(filepath.Join(result.ResourcesPath, "shared", "logo.txt"))
		if err == nil && string(updated) == "after" {
			cancel()
			<-done
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	cancel()
	<-done
	t.Fatal("watcher did not recopy changed static resource")
}
