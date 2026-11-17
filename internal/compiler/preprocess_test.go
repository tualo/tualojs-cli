package compiler

import (
	"strings"
	"testing"
)

func TestPreprocessDebugBlocksByBuildMode(t *testing.T) {
	source := `before();
//<debug>
Ext.require('Example.DebugOnly');
//<debug>
console.log('nested debug');
//</debug>
//</debug>
after();
`
	development, err := preprocessSencha(source, BuildModeDevelopment, "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(development, "Example.DebugOnly") || !strings.Contains(development, "nested debug") {
		t.Fatalf("development build lost debug code: %s", development)
	}
	production, err := preprocessSencha(source, BuildModeProduction, "")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(production, "Example.DebugOnly") || strings.Contains(production, "nested debug") || !strings.Contains(production, "before();") || !strings.Contains(production, "after();") {
		t.Fatalf("production debug preprocessing produced unexpected output: %s", production)
	}
}

func TestPreprocessRejectsUnbalancedDebugMarkers(t *testing.T) {
	if _, err := preprocessSencha("//<debug>\nlog();\n", BuildModeProduction, ""); err == nil {
		t.Fatal("expected unterminated debug block error")
	}
	if got, err := preprocessSencha("keep();\n//</debug>\n", BuildModeProduction, ""); err != nil || !strings.Contains(got, "keep();") {
		t.Fatalf("standalone debug close should be ignored, got %q and error %v", got, err)
	}
}

func TestPreprocessToolkitConditionBlocks(t *testing.T) {
	source := `before();
// <if classic>
classicOnly();
// </if>
// <if modern>
modernOnly();
// </if>
after();
`
	for _, test := range []struct {
		toolkit string
		present string
		absent  string
	}{
		{toolkit: "classic", present: "classicOnly();", absent: "modernOnly();"},
		{toolkit: "modern", present: "modernOnly();", absent: "classicOnly();"},
	} {
		got, err := preprocessSencha(source, BuildModeProduction, test.toolkit)
		if err != nil {
			t.Fatalf("preprocess for %s: %v", test.toolkit, err)
		}
		if !strings.Contains(got, test.present) || strings.Contains(got, test.absent) {
			t.Errorf("preprocess for %s produced unexpected output: %s", test.toolkit, got)
		}
		if strings.Contains(got, "<if") || strings.Contains(got, "</if>") {
			t.Errorf("preprocess for %s retained conditional markers: %s", test.toolkit, got)
		}
	}
}

func TestPreprocessToolkitConditionNestedWithDebug(t *testing.T) {
	source := `// <if classic>
classic();
//<debug>
classicDebug();
//</debug>
// </if>
`
	development, err := preprocessSencha(source, BuildModeDevelopment, "classic")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(development, "classic();") || !strings.Contains(development, "classicDebug();") {
		t.Fatalf("development preprocessing lost active code: %s", development)
	}
	production, err := preprocessSencha(source, BuildModeProduction, "classic")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(production, "classic();") || strings.Contains(production, "classicDebug();") {
		t.Fatalf("production preprocessing retained debug code or lost toolkit code: %s", production)
	}
}

func TestPreprocessBrowserBuildConditions(t *testing.T) {
	source := `before();
//<if nonBrowser>
nonBrowserOnly();
//</if nonBrowser>
//<if legacyBrowser>
legacyBrowserOnly();
//</if legacyBrowser>
after();
`
	got, err := preprocessSencha(source, BuildModeProduction, "classic")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "nonBrowserOnly") || strings.Contains(got, "legacyBrowserOnly") {
		t.Fatalf("browser build retained non-browser or legacy-browser code: %s", got)
	}
	if !strings.Contains(got, "before();") || !strings.Contains(got, "after();") {
		t.Fatalf("browser build lost surrounding code: %s", got)
	}
}

func TestPreprocessFeatureMarkers(t *testing.T) {
	source := `before();
//<feature legacyBrowser>
legacyBrowserOnly();
//</feature>
//<feature classSystem.statics>
classSystemFeature();
//</feature>
//<feature logger>
loggerFeature();
//</feature>
after();
`
	got, err := preprocessSencha(source, BuildModeProduction, "classic")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(got, "legacyBrowserOnly") {
		t.Fatalf("modern browser build retained legacy-browser feature code: %s", got)
	}
	for _, expected := range []string{"before();", "classSystemFeature();", "loggerFeature();", "after();"} {
		if !strings.Contains(got, expected) {
			t.Errorf("feature preprocessing lost enabled content %q: %s", expected, got)
		}
	}
	if strings.Contains(got, "<feature") || strings.Contains(got, "</feature>") {
		t.Fatalf("feature preprocessing retained marker comments: %s", got)
	}
}

func TestPreprocessRejectsUnsupportedOrUnresolvedToolkitConditions(t *testing.T) {
	for _, test := range []struct {
		source, toolkit string
	}{
		{source: "// <if desktop>\ndesktop();\n// </if>\n", toolkit: "classic"},
		{source: "// <if classic>\nclassic();\n// </if>\n", toolkit: ""},
		{source: "// </if>\n", toolkit: "classic"},
		{source: "// <if classic>\nclassic();\n", toolkit: "classic"},
		{source: "//<feature desktop>\ndesktopOnly();\n//</feature>\n", toolkit: "classic"},
		{source: "//<feature legacyBrowser>\nlegacyOnly();\n", toolkit: "classic"},
	} {
		if _, err := preprocessSencha(test.source, BuildModeProduction, test.toolkit); err == nil {
			t.Errorf("preprocessSencha(%q, %q) succeeded, want conditional marker error", test.source, test.toolkit)
		}
	}
}
