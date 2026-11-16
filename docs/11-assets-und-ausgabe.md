# Schritt 11: App-Ressourcen

## Verhalten

Der Compiler verarbeitet lokale Einträge aus `app.json.resources` als Pfadstrings oder Objekte mit `path`, `output` und optional `remote`. Verzeichnisse werden rekursiv kopiert nach `<out>/resources/<output>`; ohne `output` wird ein stabiler relativer Zielpfad aus dem Quellpfad gebildet. Remote-Ressourcen werden nicht kopiert.

Build-Pfade bleiben auf das App-Verzeichnis beschränkt, Output-Pfade auf den Ressourcen-Zielordner. Symlinks werden übersprungen und doppelte Zielpfade mit unterschiedlichem Inhalt als Konflikt gemeldet. Resource-Quellpfade werden vom Watcher beobachtet. Die Preview liefert generierte Ressourcen unter `/resources/` aus.

## Prüfungen

- Integrationstest für Verzeichnis- und Einzeldatei-Ressourcen sowie `output`-Ziele.
- Preview-HTTP-Test für den Abruf einer kopierten Resource.
- `testdata/sample` wird aus seinem Sencha-`resources`-Ordner nach `build-go/resources/shared` kopiert.

## Grenzen

Package-Ressourcen, CSS/Sass-Transformationen, Inlining, Cache-Busting und AppCache-Manifeste sind noch nicht implementiert.
