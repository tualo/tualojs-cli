# Schritt 3: Klassen-Overrides

## Verhalten

Ext JS Overrides werden über `Ext.define` mit einer `override`-Eigenschaft deklariert, zum Beispiel:

```javascript
Ext.define('MyApp.overrides.Panel', {
    override: 'MyApp.view.Main',
    // zusätzliche oder ersetzte Klassenmitglieder
});
```

Der Compiler liest Override-Dateien aus den Pfaden in `app.json` unter `overrides`. Er nimmt ein Override nur dann in das Bundle auf, wenn dessen Zielklasse über den App-Einstieg oder eine weitere Abhängigkeit geladen wird. Die Zieldatei steht im Bundle vor ihrem Override.

Framework-Ziele wie `Ext.Panel` können ebenfalls Overrides haben: sobald ein solches Ziel als Klassenabhängigkeit auftritt, kann der zugehörige App-Override angefügt werden. Der Framework-Code selbst wird in diesem Schritt weiterhin nicht gebündelt.

## Prüfungen

- Das Fixture enthält `overrides/Main.js`, das `GoCompilerFixture.view.main.Main` erweitert.
- Der Go-Integrationstest prüft die Reihenfolge Ziel vor Override und stellt sicher, dass ein Override für ein nicht geladenes Ziel ausgeschlossen bleibt.
- `sencha app build --dev` dient als Referenz für die Fixture-Klasse und den App-Override.

## Grenzen

Das aktuelle Modell behandelt statisch benannte `Ext.define`-Overrides. Bedingte Overrides, Toolkit-/Plattformfilter, Package-Overrides und dynamisch berechnete Ziele sind noch nicht implementiert.
