# Schritt 7: Watch- und Preview-Server

## Start

```sh
go run ./cmd/tualojs watch \
  -app testdata/fixtures/minimal-modern \
  -toolkit modern \
  -mode development \
  -normalize-let-const \
  -addr 127.0.0.1:1841
```

Danach die im Terminal angezeigte lokale URL öffnen. Mit `Ctrl+C` wird der Server beendet.

## Verhalten

- Führt beim Start einen Go-Build aus.
- Verwendet standardmäßig `development`; `-mode production` aktiviert die Production-Präprozessorregeln.
- `-platform desktop|phone|tablet` wählt optionale Sencha-Plattformbedingungen aus; ohne die Option führen Plattformmarker zu einer Diagnose statt zu einer impliziten Auswahl.
- Liefert eine Preview-Seite, das zum Toolkit passende Ext-JS-Debug-Runtime-Script aus dem konfigurierten SDK, dessen Theme-CSS, das Go-Bundle und statische App-Dateien aus.
- Liefert kopierte `app.json.resources` unter `/resources/` aus und beobachtet Änderungen in diesen Quellpfaden.
- Kopiert und verlinkt lokale statische CSS-Einträge aus `app.json.css`.
- Kompiliert Standard-SCSS-Dateien neben gleichnamigen JavaScript-Klassen und lädt `app-sass.css` nach dem Theme-CSS.
- Schreibt zusätzlich das kombinierte `<app-name>-all.css` in den Watch-Ausgabeordner; es enthält das vorgebaute SDK-Theme-CSS, lokale CSS-Einträge und unterstützte Sass-Ausgaben.
- Liefert optionale `app.js.map`-Dateien sowie ihre `sources` über `/sources/` aus.
- Build-Optionen `-minify` und `-source-map` werden auch im Watch-Modus unterstützt.
- Bevorzugt vorhandenes kombiniertes Sencha-App-CSS unter `build/<mode>/<app>/<toolkit>/resources` oder `build/<mode>/<app>/resources`, damit App- und Package-Styles mit angezeigt werden. Andernfalls verwendet sie das SDK-Theme-CSS; fehlt auch dieses, startet sie ohne Theme-CSS.
- Aktiviert `Ext.Loader`, setzt den SDK-Classpath und lädt die ermittelten Framework-Klassen vor dem App-Bundle; Framework-Quelldateien werden bei Bedarf über `/sdk/classes/...` ausgeliefert.
- Prüft Projekt- und Quellpfade standardmäßig alle 500 ms auf Änderungen (`-interval` ändert das Intervall).
- Baut bei Änderungen neu. Die Browserseite fragt den Build-Stand ab und lädt nach erfolgreichem Build neu.
- Erzeugt das Bundle atomar. Bei Build-Fehlern bleibt der letzte erfolgreiche Build verfügbar; der Fehler erscheint im Terminal.

## Grenzen

Die Preview verwendet das vorgebaute Debug-Runtime-Script des Ext-SDK und das ausgewählte Theme. Sie erzeugt noch nicht den Sencha-Microloader, Produktions-Bundles oder eine vollständige Fashion-kompatible Theme-Kompilierung. App- und aufgelöste Package-Ressourcen werden vom Build kopiert. Der Server bindet standardmäßig nur an `127.0.0.1`.

## Prüfungen

- HTTP-Integrationstest für Preview-Seite, Framework-Runtime, Theme-CSS und Go-Bundle.
- Watch-Test, der eine Quelländerung erkennt und das Bundle neu erzeugt.
