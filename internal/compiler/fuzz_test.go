package compiler

import "testing"

func FuzzParseExtMetadata(f *testing.F) {
	f.Add("Ext.define('App.view.Main', { extend: 'Ext.Panel', requires: ['App.util.Helper'] });")
	f.Add("Ext.define(\"App.Bad\", { foo: [ }")
	f.Add("// arbitrary UTF-8: Grüße 🧪")
	f.Fuzz(func(t *testing.T, source string) {
		_, _, _, _ = parseExtMetadata(source)
	})
}

func FuzzParseLoaderPaths(f *testing.F) {
	f.Add("Ext.Loader.setPath('App', './app');")
	f.Add("Ext.Loader.setPath({ App: './app', 'App.lib': './lib' });")
	f.Add("Ext.Loader.setPath(")
	f.Fuzz(func(t *testing.T, source string) {
		_, _ = parseLoaderPaths(source)
	})
}

func FuzzPreprocessSenchaForTarget(f *testing.F) {
	f.Add("//<debug>\nconsole.log('debug');\n//</debug>", BuildModeDevelopment, "modern", "")
	f.Add("//<if toolkit == 'modern'>\nExt.define('App.Main', {});\n//</if>", BuildModeProduction, "modern", "")
	f.Add("//</debug>", BuildModeProduction, "classic", "desktop")
	f.Fuzz(func(t *testing.T, source, mode, toolkit, platform string) {
		_, _ = preprocessSenchaForTarget(source, mode, toolkit, platform)
	})
}

func FuzzStripJSONComments(f *testing.F) {
	f.Add([]byte(`{"name":"app","path":"//not-a-comment"}`))
	f.Add([]byte("{\n // comment\n \"key\": /* inline */ \"value\"\n}"))
	f.Add([]byte{0xff, 0xfe, '{', '/', '/'})
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = stripJSONComments(data)
	})
}
