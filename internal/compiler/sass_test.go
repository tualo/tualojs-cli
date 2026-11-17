package compiler

import (
	"os"
	"path/filepath"
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
	if err == nil || !strings.Contains(err.Error(), "Dart Sass executable") {
		t.Fatalf("compileClassSass error = %v, want actionable missing-executable error", err)
	}
}
