# Schritt 4: Realprojekt `default`

## Projekt und Versionshinweis

Der Einstieg liegt unter `testdata/ext-build/default/build/production/Tualo/`. Es handelt sich um eine Universal-App mit gemeinsamen Quellen (`both/`) und toolkit-spezifischen Verzeichnissen (`classic/`, `modern/`). `app.json` verwendet `${toolkit.name}` in Classpath- und Override-Pfaden und fordert in `app.js` `Tualo.*` sowie `Ext.*` an.

Das zugehörige `workspace.json` nennt aktuell Ext JS **8.0.0.43** und passt damit zum festgelegten Kompatibilitätsziel Ext JS **8.0.0**. `fb_wvd` weist eine ähnliche Projektstruktur auf und ergänzt unter anderem das Package `exporter`.

## Anpassungen im Go-Compiler

- CLI-Option `-toolkit modern|classic` für universelle Apps ohne einzelnes `toolkit`-Feld.
- Expansion von `${toolkit.name}` in `classpath` und `overrides`; nicht vorhandene optionale Override-Verzeichnisse werden übersprungen.
- Namespace-Wildcards wie `Tualo.*` laden deterministisch alle passenden App-Klassen; `Ext.*` wird gegen den ausgewählten Ext-SDK-Classpath aufgelöst.
- Package-Klassen mit `Ext.define('Name', function () { return {...}; })`, darunter `AnimationParser` und `TimingFunctions` aus `charts`, werden ebenfalls indiziert.
- `app.json.locale` expandiert locale-spezifische Package-Overrides; der Classic-Build nimmt damit die deutschen Ext-Locale-Dateien aus dem SDK auf.
- Statische `mixins`-Arrays und -Objekte bilden Abhängigkeiten; dadurch steht `ControllerTools.js` vor `Controller.js` im Bundle.
- Classpath-Priorität entscheidet zwischen Toolkit- und gemeinsamen Klassen. Bei Backup-Dateien mit doppelten Klassennamen wird ein eindeutiger kanonischer Dateiname wie `Store.js` bevorzugt.
- `app.json.ignore`-Muster werden auf die App-Quell- und Asset-Pfade angewendet.
- Nicht gefundene App-Klassen werden als Warnungen ausgegeben, da sie über dynamische Loader-Pfade nachgeladen werden können.
- Vorhandene, statisch gemappte `Ext.Loader.setPath`-Verzeichnisse werden zusätzlich durchsucht; externe Serverpfade bleiben dynamisch.

## Go-Builds

Die Realprojekt-Quellen wurden nicht verändert. Die Ausgabe wurde außerhalb des Projekts erzeugt:

```sh
go run ./cmd/extjsc build --app testdata/ext-build/default/build/production/Tualo \
  --toolkit modern --out /tmp/default-modern-go --normalize-let-const

go run ./cmd/extjsc build --app testdata/ext-build/default/build/production/Tualo \
  --toolkit classic --out /tmp/default-classic-go --normalize-let-const
```

Beide Toolkit-Läufe lösen die Klassenabhängigkeiten auf und erzeugen Bundles: 409 Quelldateien für Modern und 602 für Classic. Der Go-Build meldet dynamisch geladene, nicht vorhandene Quellen als Warnungen.

## JavaScript-Transformation

In `both/src/system/sass_css/Routes.js` wird der Funktionsparameter `tablename` im selben Scope nochmals mit `let` deklariert. Die Go-Pipeline senkt direkte Funktions-Scope-Deklarationen von `let`/`const` zu `var`, wie es auch die vorhandene Sencha-Ausgabe tut. Der erzeugte Classic-Build besteht damit `node --check`.

Diese Transformation ist bewusst eng begrenzt; eine allgemeine ECMAScript-/Ext-Klassen-Transformation ist weiterhin erforderlich. Ein Sencha-Cmd-Referenzbuild wurde für dieses Projekt nicht ausgeführt.
