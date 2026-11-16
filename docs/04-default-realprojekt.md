# Schritt 4: Realprojekt `default`

## Projekt und Versionshinweis

Der Einstieg liegt unter `testdata/ext-build/default/build/production/Tualo/`. Es handelt sich um eine Universal-App mit gemeinsamen Quellen (`both/`) und toolkit-spezifischen Verzeichnissen (`classic/`, `modern/`). `app.json` verwendet `${toolkit.name}` in Classpath- und Override-Pfaden und fordert in `app.js` `Tualo.*` sowie `Ext.*` an.

Das zugehörige `workspace.json` nennt aktuell Ext JS **8.0.0.43** und passt damit zum festgelegten Kompatibilitätsziel Ext JS **8.0.0**. `fb_wvd` weist eine ähnliche Projektstruktur auf und ergänzt unter anderem das Package `exporter`.

## Anpassungen im Go-Compiler

- CLI-Option `-toolkit modern|classic` für universelle Apps ohne einzelnes `toolkit`-Feld.
- Expansion von `${toolkit.name}` in `classpath` und `overrides`; nicht vorhandene optionale Override-Verzeichnisse werden übersprungen.
- Namespace-Wildcards wie `Tualo.*` laden deterministisch alle passenden App-Klassen; `Ext.*` wird gegen den ausgewählten Ext-SDK-Classpath aufgelöst.
- Package-Klassen mit `Ext.define('Name', function () { return {...}; })`, darunter `AnimationParser` und `TimingFunctions` aus `charts`, werden ebenfalls indiziert.
- Statische `mixins`-Arrays und -Objekte bilden Abhängigkeiten; dadurch steht `ControllerTools.js` vor `Controller.js` im Bundle.
- Classpath-Priorität entscheidet zwischen Toolkit- und gemeinsamen Klassen. Bei Backup-Dateien mit doppelten Klassennamen wird ein eindeutiger kanonischer Dateiname wie `Store.js` bevorzugt.
- Nicht gefundene App-Klassen werden als Warnungen ausgegeben, da sie über dynamische Loader-Pfade nachgeladen werden können.

## Go-Builds

Die Realprojekt-Quellen wurden nicht verändert. Die Ausgabe wurde außerhalb des Projekts erzeugt:

```sh
go run ./cmd/extjsc build --app testdata/ext-build/default/build/production/Tualo \
  --toolkit modern --out /tmp/default-modern-go

go run ./cmd/extjsc build --app testdata/ext-build/default/build/production/Tualo \
  --toolkit classic --out /tmp/default-classic-go
```

Beide Toolkit-Läufe lösen die Klassenabhängigkeiten auf und erzeugen Bundles: 409 Quelldateien für Modern und 602 für Classic. Der Go-Build meldet dynamisch geladene, nicht vorhandene Quellen als Warnungen.

## Noch nicht kompatibel: JavaScript-Transformation

Die Go-Ausgabe enthält derzeit unveränderte Quelldateien. `node --check` meldet im Realprojekt einen Syntaxfehler in `both/src/system/sass_css/Routes.js`: Der Funktionsparameter `tablename` wird im selben Scope nochmals mit `let` deklariert. Auch die vorhandene Sencha-Ausgabe enthält die Klasse, transformiert den Code aber unter anderem zu neu benannten `var`-Variablen; deren vorgebaute Classic-App besteht `node --check`.

Damit ist JavaScript-Parsing und -Transformation der nächste notwendige Schritt, bevor ein erzeugtes Realprojekt-Bundle als ausführbar gelten kann. Ein Sencha-Cmd-Referenzbuild wurde für dieses Projekt nicht ausgeführt.
