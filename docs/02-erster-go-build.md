# Schritt 2: Erster Go-Build

## Ziel

Ein erster vertikaler Schnitt liest das Modern-Toolkit-Fixture und erzeugt ohne Sencha Cmd ein deterministisches JavaScript-Bundle. Sencha Cmd bleibt für die Referenz-Builds zuständig, wird aber nicht zur Laufzeit des Go-Compilers aufgerufen.

## Aufruf

Vom Repository-Wurzelverzeichnis:

```sh
go run ./cmd/extjsc build \
  -app testdata/fixtures/minimal-modern \
  -out build-go \
  -source-map
```

Das Ergebnis liegt unter `testdata/fixtures/minimal-modern/build-go/app.js`. Relative Eingabe- und Ausgabepfade beziehen sich auf das jeweilige App-Verzeichnis. Die Ausgabe bricht ab, wenn sie eine Quell-JavaScript-Datei überschreiben würde.

## Implementiertes Verhalten

- Liest App-Name, Namespace, Classpath und JavaScript-Einträge aus `app.json`.
- Wendet `app.json.ignore`-Regexe auf App-Classpaths, JS-/CSS-Einträge und Resource-Kopien an; die Muster verwenden Go-`regexp`/RE2-Syntax.
- Liest `workspace.json`, validiert das Ext-JS-SDK gegen das feste Kompatibilitätsziel 8.0.0 und löst Paketnamen sowie `alternateName`-Aliase auf.
- Unterstützt `-mode development|production`; Sencha-`//<debug>`-Blöcke werden im Development-Modus erhalten und im Production-Modus entfernt, bevor Klassenabhängigkeiten ausgewertet werden.
- Folgt transitiven `app.json.requires`- und toolkit-/Build-Profil-`requires`- sowie Package-`sencha.requires`-Abhängigkeiten und nimmt deren Classpath- und Override-Dateien in die Klassenauflösung auf.
- Ersetzt `${toolkit.name}` in Classpath- und Override-Pfaden; universelle Apps wählen das Toolkit mit `-toolkit classic|modern`.
- Verlangt genau einen JavaScript-Einstieg mit `bundle: true`.
- Kopiert lokale zusätzliche JavaScript-Einträge und erhält die Reihenfolge von lokalen, gebündelten und Remote-Scripts aus `app.json.js`.
- Nimmt lokale `includeInBundle:true`-Scripts in das Go-Bundle auf.
- Kopiert lokale `app.json.resources`-Dateien und -Verzeichnisse mit optionalen `output`-Zielen ins Build-Verzeichnis.
- Kopiert lokale statische `app.json.css`-Dateien und bindet sie in der Preview ein; generierte Sass-CSS-Pfade bleiben beim Theme-CSS-Fallback.
- Sucht `.js`-Dateien in den Classpath-Verzeichnissen und erkennt `Ext.define`-Deklarationen sowohl mit Objekt- als auch mit Factory-Funktions-Konfiguration sowie `Ext.application`.
- Ermittelt `extend`, `requires`, `mainView`, `mixins`, `uses` und `xclass` aus statischen Konfigurationsobjekten sowie statische Klassennamen in `Ext.create`, `Ext.require` und `Ext.syncRequire`.
- Verknüpft Override-Klassen aus den in `app.json` angegebenen `overrides`-Pfaden mit ihrem Ziel; Overrides werden direkt nach einem tatsächlich geladenen Ziel ausgegeben.
- Löst Klassen-Aliase und `alternateClassName` aus `alias`, `xtype`, `controller`, `viewModel` und `plugins` anhand der jeweiligen Config-Referenzen auf.
- Mit `-normalize-let-const` werden `let`-/`const`-Deklarationen im direkten Funktions-Scope zu `var` normalisiert, einschließlich eines gleichnamigen Funktionsparameters. Standardmäßig bleiben sie unverändert.
- Löst `Namespace.*` und `Ext.*` gegen die App- und ausgewählten SDK-Classpaths auf. App-/Package-Klassen kommen ins Bundle; Framework-Klassen werden als Preview-Preloads ausgewiesen.
- Liest statische `Ext.Loader.setPath`-Mappings und scannt vorhandene App-Verzeichnisse für die gemappten Namespaces.
- Beachtet die Reihenfolge der Classpath-Pfade. Bei doppelten Klassendefinitionen gewinnt ein eindeutiger kanonischer Dateiname, ansonsten die frühere Classpath-Quelle; uneindeutige Fälle sind Fehler.
- Ordnet referenzierte App-Klassen deterministisch in Abhängigkeitsreihenfolge. Zyklen und nicht eindeutig auflösbare doppelte Klassennamen sind Fehler; fehlende App-Klassen werden als Warnungen ausgegeben, da Projekte Klassen über dynamische Loader-Pfade nachladen können.
- Schreibt die ausgewählten Quelldateien mit Herkunftskommentaren in ein Bundle; neben der dokumentierten Normalisierung bleiben Quellen weitgehend unverändert.
- `-source-map` erzeugt `app.js.map` mit zeilenbasierten Zuordnungen und `sourcesContent`.
- `-minify` entfernt token-sicher Kommentare und unnötigen Whitespace, ohne Strings, Regex-Literale oder Zeilenumbrüche zu verändern.
- Nutzt ausschließlich die Go-Standardbibliothek.

## Grenzen dieses Schritts

Der Metadatenscanner ist kein vollständiger ECMAScript-Parser. Er behandelt insbesondere keine dynamisch berechneten Anforderungen und keine beliebigen JavaScript-Ausdrücke. Die aktuelle ES-Transformation lässt `let` in Block-Scope und `for`-Köpfen unverändert, damit deren Semantik nicht durch blindes Ersetzen verloren geht. Das Bundle enthält referenzierte App- und Package-Klassen sowie den Einstieg; Framework-Runtime im Bundle, Package-Ressourcen, Sass-Kompilierung, allgemeine ES-Transformationen und Sencha-Microloader-Artefakte sind noch nicht implementiert. Minifizierung ist token-aware, aber noch ohne Sencha-Closure-Optimierungen. Source Maps ordnen JavaScript-Tokens mit UTF-16-Spalten zu, sind aber nicht transformations- oder Sencha-Closure-genau. Das Ergebnis ist deshalb ein erster Compiler-Schnitt und noch kein vollständiger Ersatz für `sencha app build`.

## Prüfungen

- `go test ./...` testet Metadaten-Erkennung, Klassenreihenfolge, Zyklenerkennung, Pfadschutz und Ausgabe.
- Der CLI-Aufruf oben wurde gegen `testdata/fixtures/minimal-modern/` ausgeführt und erzeugt acht Quelldateien im Bundle: Greeting, Created, WidgetOnly, Main, Main-Override, Application, Required und `app.js`.
- Der gleiche Fixture-Build mit Sencha Cmd 8.0.0.48 wurde erfolgreich ausgeführt; dessen `bootstrap.json` führt die App-Klassen und den Override ebenfalls auf.
