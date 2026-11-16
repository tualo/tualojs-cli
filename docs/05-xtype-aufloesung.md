# Schritt 5: Xtype- und Alias-Auflösung

## Verhalten

Ext-Komponenten können über `alias: 'widget.foo'` oder `xtype: 'foo'` registriert werden. Verbraucher referenzieren sie häufig als `xtype: 'foo'` in einer Konfiguration oder mit `Ext.widget('foo')`, ohne `requires` anzugeben.

Der Go-Compiler indiziert statische `alias`- und `xtype`-Eigenschaften pro `Ext.define` und nutzt die Alias-Tabelle zum Auflösen von statischen `xtype`-Feldern und `Ext.widget()`-Aufrufen. Die gefundene Komponentenklasse wird vor ihrer verwendenden Klasse in das Bundle aufgenommen. Xtype-Aliase werden als `widget.<name>` normalisiert.

## Prüfungen

- Parser-Test für `alias`, `xtype` und `Ext.widget()`.
- Build-Test, der sicherstellt, dass die Widget-Klasse vor der Main-Klasse ausgegeben wird.
- Im Modern-Toolkit-Fixture wird `WidgetOnly` ausschließlich über einen `xtype`-Verweis erreicht; Sencha Cmd führt die Klasse ebenfalls im Bootstrap auf.
- `go test ./...`, `go vet ./...` und `node --check` des Fixture-Bundles laufen erfolgreich.

## Grenzen

Dynamisch berechnete Xtypes, Alias-Kollisionen mit bedingten Plattformkonfigurationen und Framework-Aliase aus dem SDK sind noch nicht vollständig modelliert.
