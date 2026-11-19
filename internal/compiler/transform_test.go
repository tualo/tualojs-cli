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

func TestUpgradeES5UsesConstAndLetForSafeFunctionLocals(t *testing.T) {
	source := `function update(value) {
    var stable = value + 1;
    var count = 0;
    count++;
    return stable + count;
}`
	got, err := upgradeES5JavaScript(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"const stable = value + 1;", "let count = 0;"} {
		if !strings.Contains(got, expected) {
			t.Errorf("upgraded source missing %q: %s", expected, got)
		}
	}
}

func TestUpgradeES5LeavesSemanticallyUncertainDeclarationsAlone(t *testing.T) {
	source := `var globalValue = 1;
function update(parameter) {
    var shadowed = parameter;
    var loopIndex = 0;
    for (var loop = 0; loop < 1; loop++) {}
    var priorRead = typeof later;
    var later = 2;
    if (parameter) { var nestedBlock = 3; }
    var first = 1, second = 2;
    var captured = 4;
    var destructured = 0;
    [destructured] = [1];
    var asiRisk = 0
    asiRisk = 1;
    function readCaptured() { return captured; }
    return shadowed + loopIndex + loop + priorRead + later + nestedBlock + first + second + destructured + asiRisk + readCaptured();
}`
	got, err := upgradeES5JavaScript(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "var globalValue = 1") || !strings.Contains(got, "var loop = 0") || !strings.Contains(got, "var nestedBlock = 3") || !strings.Contains(got, "var first = 1, second = 2") {
		t.Fatalf("global, loop, block, or multi-declarator var was upgraded: %s", got)
	}
	if !strings.Contains(got, "var later = 2") || !strings.Contains(got, "var captured = 4") {
		t.Fatalf("hoisted or closure-captured binding was upgraded: %s", got)
	}
	if !strings.Contains(got, "let destructured = 0") || !strings.Contains(got, "var asiRisk = 0") {
		t.Fatalf("destructuring write was not classified as mutable or ASI-sensitive declaration was upgraded: %s", got)
	}
	if !strings.Contains(got, "const shadowed") || !strings.Contains(got, "const loopIndex") {
		t.Fatalf("safe initialized function locals were not upgraded: %s", got)
	}
}

func TestUpgradeES5DoesNotChangeFunctionParameterBindings(t *testing.T) {
	source := `function update(value) { var value = 2; return value; }`
	got, err := upgradeES5JavaScript(source)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got, "var value = 2") {
		t.Fatalf("parameter redeclaration was upgraded: %s", got)
	}
}
