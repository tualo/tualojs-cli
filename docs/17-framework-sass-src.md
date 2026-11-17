# Schritt 17: Framework-Class-SCSS

## Beobachtete Lücke

Das SDK ist im Projekt vorhanden und seine Toolkit-/Core-Packages werden für Runtime und Sass-`etc`-/Variablen-Kontext aufgelöst. Die `sass.src`-Wurzeln dieser `frameworkPackages` wurden bisher jedoch nicht an die Class-SCSS-Auflösung übergeben. Framework-Styles fehlten daher auch dann, wenn JavaScript- und SCSS-Datei gemeinsam im SDK liegen.

## Verhalten

- `sass.src`-Wurzeln der ausgewählten Framework- und Core-Packages werden zusammen mit den Quellen der App-Packages durchsucht.
- Eine Framework-SCSS-Datei wird aufgenommen, wenn eine gleichnamige JavaScript-Klasse im zugehörigen Package-Classpath vorhanden ist.
- Theme-/Package-Variablen und Watch-Pfade werden mit den vorhandenen Sass-Pipelines geteilt; doppelte Wurzeln werden dedupliziert.
- Weiterhin nicht unterstützte Fashion-Mixins, Variablen oder Imports werden wie bisher gewarnt und ausgelassen.

## Prüfungen

- Fake-SDK-Integrationstest: Framework-Package mit `sass.src` wird ohne explizite App-Package-Anforderung eingebunden und kompiliert.
- Sass-Datei und Framework-Package-Pfade erscheinen in der Watch-Konfiguration.
- Modern-/Classic-Builds des `default`-Realprojekts sowie `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Das deckt Framework-Styles ab, deren Quellen über SDK-Package-Manifeste beschrieben sind. Vollständige Sencha-Fashion-Generierung, Theme-`@include`-Aufrufe und Fashion-abhängige Framework-Styles bleiben weitere Schritte.
