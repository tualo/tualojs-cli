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
	development, err := preprocessSencha(source, BuildModeDevelopment)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(development, "Example.DebugOnly") || !strings.Contains(development, "nested debug") {
		t.Fatalf("development build lost debug code: %s", development)
	}
	production, err := preprocessSencha(source, BuildModeProduction)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(production, "Example.DebugOnly") || strings.Contains(production, "nested debug") || !strings.Contains(production, "before();") || !strings.Contains(production, "after();") {
		t.Fatalf("production debug preprocessing produced unexpected output: %s", production)
	}
}

func TestPreprocessRejectsUnbalancedDebugMarkers(t *testing.T) {
	if _, err := preprocessSencha("//<debug>\nlog();\n", BuildModeProduction); err == nil {
		t.Fatal("expected unterminated debug block error")
	}
	if got, err := preprocessSencha("keep();\n//</debug>\n", BuildModeProduction); err != nil || !strings.Contains(got, "keep();") {
		t.Fatalf("standalone debug close should be ignored, got %q and error %v", got, err)
	}
}
