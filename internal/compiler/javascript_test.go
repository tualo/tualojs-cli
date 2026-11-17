package compiler

import (
	"reflect"
	"testing"
)

func TestParseExtMetadata(t *testing.T) {
	source := `
// Ext.define('Ignored.Comment', {});
Ext.define('Example.Application', {
    extend: 'Ext.app.Application',
    mainView: 'Example.view.Main',
    config: { text: "Ext.define('Ignored.String', {})" }
});
Ext.define('Example.view.Main', {
    extend: 'Ext.Panel',
    requires: ['Example.view.Greeting'],
    initialize: function () {
        Ext.create('Example.view.Created');
        Ext.create({ xclass: 'Example.view.ConfigCreated' });
        Ext.create('Example.view.Dynamic.' + suffix);
        Ext.create(getDynamicClassName());
    }
});
Ext.application({
    name: 'Example',
    extend: 'Example.Application'
});
Ext.create('Example.view.EntryCreated');
Ext.require('Example.view.Required');
Ext.syncRequire(['Example.view.SyncOne', 'Example.view.SyncTwo']);
`

	classes, appDependencies, sourceDependencies, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 2 {
		t.Fatalf("got %d declarations, want 2", len(classes))
	}
	if got, want := classes[0].name, "Example.Application"; got != want {
		t.Fatalf("first class = %q, want %q", got, want)
	}
	if got, want := classes[0].dependencies, []string{"Ext.app.Application", "Example.view.Main"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("application class dependencies = %#v, want %#v", got, want)
	}
	if got, want := classes[1].dependencies, []string{"Ext.Panel", "Example.view.Greeting"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("view dependencies = %#v, want %#v", got, want)
	}
	if got, want := appDependencies, []string{"Example.Application"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("application dependencies = %#v, want %#v", got, want)
	}
	if got, want := sourceDependencies, []string{
		"Example.view.Created",
		"Example.view.ConfigCreated",
		"Example.view.EntryCreated",
		"Example.view.Required",
		"Example.view.SyncOne",
		"Example.view.SyncTwo",
	}; !reflect.DeepEqual(got, want) {
		t.Fatalf("Ext.create dependencies = %#v, want %#v", got, want)
	}
}

func TestTokenizeReportsUnterminatedString(t *testing.T) {
	if _, err := tokenize(`Ext.define('Broken, {});`); err == nil {
		t.Fatal("expected unterminated string error")
	}
}

func TestParseMetadataWithRegexLiteralContainingQuotes(t *testing.T) {
	source := `
Ext.define('Example.Controller', {
    onError: function (message) {
        return message.match(/Column\s\'(.*?)\'(.*)/);
    }
});
`
	classes, _, _, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 1 || classes[0].name != "Example.Controller" {
		t.Fatalf("parsed declarations = %#v, want Example.Controller", classes)
	}
}

func TestParserIgnoresDelimiterCharactersInStringsAndRegex(t *testing.T) {
	source := `Ext.define('Example.Template', {
    extend: 'Ext.Base',
    valueRe: /^[{][a-z.]+[}]$/i,
    compile: function () {
        return this.template.indexOf('{') + this.template.indexOf('}');
    }
});`
	classes, _, _, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 1 || classes[0].name != "Example.Template" {
		t.Fatalf("parsed declarations = %#v, want Example.Template", classes)
	}
}

func TestParseXtypeAndWidgetReferences(t *testing.T) {
	source := `
Ext.define('Example.view.Greeting', {
    alias: ['widget.greeting', 'widget.hello']
});
Ext.define('Example.view.Main', {
    items: [{ xtype: 'greeting' }],
    showGreeting: function () { return Ext.widget('hello'); }
});
`
	classes, _, dependencies, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := classes[0].aliases, []string{"widget.greeting", "widget.hello"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("class aliases = %#v, want %#v", got, want)
	}
	if got, want := dependencies, []string{"widget.greeting", "widget.hello"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("widget dependencies = %#v, want %#v", got, want)
	}
}

func TestParseMixinsDependencies(t *testing.T) {
	source := `
Ext.define('Example.Controller', {
    mixins: {
        util: 'Example.mixins.ControllerTools',
        audit: 'Example.mixins.Audit'
    }
});
Ext.define('Example.Other', {
    mixins: ['Example.mixins.ArrayMixin']
});
`
	classes, _, _, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := classes[0].dependencies, []string{"Example.mixins.ControllerTools", "Example.mixins.Audit"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("object mixins = %#v, want %#v", got, want)
	}
	if got, want := classes[1].dependencies, []string{"Example.mixins.ArrayMixin"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("array mixins = %#v, want %#v", got, want)
	}
}

func TestParseUsesAsDeferredDependency(t *testing.T) {
	source := `Ext.define('Example.Main', { uses: ['Example.Optional'] });`
	classes, _, _, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := classes[0].uses, []string{"Example.Optional"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("uses = %#v, want %#v", got, want)
	}
	if len(classes[0].dependencies) != 0 {
		t.Fatalf("uses should not be an eager dependency: %#v", classes[0].dependencies)
	}
}

func TestParseControllerViewModelAndPluginAliases(t *testing.T) {
	source := `Ext.define('Example.view.Main', {
    controller: 'main',
    viewModel: 'main',
    plugins: ['gridexporter'],
    items: [{ plugins: [{ ptype: 'rowexpander' }] }]
});`
	_, _, dependencies, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"controller.main", "viewmodel.main", "plugin.gridexporter", "plugin.rowexpander"}
	if !reflect.DeepEqual(dependencies, want) {
		t.Fatalf("config alias dependencies = %#v, want %#v", dependencies, want)
	}
}

func TestParseAlternateClassNames(t *testing.T) {
	classes, _, _, err := parseExtMetadata(`Ext.define('Example.Button', { alternateClassName: ['Legacy.Button', 'Example.OldButton'] });`)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := classes[0].aliases, []string{"Legacy.Button", "Example.OldButton"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("alternate class names = %#v, want %#v", got, want)
	}
}

func TestParseLoaderPathMappings(t *testing.T) {
	source := `
Ext.Loader.setPath('Example.dashboard', './dashboard');
Ext.Loader.setPath({
    'Example.lazy': './lazy',
    'Example.other': './other'
});
`
	mappings, err := parseLoaderPaths(source)
	if err != nil {
		t.Fatal(err)
	}
	want := []loaderPathMapping{
		{Namespace: "Example.dashboard", Path: "./dashboard"},
		{Namespace: "Example.lazy", Path: "./lazy"},
		{Namespace: "Example.other", Path: "./other"},
	}
	if !reflect.DeepEqual(mappings, want) {
		t.Fatalf("loader mappings = %#v, want %#v", mappings, want)
	}
}

func TestParseExtDefineFactoryFunction(t *testing.T) {
	source := `
Ext.define('Example.draw.AnimationParser', function () {
    function compute(from, to) {
        return from + to;
    }
    return {
        singleton: true,
        requires: ['Example.draw.Color'],
        compute: compute
    };
});
`
	classes, _, _, err := parseExtMetadata(source)
	if err != nil {
		t.Fatal(err)
	}
	if len(classes) != 1 || classes[0].name != "Example.draw.AnimationParser" {
		t.Fatalf("parsed declarations = %#v, want factory-defined class", classes)
	}
	if got, want := classes[0].dependencies, []string{"Example.draw.Color"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("factory class dependencies = %#v, want %#v", got, want)
	}
}
