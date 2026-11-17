# Schritt 11: Assets und Build-Ausgabe

## Verhalten

Der Compiler verarbeitet lokale Einträge aus `app.json.resources` als Pfadstrings oder Objekte mit `path`, `output` und optional `remote`. Verzeichnisse werden rekursiv kopiert nach `<out>/resources/<output>`; ohne `output` wird ein stabiler relativer Zielpfad aus dem Quellpfad gebildet. Remote-Ressourcen werden nicht kopiert.

Lokale statische Stylesheets aus `app.json.css` werden an ihrem relativen Pfad ins Build-Verzeichnis kopiert und in der Preview eingebunden. Remote-CSS wird direkt verlinkt. Dynamische Sencha-Platzhalter wie `${build.out.css.path}` werden übersprungen, da sie auf Sass-/Theme-Build-Ausgaben verweisen.

Zusätzliche `app.json.js`-Einträge mit `bundle:false` werden ebenfalls in ihrer Manifest-Reihenfolge kopiert und geladen. `remote:true`-Einträge werden nicht kopiert und als externe Script-URL in der Preview beibehalten.

Ein lokales Script mit `includeInBundle:true` wird statt als separate Datei in das Go-Bundle aufgenommen. Seine statischen Ext-Klassendeklarationen nehmen an der normalen Dependency-Auflösung teil.

Build-Pfade bleiben auf das App-Verzeichnis beschränkt, Output-Pfade auf den Ressourcen-Zielordner. Symlinks werden übersprungen und doppelte Zielpfade mit unterschiedlichem Inhalt als Konflikt gemeldet. Resource-Quellpfade werden vom Watcher beobachtet. Die Preview liefert generierte Ressourcen unter `/resources/` aus.

Der Sencha-Platzhalter `${build.id}` wird in Ressourcenpfaden anhand vorhandener `build/<mode>/<app-name>`-Verzeichnisse aufgelöst. Für Development-Builds wird zunächst das Development-Verzeichnis, danach ein vorhandener Production-Build verwendet; fehlt das referenzierte Verzeichnis, wird dieser Resource-Eintrag übersprungen.

Transitive SDK-Packages können Ressourcen als `sencha.resources`-Einträge oder als `sencha.resource.paths`-Pfade deklarieren. Beide Formen werden nach der Package-Auflösung in dasselbe `resources`-Ausgabeziel unter `shared/` kopiert; toolkit-spezifische Pfade werden für das ausgewählte Toolkit expandiert, fehlende Verzeichnisse übersprungen.

SCSS-Dateien neben gleichnamigen JavaScript-Klassen in den App-Classpaths und Override-Pfaden werden mit Dart Sass über einen deterministischen gemeinsamen Sass-Einstieg kompiliert. Dadurch teilen die enthaltenen Class-SCSS-Dateien normale Sass-Variablen. Das Ergebnis wird in `<out>/<toolkit>/resources/app-sass.css` geschrieben. Der Toolkit-Unterordner hält relative CSS-Asset-URLs kompatibel mit dem Sencha-Resource-Pfad; die Preview stellt dieselbe Datei unter `/css/app-sass.css` bereit und lädt sie nach dem vorhandenen Theme-CSS. `SASS` kann den Pfad zum ausführbaren `sass`-Programm überschreiben; standardmäßig wird `sass` über `PATH` gesucht. `-minify` verwendet Dart Sass' komprimierte CSS-Ausgabe. Die SCSS-Dateien werden vom Watcher beobachtet.

## Prüfungen

- Integrationstest für Verzeichnis- und Einzeldatei-Ressourcen sowie `output`-Ziele.
- Integrationstest für `${build.id}/resources` aus einem vorherigen Sencha-Build.
- Preview-HTTP-Test für den Abruf einer kopierten Resource.
- Build- und Preview-Test für ein lokales `app.json.css`-Stylesheet.
- Build-Test für ein per `includeInBundle` eingebettetes JavaScript-Hilfsscript.
- Fake-SDK-Integrationstest für `sencha.resources` und String-/Array-Formen von `sencha.resource.paths`, einschließlich toolkit-spezifischer Pfade.
- Build-Test für SCSS neben gleichnamigen Klassen, CSS-Konkatenation, Watch-Pfaden und Einbindung in die Stylesheet-Ausgabe.
- `testdata/sample` wird aus seinem Sencha-`resources`-Ordner nach `build-go/resources/shared` kopiert.

## Grenzen

Sencha-Fashion-Themes sowie Sass-Variablen und Mixins aus globalen Theme-/Package-Einstiegspunkten werden noch nicht nachgebildet. Klassenstylesheets mit nicht auflösbaren Fashion-Mixins, -Variablen oder `dynamic()`-Ausgaben werden ausgelassen und als Build-Warnung gemeldet; andere Sass-Syntaxfehler brechen den Build ab. CSS-Inlining, Cache-Busting und AppCache-Manifeste sind ebenfalls noch nicht implementiert. Für die Preview werden vorhandene Sencha-App-CSS-Artefakte als Theme-Basis bevorzugt.
