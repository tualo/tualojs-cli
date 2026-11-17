# Schritt 11: Assets und Build-Ausgabe

## Verhalten

Der Compiler verarbeitet lokale Einträge aus `app.json.resources` als Pfadstrings oder Objekte mit `path`, `output` und optional `remote`. Verzeichnisse werden rekursiv kopiert nach `<out>/resources/<output>`; ohne `output` wird ein stabiler relativer Zielpfad aus dem Quellpfad gebildet. Remote-Ressourcen werden nicht kopiert.

Lokale statische Stylesheets aus `app.json.css` werden an ihrem relativen Pfad ins Build-Verzeichnis kopiert und in der Preview eingebunden. Remote-CSS wird direkt verlinkt. Dynamische Sencha-Platzhalter wie `${build.out.css.path}` werden übersprungen, da sie auf Sass-/Theme-Build-Ausgaben verweisen.

Zusätzliche `app.json.js`-Einträge mit `bundle:false` werden ebenfalls in ihrer Manifest-Reihenfolge kopiert und geladen. `remote:true`-Einträge werden nicht kopiert und als externe Script-URL in der Preview beibehalten.

Ein lokales Script mit `includeInBundle:true` wird statt als separate Datei in das Go-Bundle aufgenommen. Seine statischen Ext-Klassendeklarationen nehmen an der normalen Dependency-Auflösung teil.

Build-Pfade bleiben auf das App-Verzeichnis beschränkt, Output-Pfade auf den Ressourcen-Zielordner. Symlinks werden übersprungen und doppelte Zielpfade mit unterschiedlichem Inhalt als Konflikt gemeldet. Resource-Quellpfade werden vom Watcher beobachtet. Die Preview liefert generierte Ressourcen unter `/resources/` aus.

Der Sencha-Platzhalter `${build.id}` wird in Ressourcenpfaden anhand vorhandener `build/<mode>/<app-name>`-Verzeichnisse aufgelöst. Für Development-Builds wird zunächst das Development-Verzeichnis, danach ein vorhandener Production-Build verwendet; fehlt das referenzierte Verzeichnis, wird dieser Resource-Eintrag übersprungen.

Transitive SDK-Packages können Ressourcen als `sencha.resources`-Einträge oder als `sencha.resource.paths`-Pfade deklarieren. Beide Formen werden nach der Package-Auflösung in dasselbe `resources`-Ausgabeziel unter `shared/` kopiert; toolkit-spezifische Pfade werden für das ausgewählte Toolkit expandiert, fehlende Verzeichnisse übersprungen.

Die Sass-Phasen beginnen mit `app.json.sass.etc`, danach folgen Fashion-freie Package-Variablen aus `sencha.sass.var` und App-`sass.var`; anschließend kommen Package-`sass.src`, App-`sass.src` und zuletzt SCSS-Dateien neben gleichnamigen JavaScript-Klassen aus App-Classpaths und Override-Pfaden. Package-SCSS wird auf `sass/src` und `classic|modern/sass/src` abgebildet; aufgenommen werden nur Dateien mit passender JS-Klasse. Dart Sass kompiliert die Quellen über einen gemeinsamen deterministischen Einstieg, sodass normale Sass-Variablen geteilt werden. Das Ergebnis wird neben die kopierten Toolkit-Ressourcen nach `<out>/resources/<toolkit>/resources/app-sass.css` geschrieben, damit relative CSS-`url(...)`-Pfade dieselbe Basis wie die Sencha-Ressourcen verwenden. Toolkit-Ressourcen ohne explizites `output` behalten unter `resources/` ihren Toolkit-Pfad. Die Preview stellt die Datei unter `/css/app-sass.css` bereit und lädt sie nach dem vorhandenen Theme-CSS. `SASS` kann den Pfad zum ausführbaren `sass`-Programm überschreiben; standardmäßig wird `sass` über `PATH` gesucht. `-minify` verwendet Dart Sass' komprimierte CSS-Ausgabe. Sass-Einstiegsdateien und lokale `@import`-/`@use`-/`@forward`-Partials werden vom Watcher beobachtet.

## Prüfungen

- Integrationstest für Verzeichnis- und Einzeldatei-Ressourcen sowie `output`-Ziele.
- Integrationstest für `${build.id}/resources` aus einem vorherigen Sencha-Build.
- Preview-HTTP-Test für den Abruf einer kopierten Resource.
- Build- und Preview-Test für ein lokales `app.json.css`-Stylesheet.
- Build-Test für ein per `includeInBundle` eingebettetes JavaScript-Hilfsscript.
- Fake-SDK-Integrationstest für `sencha.resources` und String-/Array-Formen von `sencha.resource.paths`, einschließlich toolkit-spezifischer Pfade.
- Fake-SDK-Test für Class-SCSS aus Package-Manifesten und toolkit-spezifische `sass/src`-Pfade.
- Build-Test für SCSS neben gleichnamigen Klassen, CSS-Konkatenation, Watch-Pfaden und Einbindung in die Stylesheet-Ausgabe.
- `testdata/sample` wird aus seinem Sencha-`resources`-Ordner nach `build-go/resources/shared` kopiert.

## Grenzen

Sencha-Fashion-Themes sowie Package-`etc`- und Theme-`var`-Kontexte werden noch nicht nachgebildet. Einfache `dynamic(<literal>)`-Zuweisungen in Sass-Variablen werden auf ihren Literalwert normalisiert. Dynamische Variablen mit Abhängigkeiten, Funktionsausdrücken und Fashion-Mixins bleiben unsupported; betroffene Stylesheets werden ausgelassen und gewarnt. Package-Class-SCSS mit nicht geladenen Theme-Variablen, Imports oder `url(...)` wird ebenfalls ausgelassen. Nicht definierte Variablen oder Mixins aus importierten Dateien führen weiterhin zu einem Sass-Fehler. CSS-Inlining, Cache-Busting und AppCache-Manifeste sind ebenfalls noch nicht implementiert. Für die Preview werden vorhandene Sencha-App-CSS-Artefakte als Theme-Basis bevorzugt.
