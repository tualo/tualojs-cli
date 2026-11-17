# Schritt 5: Xtype- und Alias-Auflösung

## Verhalten

Ext-Klassen können über `alias` und `alternateClassName` registriert werden. Komponenten erscheinen häufig als `alias: 'widget.foo'` oder `xtype: 'foo'`; Controller, ViewModels und Plugins verwenden Aliase wie `controller.foo`, `viewmodel.foo` und `plugin.foo`. Verbraucher referenzieren sie über `xtype`, `Ext.widget()`, `Ext.createByAlias()`, `Ext.ClassManager.instantiateByAlias()`, `controller`, `viewModel`, `plugins` oder einen alternativen Klassennamen, oft ohne `requires`.

Der Go-Compiler indiziert statische `alias`, `xtype`- und `alternateClassName`-Eigenschaften pro `Ext.define` und nutzt die Alias-Tabelle zum Auflösen der genannten Config-Referenzen. Die gefundene Klasse wird vor ihrer verwendenden Klasse in das Bundle aufgenommen. Kurze Widget-, Controller-, ViewModel- und Plugin-Aliasnamen werden mit dem passenden Typpräfix normalisiert.

## Prüfungen

- Parser-Test für `alias`, `xtype` und `Ext.widget()`.
- Parser-Test für `Ext.createByAlias()` und `Ext.ClassManager.instantiateByAlias()`.
- Build-Test, der sicherstellt, dass die Widget-Klasse vor der Main-Klasse ausgegeben wird.
- Build-Test für Controller-, ViewModel- und Plugin-Aliasreferenzen.
- Im Modern-Toolkit-Fixture wird `WidgetOnly` ausschließlich über einen `xtype`-Verweis erreicht; Sencha Cmd führt die Klasse ebenfalls im Bootstrap auf.
- `go test ./...`, `go vet ./...` und `node --check` des Fixture-Bundles laufen erfolgreich.

## Grenzen

Dynamisch berechnete Xtypes, Alias-Kollisionen mit bedingten Plattformkonfigurationen und Framework-Aliase aus dem SDK sind noch nicht vollständig modelliert.
