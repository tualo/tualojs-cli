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

func TestTransformLeavesOrdinaryLetAndConstDeclarations(t *testing.T) {
	source := `function route(tablename) {
    let current = tablename;
    const fixed = 'fixed';
    return current + fixed;
}`
	got, err := transformJavaScript(source)
	if err != nil {
		t.Fatal(err)
	}
	if got != source {
		t.Fatalf("ordinary lexical declarations changed: got %q, want %q", got, source)
	}
}
