# Schritt 12: Source Maps und Minifizierung

## Source Maps

`build` und `watch` unterstützen `-source-map`. Der Compiler erzeugt `app.js.map` im Source Map v3-Format und ergänzt `//# sourceMappingURL=app.js.map` im Bundle. Die Maps enthalten Zeilen-Zuordnungen, die Quelldateien relativ zum Build-Ordner und `sourcesContent`.

Der Preview-Server liefert die Map über `/app.js.map` und Quelldateien über `/sources/` aus. Die Zuordnungen sind absichtlich zeilenbasiert; die aktuelle Transformation kann Token innerhalb einer Zeile ändern.

## Minifizierung

Minifizierung ist noch nicht implementiert. Vor der Umsetzung muss das Verhalten der festgelegten Sencha-Cmd-8.0.0-Referenzbuilds bestimmt werden.

## Prüfungen

- Integrationstest prüft die Source Map v3-Struktur, Quellinhalte und `sourceMappingURL`.
- HTTP-Test prüft den Abruf von `app.js.map` und einer Quelldatei.
- `node --check` prüft das erzeugte Bundle mit Source-Map-Kommentar.
