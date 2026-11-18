# Schritt 29: Package-Sass-Einstiegsdateien

## Beobachtete Lücke

Einige Package-Manifeste listen neben dem Verzeichnis `sass/src` auch eine konkrete Einstiegsdatei wie `sass/src/all.scss`. Bisher wurden solche Dateien als ungültiger Sass-Pfad gemeldet und nicht kompiliert; dadurch fehlten globale Package-Styles, etwa Font-Awesome-Regeln.

## Verhalten

- `sencha.sass.src` akzeptiert sowohl Verzeichniswurzeln für klassenbezogene Sass-Dateien als auch explizite `.scss`-Dateien innerhalb des Packages.
- Explizite Einstiegsdateien werden einmalig als Package-Stylesheets aufgenommen, ohne eine gleichnamige JavaScript-Klasse vorauszusetzen.
- Sass-Imports der Einstiegsdatei bleiben Dart Sass überlassen und werden in die Watch-Pfade aufgenommen.
- Nicht unterstützte Fashion-Konstrukte in einer Einstiegsdatei führen weiterhin zu einer aussagekräftigen Warnung und werden ausgelassen.

## Prüfungen

- Fixture prüft die Kombination aus `sass/src`-Verzeichnis und `sass/src/all.scss`, Importauflösung, Watch-Pfade und erzeugtes CSS.
- Classic-Build des `testdata/sample`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Die Aufnahme einer Einstiegsdatei implementiert keine Sencha-Fashion-JavaScript-Funktionen. Einträge, die etwa `get-resource-path()` benötigen, werden weiterhin gewarnt und ausgelassen.
