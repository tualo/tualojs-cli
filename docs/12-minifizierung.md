# Schritt 12: Source Maps und Minifizierung

## Source Maps

`build` und `watch` unterstützen `-source-map`. Der Compiler erzeugt `app.js.map` im Source Map v3-Format und ergänzt `//# sourceMappingURL=app.js.map` im Bundle. Die Maps enthalten Token-Zuordnungen mit UTF-16-Spalten, Quelldateien und `sourcesContent`.

Der Preview-Server liefert die Map über `/app.js.map` und Quelldateien über `/sources/` aus. Source maps bleiben mit der token-aware Whitespace-Minifizierung aktiv. Die aktuelle `let`/`const`-Normalisierung und der Minifier verändern keine Token-Reihenfolge; allgemeine Transformationen könnten präzisere Mapping-Phasen erfordern.

## Minifizierung

`-minify` entfernt Kommentare und unnötigen Whitespace anhand des Go-Tokenizers. Strings, Regex-Literale und die Anzahl der Zeilenumbrüche bleiben erhalten; `/*! ... */`- und `//!`-Lizenzkommentare werden bewahrt. Die Option ist für `build` und `watch` verfügbar.

Dies ist noch keine Sencha-Cmd-äquivalente Minifizierung: Identifier werden nicht umbenannt, toter Code nicht entfernt und Closure-Compiler-Optimierungen nicht nachgebildet.

## Prüfungen

- Integrationstest prüft die Source Map v3-Struktur, mehrere Token-Spalten, Quellinhalte und `sourceMappingURL`, auch in Kombination mit `-minify`.
- HTTP-Test prüft den Abruf von `app.js.map` und einer Quelldatei.
- `node --check` prüft das erzeugte Bundle mit Source-Map-Kommentar.
