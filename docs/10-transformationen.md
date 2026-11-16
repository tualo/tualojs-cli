# Schritt 10: Sencha-`debug`-Präprozessor

## Build-Modus

`extjsc build` und `extjsc watch` akzeptieren `-mode development|production`. Der Standard ist `development`.

## `//<debug>`-Blöcke

- Development erhält Debug-Marker und ihren Inhalt.
- Production entfernt den Inhalt verschachtelter `//<debug>`/`//</debug>`-Blöcke.
- Die Präprozessierung läuft vor dem Metadatenscan, damit entfernter Code keine Klassenabhängigkeiten in Production-Builds erzeugt.
- Nicht geschlossene `<debug>`-Blöcke sind Fehler. Vereinzelte `//</debug>`-Marker aus vorhandenen SDK-/Package-Quellen werden als Kommentare behandelt.

## Prüfungen

Unit- und Integrationstests prüfen verschachtelte Blöcke, debug-only Klassenabhängigkeiten und fehlerhafte Marker.

## Grenzen

Toolkit-/Plattformbedingungen, weitere Sencha-Präprozessoren und Compiler-Makros sind noch nicht umgesetzt.
