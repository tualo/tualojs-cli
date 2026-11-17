# Schritt 10: Sencha-Präprozessoren

## Build-Modus

`extjsc build` und `extjsc watch` akzeptieren `-mode development|production`. Der Standard ist `development`.

## `//<debug>`-Blöcke

- Development erhält Debug-Marker und ihren Inhalt.
- Production entfernt den Inhalt verschachtelter `//<debug>`/`//</debug>`-Blöcke.
- Die Präprozessierung läuft vor dem Metadatenscan, damit entfernter Code keine Klassenabhängigkeiten in Production-Builds erzeugt.
- Nicht geschlossene `<debug>`-Blöcke sind Fehler. Vereinzelte `//</debug>`-Marker aus vorhandenen SDK-/Package-Quellen werden als Kommentare behandelt.

## Toolkit-Bedingungen

- `// <if classic>` und `// <if modern>` schließen Inhalt für das jeweils ausgewählte Toolkit ein.
- `//<if nonBrowser>` und `//<if legacyBrowser>` werden in Browser-Builds ausgeschlossen.
- Die Blöcke werden in Development und Production vor dem Metadatenscan verarbeitet. Dadurch erzeugen inaktive Blöcke keine Klassenabhängigkeiten.
- Die Markerzeilen selbst werden entfernt; Zeilenumbrüche bleiben zur Stabilisierung der Quellpositionen erhalten.
- `classic` und `modern` brauchen ein explizit ausgewähltes Toolkit. Unbekannte Bedingungen, nicht passende `</if condition>`-Marker und unausgeglichene Blöcke sind Fehler.
- Toolkit-Bedingungen können mit `<debug>`-Blöcken verschachtelt werden.

## Prüfungen

- Unit- und Integrationstests prüfen verschachtelte Blöcke, debug-only Klassenabhängigkeiten und fehlerhafte Marker.
- Classic-/Modern-Integrationstest prüft toolkit-spezifische Abhängigkeiten aus `<if>`-Blöcken.

## Grenzen

Plattformbedingungen wie `desktop`/`phone`, weitere Sencha-Bedingungen wie `<feature>`, negierte und zusammengesetzte Ausdrücke, weitere Präprozessoren und Compiler-Makros sind noch nicht umgesetzt.
