# Schritt 11: Assets und Build-Ausgabe

## Verhalten

Der Compiler verarbeitet lokale Einträge aus `app.json.resources` als Pfadstrings oder Objekte mit `path`, `output` und optional `remote`. Verzeichnisse werden rekursiv kopiert nach `<out>/resources/<output>`; ohne `output` wird ein stabiler relativer Zielpfad aus dem Quellpfad gebildet. Remote-Ressourcen werden nicht kopiert.

Lokale statische Stylesheets aus `app.json.css` werden an ihrem relativen Pfad ins Build-Verzeichnis kopiert und in der Preview eingebunden. Remote-CSS wird direkt verlinkt. Dynamische Sencha-Platzhalter wie `${build.out.css.path}` werden übersprungen, da sie auf Sass-/Theme-Build-Ausgaben verweisen.

Zusätzliche `app.json.js`-Einträge mit `bundle:false` werden ebenfalls in ihrer Manifest-Reihenfolge kopiert und geladen. `remote:true`-Einträge werden nicht kopiert und als externe Script-URL in der Preview beibehalten.

Build-Pfade bleiben auf das App-Verzeichnis beschränkt, Output-Pfade auf den Ressourcen-Zielordner. Symlinks werden übersprungen und doppelte Zielpfade mit unterschiedlichem Inhalt als Konflikt gemeldet. Resource-Quellpfade werden vom Watcher beobachtet. Die Preview liefert generierte Ressourcen unter `/resources/` aus.

Der Sencha-Platzhalter `${build.id}` wird in Ressourcenpfaden anhand vorhandener `build/<mode>/<app-name>`-Verzeichnisse aufgelöst. Für Development-Builds wird zunächst das Development-Verzeichnis, danach ein vorhandener Production-Build verwendet; fehlt das referenzierte Verzeichnis, wird dieser Resource-Eintrag übersprungen.

Transitive SDK-Packages können zusätzlich `sencha.resources` deklarieren. Diese Quellen werden nach ihrer Package-Auflösung in dasselbe `resources`-Ausgabeziel kopiert; ein fehlendes toolkit-spezifisches Resource-Verzeichnis wird übersprungen.

## Prüfungen

- Integrationstest für Verzeichnis- und Einzeldatei-Ressourcen sowie `output`-Ziele.
- Integrationstest für `${build.id}/resources` aus einem vorherigen Sencha-Build.
- Preview-HTTP-Test für den Abruf einer kopierten Resource.
- Build- und Preview-Test für ein lokales `app.json.css`-Stylesheet.
- Fake-SDK-Integrationstest für ein transitive Package-Resource-Verzeichnis.
- `testdata/sample` wird aus seinem Sencha-`resources`-Ordner nach `build-go/resources/shared` kopiert.

## Grenzen

Sass-Transformationen aus Source-SCSS, CSS-Inlining, Cache-Busting und AppCache-Manifeste sind noch nicht implementiert. Für die Preview werden vorhandene Sencha-App-CSS-Artefakte bevorzugt.
