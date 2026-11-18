# Schritt 32: Package-Sass-Variablenimports

## Beobachtete Lücke

Einige `sass.var`-Dateien importieren lokale Variablendateien ausdrücklich, damit deren Werte vor Ort verfügbar sind. Beispielsweise importiert `toolbar/Breadcrumb.scss` zuerst `Toolbar.scss`. Solche Dateien wurden bislang wegen jedes `@import` vollständig übersprungen.

## Verhalten

- Lokale Sass-`@import`-Pfade, die relativ zur importierenden Datei oder zum App-/SDK-Root auflösbar sind, sind in Package-Variablendateien erlaubt.
- Die Variablendatei wird wie bisher erst aufgenommen, wenn ihre externen Variablenreferenzen verfügbar sind; dadurch werden Import- und Variablenabhängigkeiten gemeinsam aufgelöst.
- Sass-Funktionsargumentnamen wie `$hue:` werden nicht als Variablenreferenzen fehlinterpretiert; vorwärts referenzierte Variablen werden im Variablenblock dependency-geordnet, auch wenn danach noch Theme-Mixins folgen.
- `@use`, `@forward`, CSS-/URL-Imports und nicht auflösbare Imports bleiben außerhalb dieses Schritts.
- Importierte Dateien bleiben über die bestehende Sass-Pipeline beobachtbar und werden beim Kompilieren inline verarbeitet.

## Prüfungen

- Fixtures prüfen lokale Import-Abhängigkeiten, Sass-Keyword-Argumente, Vorwärtsreferenzen vor nachfolgenden Mixins und erzeugtes CSS.
- Classic-Build von `testdata/sample` sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Dies bildet keine allgemeine Sass-Modulauflösung ab. Namespaces, `@use`, `@forward`, externe Imports und Import-spezifische Fashion-Semantik bleiben offen.
