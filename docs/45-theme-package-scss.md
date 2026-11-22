# Schritt 45: Theme-Package-SCSS für geladene Klassen

## Beobachtete Lücke

Theme-Packages enthalten Komponenten-SCSS, deren JavaScript-Klassen im Framework oder in anderen Packages liegen. Die bisherige Package-SCSS-Auflösung verlangte eine passende JavaScript-Datei im Theme-Package selbst und ließ diese Styles dadurch aus. Mixins, die ausschließlich in Sass-Variablendateien definiert waren, gelangten ebenfalls nicht in den Sass-Kontext.

## Verhalten

- Sass-Quellen aus `sass/src` eines Theme-Packages werden ohne lokale JavaScript-Gegenstelle erfasst.
- In den Sass-Build gelangen Theme-Regeln nur, wenn die zugehörige Klassen-JavaScript-Datei im ausgewählten App-Build geladen ist. So werden nicht benötigte Sass-Dateien mit unaufgelösten Variablen ausgelassen.
- Nach der Abhängigkeitsauflösung werden wiederholte Theme-Variablen in der Vererbungsreihenfolge der Theme-Packages erneut angewendet. So überschreibt das ausgewählte Kind-Theme die zuvor geladenen Eltern-Theme-Werte.
- Mixin-Definitionen aus Package-`sass.var`-Dateien stehen den geladenen Theme-Regeln zur Verfügung, auch wenn eine Datei keine Top-Level-Variablendeklaration enthält.
- Variable `theme-background-image(...)`-Aufrufe werden durch die Sass-Kompatibilitätsfunktion aufgelöst; statische Aufrufe werden weiterhin direkt auf kopierte Ressourcen abgebildet.
- Theme-Regeln werden zusammen mit den übrigen unterstützten Sass-Quellen in `app-sass.css` ausgegeben und in das kombinierte App-CSS übernommen.

## Prüfungen

- Integrationstest mit Dart Sass für App-Variablen-Overrides, Theme-Mixins und Theme-SCSS ohne JavaScript-Datei im Theme-Package.
- Regressionstest für eine verzögerte Eltern-Theme-Variablenquelle, die anschließend vom Kind-Theme überschrieben wird.
- Integrationstest für `theme-background-image()` mit kopierter Package-Ressource.
- Test, dass Sass-Regeln ohne geladene Klassen nicht in den Build gelangen.
- Classic-Realprojekt-Build für `theme-triton` sowie `go test ./...` und `go vet ./...`.

## Grenzen

Das stellt keinen vollständigen Sencha-Fashion-Kontext her. Nicht unterstützte Fashion-Mixins, dynamische JavaScript-Funktionen und andere Fashion-spezifische Konstrukte benötigen weiterhin gezielte Kompatibilitätsarbeit. Die Ausgabe enthält nur Theme-Regeln für Klassen, die der jeweilige Build aufgelöst hat.
