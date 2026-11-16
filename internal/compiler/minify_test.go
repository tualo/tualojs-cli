package compiler

import (
	"strings"
	"testing"
)

func TestMinifyRemovesWhitespaceWithoutChangingTokens(t *testing.T) {
	source := `/*! license */
// ordinary comment
var a = 1 + +2;
var text = 'x // stays in string';
var pattern = /a\/b/;
`
	got, err := minifyJavaScript(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"/*! license */", "var a=1+ +2;", "'x // stays in string'", "/a\\/b/"} {
		if !strings.Contains(got, expected) {
			t.Errorf("minified source missing %q: %s", expected, got)
		}
	}
	if strings.Contains(got, "ordinary comment") {
		t.Fatalf("ordinary comment was retained: %s", got)
	}
	if strings.Count(got, "\n") != strings.Count(source, "\n") {
		t.Fatalf("minifier changed line count: source=%d output=%d", strings.Count(source, "\n"), strings.Count(got, "\n"))
	}
}

func TestMinifyPreservesUnsignedShiftOperator(t *testing.T) {
	minified, err := minifyJavaScript("var hue = Math.floor(((hash >>> 0) * 137.508) % 360);")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(minified, "hash>>>0") {
		t.Fatalf("unsigned shift operator changed during minification: %s", minified)
	}
}
