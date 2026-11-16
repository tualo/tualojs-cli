# Schritt 7: Watch- und Preview-Server

## Start

```sh
go run ./cmd/extjsc watch \
  -app testdata/fixtures/minimal-modern \
  -toolkit modern \
  -addr 127.0.0.1:1841
```

Danach die im Terminal angezeigte lokale URL öffnen. Mit `Ctrl+C` wird der Server beendet.

## Verhalten

- Führt beim Start einen Go-Build aus.
- Liefert eine Preview-Seite, das zum Toolkit passende Ext-JS-Debug-Runtime-Script aus dem konfigurierten SDK, dessen Theme-CSS, das Go-Bundle und statische App-Dateien aus.
- Wenn das SDK kein vorgebautes Theme-CSS enthält, verwendet die Preview vorhandenes Development-CSS von Sencha Cmd unter `build/development/<app>/resources`; fehlt auch diese Datei, startet sie ohne Theme-CSS.
- Aktiviert `Ext.Loader`, setzt den SDK-Classpath und lädt die ermittelten Framework-Klassen vor dem App-Bundle; Framework-Quelldateien werden bei Bedarf über `/sdk/classes/...` ausgeliefert.
- Prüft Projekt- und Quellpfade standardmäßig alle 500 ms auf Änderungen (`-interval` ändert das Intervall).
- Baut bei Änderungen neu. Die Browserseite fragt den Build-Stand ab und lädt nach erfolgreichem Build neu.
- Erzeugt das Bundle atomar. Bei Build-Fehlern bleibt der letzte erfolgreiche Build verfügbar; der Fehler erscheint im Terminal.

## Grenzen

Die Preview verwendet das vorgebaute Debug-Runtime-Script des Ext-SDK und das ausgewählte Theme. Sie erzeugt noch nicht den Sencha-Microloader, Produktions-Bundles, Package-Ressourcen oder Sass-Ausgaben. Der Server bindet standardmäßig nur an `127.0.0.1`.

## Prüfungen

- HTTP-Integrationstest für Preview-Seite, Framework-Runtime, Theme-CSS und Go-Bundle.
- Watch-Test, der eine Quelländerung erkennt und das Bundle neu erzeugt.
