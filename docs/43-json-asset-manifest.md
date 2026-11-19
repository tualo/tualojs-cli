# Schritt 43: JSON-Asset-Manifest

## Ziel

`extjsc build -json` schreibt statt der menschenlesbaren Build-Zusammenfassung ein JSON-Manifest nach stdout. Damit kann ein aufrufendes Werkzeug die vom Build erzeugten Stylesheets und Scripts in eine Website einbinden.

## Format

- `css` und `js` enthalten die Asset-URLs in ihrer Build-Reihenfolge. Das Script-Array enthält auch das gebündelte `app.js`.
- `outputDir` nennt das absolute Build-Ausgabeverzeichnis. Lokale Einträge in `css`/`js` enthalten `path` relativ zu diesem Verzeichnis und `file` als absoluten Fundort auf der Platte.
- `basePath` ist der normalisierte öffentliche URL-Präfix des Build-Ausgabeverzeichnisses. Lokale Asset-URLs werden damit bereits vollständig aufgelöst; absolute und Remote-URLs bleiben unverändert.
- `resources` bezeichnet den öffentlichen Ressourcenpfad. `resourceFiles` listet jedes kopierte Bild, jeden Font und jede weitere Ressource mit relativem Zielpfad und absolutem Dateipfad.
- `files` fasst alle lokalen CSS-, JavaScript- und Ressourcen-Dateien zum Kopieren zusammen; eine erzeugte Source Map wird ebenfalls aufgeführt.
- `frameworkIncluded` ist `true`, wenn die Ext-JS-Runtime mit `-include-extjs` in `app.js` eingebettet wurde.
- App-Name, Toolkit und Theme stehen als Kontext im Manifest. Build-Warnungen bleiben auf stderr, damit stdout valides JSON enthält.

Beispiel: `extjsc build -app ./app -out build-go -json -base-path /static/example/`

## Prüfungen

- Manifest-Tests prüfen Asset-Reihenfolge, Remote-URLs, Root-relative URLs, Basepath-Normalisierung und physische Ausgabepfade einschließlich Ressourcen-Dateien.
- Build- und CLI-Prüfungen bestätigen, dass die JSON-Ausgabe das erwartete Manifest enthält und keine Text-Zusammenfassung auf stdout schreibt.
- `go test -count=1 ./...` und `go vet ./...`.

## Grenzen

Das Manifest listet die vom Compiler bereitgestellten CSS-/JavaScript-Einbindungen und den Ressourcenpfad. Framework-Runtime und dynamisch nachgeladene Framework-Klassen werden weiterhin über das bestehende SDK-/Loader-Setup bereitgestellt.
