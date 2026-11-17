package compiler

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestPathListUnmarshalCommaSeparatedString(t *testing.T) {
	var paths pathList
	if err := json.Unmarshal([]byte(`"app, overrides, ,classic/src"`), &paths); err != nil {
		t.Fatal(err)
	}
	if got, want := []string(paths), []string{"app", "overrides", "classic/src"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}

func TestPathListUnmarshalStringArray(t *testing.T) {
	var paths pathList
	if err := json.Unmarshal([]byte(`["app", "classic/src"]`), &paths); err != nil {
		t.Fatal(err)
	}
	if got, want := []string(paths), []string{"app", "classic/src"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("paths = %#v, want %#v", got, want)
	}
}
