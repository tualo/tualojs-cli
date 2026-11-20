# tualojs

[English](README.md) · [Deutsch](README.de.md)

**tualojs ist ein eigenständiger Compiler für Ext-JS-Anwendungen in Go.** Er erstellt JavaScript-Bundles und zugehörige Anwendungsdateien aus einem Ext-JS-Projekt. Die Kompatibilität mit dokumentiertem Sencha-Cmd-Verhalten wird schrittweise umgesetzt.

> **Kompatibilitätsumfang:** Aktuelles Ziel ist Ext JS 8.0.0 (SDK-Kompatibilitätsversion `8.0.0`). tualojs ist kein vollständiger Ersatz für Sencha Cmd. Unterstützte Funktionen und bekannte Einschränkungen sind in [`docs/`](docs/) dokumentiert.

## Funktionen

- Erstellt Builds für Ext-JS-Anwendungen mit **Classic** und **Modern** Toolkit.
- Löst Anwendungsklassen und Abhängigkeiten auf, darunter `requires`, `uses`, Overrides, Aliase und Xtype-Referenzen.
- Verarbeitet `app.json`, `workspace.json`, Ext-JS-SDK-Manifeste und unterstützte SDK-Packages.
- Verarbeitet JavaScript, ausgewählte Sencha-Präprozessor-Konstrukte, Stylesheets, Sass und statische Ressourcen.
- Unterstützt Development- und Production-Modus, Source Maps, optionale JavaScript-Minifizierung und ein JSON-Asset-Manifest.
- Bietet einen lokalen Watch- und Preview-Server.
- Kann optional die Ext-JS-Runtime aus dem SDK an den Anfang des erzeugten `app.js`-Bundles stellen.
- Benötigt Sencha Cmd nicht zur Laufzeit und hat keine externen Go-Modulabhängigkeiten.

## Voraussetzungen

- **Go 1.23 oder neuer** zum Erstellen aus dem Quellcode.
- Ein lokal verfügbares Ext-JS-8-SDK, das im Workspace konfiguriert ist, wenn SDK-/Package-Auflösung oder die Ext-JS-Runtime benötigt wird. Das SDK muss die Kompatibilitätsversion `8.0.0` ausweisen.
- **Dart Sass** als `sass` im `PATH`, wenn der Build Sass-Quellen enthält. Mit der Umgebungsvariable `SASS` kann ein anderer Pfad zum ausführbaren Programm angegeben werden.

Der Compiler enthält keine Lizenz für Ext JS, Sencha Cmd, SDK-Packages oder andere Sencha-Ressourcen und erteilt auch keine solchen Rechte. Siehe [Lizenz und Drittsoftware](#lizenz-und-drittsoftware).

## tualojs erstellen

Binärdatei für die aktuelle Plattform erstellen:

```sh
go build -o tualojs ./cmd/tualojs
```

Binärdateien für Linux, Windows und macOS auf amd64 und arm64 erstellen:

```sh
./build.sh
```

Die Binärdateien werden in `dist/` abgelegt. Tests und statische Prüfungen ausführen:

```sh
go test ./...
go vet ./...
```

## Ext-JS-SDK konfigurieren

Für die SDK- und Package-Auflösung muss die `workspace.json` der Anwendung auf ein lokal verfügbares Ext-JS-SDK zeigen. Liegt das SDK beispielsweise neben dem Anwendungsverzeichnis, kann die Datei so aussehen:

```json
{
  "frameworks": {
    "ext": {
      "path": "../ext-8.0.0",
      "version": "8.0.0.43"
    }
  }
}
```

Die `package.json` des SDKs muss das Framework als `ext` kennzeichnen und `compatVersion` `8.0.0` angeben. Das SDK wird separat bereitgestellt und lizenziert; es ist nicht Teil von tualojs.

## Verwendung

Eine Anwendung bauen. Ihr Verzeichnis kann mit `-app` angegeben werden:

```sh
./tualojs build \
  -app ./meine-ext-js-app \
  -out ./meine-ext-js-app/build-tualojs \
  -toolkit modern \
  -mode production \
  -source-map \
  -minify
```

Ist das Toolkit bereits in `app.json` festgelegt, kann `-toolkit` entfallen. Für Universal Apps `classic` oder `modern` auswählen. `app.js` und alle erzeugten oder kopierten Dateien werden im Ausgabeverzeichnis abgelegt.

Während der Entwicklung den lokalen Watch- und Preview-Server starten:

```sh
./tualojs watch \
  -app ./meine-ext-js-app \
  -toolkit modern \
  -addr 127.0.0.1:1841
```

Der Server gibt die lokale Preview-URL im Terminal aus. Er überwacht Projektdateien, baut bei Änderungen neu und aktualisiert die Preview nach einem erfolgreichen Build.

### Nützliche Optionen

| Option | Befehl | Beschreibung |
| --- | --- | --- |
| `-app <directory>` | `build`, `watch` | Anwendungsverzeichnis mit `app.json` (Standard: `.`). |
| `-out <directory>` | `build`, `watch` | Ausgabeverzeichnis (Standard: `build-go`). |
| `-toolkit <toolkit>` | `build`, `watch` | `classic` oder `modern` auswählen, auch für Universal Apps. |
| `-platform <platform>` | `build`, `watch` | Sencha-Plattformbedingungen für `desktop`, `phone` oder `tablet` auswählen. |
| `-mode <mode>` | `build`, `watch` | `development` oder `production` auswählen (Standard: `development`). |
| `-source-map` | `build`, `watch` | Eine `app.js.map`-Source-Map schreiben. |
| `-minify` | `build`, `watch` | JavaScript-Kommentare und überflüssige Leerzeichen entfernen. |
| `-normalize-let-const` | `build`, `watch` | Unterstützte `let`-/`const`-Deklarationen im Funktions-Scope in `var` umwandeln. |
| `-upgrade-es5` | `build`, `watch` | Unterstützte lokale ES5-`var`-Deklarationen in `const`/`let` umwandeln. |
| `-json` | `build` | Ein Asset-Manifest nach stdout schreiben. |
| `-base-path <url-prefix>` | `build` | Öffentlichen URL-Präfix im JSON-Manifest setzen. |
| `-include-extjs` | `build` | Die aufgelöste Ext-JS-Runtime in `app.js` einfügen. Erfordert SDK und Toolkit. |
| `-addr <address>` | `watch` | Adresse des Preview-Servers (Standard: `127.0.0.1:1841`). |
| `-interval <duration>` | `watch` | Prüfintervall für Projektänderungen (Standard: `500ms`). |

Die vollständige CLI-Hilfe gibt es mit `./tualojs --help`, `./tualojs build --help` oder `./tualojs watch --help`.

### JSON-Asset-Manifest

Das Manifest kann einem Deployment-Werkzeug mitteilen, wo erzeugte CSS-, JavaScript- und Ressourcen-Dateien liegen:

```sh
./tualojs build \
  -app ./meine-ext-js-app \
  -out ./meine-ext-js-app/build-tualojs \
  -toolkit modern \
  -json \
  -base-path /assets/meine-ext-js-app/
```

Das JSON wird nach stdout geschrieben; Build-Warnungen erscheinen auf stderr. Die Felder des Manifests sind in [`docs/43-json-asset-manifest.md`](docs/43-json-asset-manifest.md) beschrieben.

## Kompatibilität und Einschränkungen

tualojs implementiert einen dokumentierten Teil des Ext-JS- und Sencha-Cmd-Verhaltens. Insbesondere gelten folgende Grenzen:

- Dynamische JavaScript-Klassennamen und andere erst zur Laufzeit berechnete Referenzen lassen sich im Allgemeinen nicht zur Build-Zeit auflösen.
- Sencha Fashion wird nur teilweise unterstützt. Die Sass-Kompatibilität deckt dokumentierte Einzelfälle ab, nicht die vollständige Fashion-Laufzeit oder das gesamte Theme-System.
- Die JavaScript-Minifizierung entfernt Kommentare und überflüssige Leerzeichen; sie benennt keine Identifier um und führt nicht alle Optimierungen der Sencha-Cmd-Produktionspipeline aus.
- Weitere nicht unterstützte Fälle und bekannte Abweichungen stehen in den jeweiligen Dateien unter [`docs/`](docs/).

Ob tualojs für eine konkrete Anwendung geeignet ist, hängt davon ab, ob die benötigten Build-Funktionen vom dokumentierten Umfang abgedeckt sind.

## Dokumentation

- [`goals.md`](goals.md) — Projektziele, umgesetzte Funktionen und aktueller Umfang.
- [`docs/00-roadmap.md`](docs/00-roadmap.md) — Roadmap und Architekturleitlinien.
- [`docs/`](docs/) — Verhalten, Beispiele, Tests und Grenzen der einzelnen Funktionen.

## Lizenz und Drittsoftware

Der Quellcode von tualojs steht unter der [Apache License 2.0](LICENSE). Die Lizenz gilt nur für tualojs und **erteilt keine Rechte an Sencha Ext JS, Sencha Cmd oder einem Sencha-SDK, -Package oder -Asset**. Für diese gelten die jeweiligen Lizenzbedingungen. Ein Build kann Inhalte aus dem SDK einer Anwendung übernehmen oder kopieren; Nutzer sind dafür verantwortlich, die jeweils geltenden Bedingungen einzuhalten. Siehe [`NOTICE`](NOTICE).
