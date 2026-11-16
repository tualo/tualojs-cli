package compiler

import (
	"strings"
	"testing"
)

func TestTransformRenamesLexicalRedeclarationOfFunctionParameter(t *testing.T) {
	source := `function route(tablename) {
    let tablename = 'scss';
    const current = tablename;
    if (current) {
        let tablename = 'nested';
    }
    return 'let tablename';
}`
	got, err := transformJavaScript(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "var tablename = 'scss'") {
		t.Fatalf("parameter shadow was not normalized: %s", got)
	}
	if !strings.Contains(got, "let tablename = 'nested'") || !strings.Contains(got, "'let tablename'") {
		t.Fatalf("nested block or string content was incorrectly changed: %s", got)
	}
}

func TestTransformLowersFunctionScopeLetAndConst(t *testing.T) {
	source := `function route(tablename) {
    let current = tablename;
    const fixed = 'fixed';
    for (let index = 0; index < 1; index++) {}
    return {let: current};
}`
	got, err := transformJavaScript(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"var current = tablename", "var fixed = 'fixed'", "for (let index = 0"} {
		if !strings.Contains(got, expected) {
			t.Errorf("transformed source missing %q: %s", expected, got)
		}
	}
	if !strings.Contains(got, "{let: current}") {
		t.Errorf("property named let was transformed: %s", got)
	}
}
