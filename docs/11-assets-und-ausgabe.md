# Schritt 11: Assets und Build-Ausgabe

## Verhalten

Der Compiler verarbeitet lokale Einträge aus `app.json.resources` als Pfadstrings oder Objekte mit `path`, `output` und optional `remote`. Verzeichnisse werden rekursiv kopiert nach `<out>/resources/<output>`; ohne `output` wird ein stabiler relativer Zielpfad aus dem Quellpfad gebildet. Remote-Ressourcen werden nicht kopiert.

Lokale statische Stylesheets aus `app.json.css` werden an ihrem relativen Pfad ins Build-Verzeichnis kopiert und in der Preview eingebunden. Remote-CSS wird direkt verlinkt. Dynamische Sencha-Platzhalter wie `${build.out.css.path}` werden übersprungen, da sie auf Sass-/Theme-Build-Ausgaben verweisen.

Wenn lokale CSS-Quellen vorhanden sind, erzeugt `build` zusätzlich `<out>/resources/<toolkit>/resources/<app-name>-all.css`. Dafür werden das vorgebaute Theme-CSS aus `SDK/build/<toolkit>/<theme>/resources/<theme>-all.css`, lokale Stylesheets aus `app.json.css` und – sofern erzeugt – `app-sass.css` in dieser Reihenfolge zusammengeführt. Die Ressourcen des SDK-Themes werden mitkopiert, damit relative Theme-URLs erhalten bleiben; relative `url(...)`-Pfade aus lokalen Stylesheets werden auf die Position der kombinierten Datei umgeschrieben. Ohne Toolkit liegt die Datei unter `<out>/resources/<app-name>-all.css`. Remote-CSS bleibt ein separates Asset. Der JSON-Modus weist die Datei unter `combinedCSS` aus.

Zusätzliche `app.json.js`-Einträge mit `bundle:false` werden ebenfalls in ihrer Manifest-Reihenfolge kopiert und geladen. `remote:true`-Einträge werden nicht kopiert und als externe Script-URL in der Preview beibehalten.

Ein lokales Script mit `includeInBundle:true` wird statt als separate Datei in das Go-Bundle aufgenommen. Seine statischen Ext-Klassendeklarationen nehmen an der normalen Dependency-Auflösung teil.

Build-Pfade bleiben auf das App-Verzeichnis beschränkt, Output-Pfade auf den Ressourcen-Zielordner. Symlinks werden übersprungen und doppelte Zielpfade mit unterschiedlichem Inhalt als Konflikt gemeldet. Resource-Quellpfade werden vom Watcher beobachtet. Die Preview liefert generierte Ressourcen unter `/resources/` aus.

Der Sencha-Platzhalter `${build.id}` wird in Ressourcenpfaden anhand vorhandener `build/<mode>/<app-name>`-Verzeichnisse aufgelöst. Für Development-Builds wird zunächst das Development-Verzeichnis, danach ein vorhandener Production-Build verwendet; fehlt das referenzierte Verzeichnis, wird dieser Resource-Eintrag übersprungen.

Transitive SDK-Packages können Ressourcen als `sencha.resources`-Einträge oder als `sencha.resource.paths`-Pfade deklarieren. Beide Formen werden nach der Package-Auflösung in dasselbe `resources`-Ausgabeziel unter `shared/` kopiert; toolkit-spezifische Pfade werden für das ausgewählte Toolkit expandiert, fehlende Verzeichnisse übersprungen.

Die Sass-Phasen beginnen mit `app.json.sass.etc`, danach folgen SDK-Root- und Package-`sencha.sass.etc`-Einstiege außerhalb von Theme-Packages, Theme-Helper aus lokalen Imports, Package-Variablen aus `sencha.sass.var`, Theme-Package-`etc`-Einstiege samt unterstützten Imports und App-`sass.var`; anschließend kommen Class-SCSS aus App- und Framework-Packages, App-`sass.src` und zuletzt SCSS-Dateien neben gleichnamigen JavaScript-Klassen aus App-Classpaths und Override-Pfaden. Package-SCSS wird auf `sass/src` und `classic|modern/sass/src` abgebildet; aufgenommen werden nur Dateien mit passender JS-Klasse. Dart Sass kompiliert die Quellen über einen gemeinsamen deterministischen Einstieg, sodass normale Sass-Variablen geteilt werden. Das Ergebnis wird neben die kopierten Toolkit-Ressourcen nach `<out>/resources/<toolkit>/resources/app-sass.css` geschrieben, damit relative CSS-`url(...)`-Pfade dieselbe Basis wie die Sencha-Ressourcen verwenden. Toolkit-Ressourcen ohne explizites `output` behalten unter `resources/` ihren Toolkit-Pfad. Die Preview stellt die Datei unter `/css/app-sass.css` bereit und lädt sie nach dem vorhandenen Theme-CSS. `SASS` kann den Pfad zum ausführbaren `sass`-Programm überschreiben; standardmäßig wird `sass` über `PATH` gesucht. `-minify` verwendet Dart Sass' komprimierte CSS-Ausgabe. Sass-Einstiegsdateien und lokale `@import`-/`@use`-/`@forward`-Partials werden vom Watcher beobachtet.

## Prüfungen

- Integrationstest für Verzeichnis- und Einzeldatei-Ressourcen sowie `output`-Ziele.
- Integrationstest für `${build.id}/resources` aus einem vorherigen Sencha-Build.
- Preview-HTTP-Test für den Abruf einer kopierten Resource.
- Build- und Preview-Test für ein lokales `app.json.css`-Stylesheet.
- Build-Test für ein per `includeInBundle` eingebettetes JavaScript-Hilfsscript.
- Fake-SDK-Integrationstest für `sencha.resources` und String-/Array-Formen von `sencha.resource.paths`, einschließlich toolkit-spezifischer Pfade.
- Fake-SDK-Test für Class-SCSS aus Package-Manifesten und toolkit-spezifische `sass/src`-Pfade.
- Build-Test für SCSS neben gleichnamigen Klassen, CSS-Konkatenation, Watch-Pfaden und Einbindung in die Stylesheet-Ausgabe.
- Integrationstest für Theme- plus App-CSS in `<app-name>-all.css`, URL-Rebasing, kopierte Theme-Ressourcen und den Manifest-Eintrag.
- `testdata/sample` wird aus seinem Sencha-`resources`-Ordner nach `build-go/resources/shared` kopiert.

## Grenzen

Sencha-Fashion-Themes und Fashion-Mixins werden noch nicht vollständig nachgebildet. Das kombinierte Stylesheet führt das vorgebaute SDK-Theme-CSS mit den vom Compiler unterstützten lokalen CSS-/Sass-Ausgaben zusammen; es ist kein Ersatz für eine vollständige Sencha-Cmd-Fashion-Kompilierung. `dynamic()`-Wrapper um Dart-Sass-kompatible Variablen-Ausdrücke werden normalisiert; Fashion-spezifische Funktionen und dynamische Theme-Auswertung bleiben unsupported. Package-Class-SCSS mit nicht geladenen Theme-Variablen, nicht unterstützten Imports oder `url(...)` wird ebenfalls ausgelassen. Unbekannte Variablen oder Mixins aus importierten Dateien können weiterhin zu Sass-Warnungen oder -Fehlern führen. CSS-`@import`-Abhängigkeiten, Cache-Busting und AppCache-Manifeste werden nicht eingebettet beziehungsweise erzeugt. Für die Preview werden vorhandene Sencha-App-CSS-Artefakte als Theme-Basis bevorzugt.
