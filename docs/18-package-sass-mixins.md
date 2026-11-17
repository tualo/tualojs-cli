# Schritt 18: Definierte Package-Sass-Mixins

## Beobachtete Abweichung

Package-`sass.etc`-Einstiege laden bereits lokale Mixins. Die Class-SCSS-Auflösung verwirft derzeit jedoch jeden `@include`-Aufruf pauschal, sodass auch normale Sass-Mixins aus diesen Einstiegen ungenutzt bleiben.

## Verhalten

- Der Compiler sammelt statische `@mixin`-Definitionen aus Sass-Einstiegen und deren beobachteten lokalen Imports.
- Class-SCSS mit `@include` wird kompiliert, wenn alle statisch referenzierten Mixins definiert sind.
- Nicht bekannte Mixins führen zur bestehenden Warnung; das betroffene Stylesheet wird ausgelassen.
- `@include`-Aufrufe in Package-Variablendateien und Fashion-generierte Mixins sind nicht Teil dieses Schritts.

## Prüfungen

- Dart-Sass-Integrationstest für ein Package-Mixin aus einem importierten `sass.etc`-Partial und eine Klasse, die es verwendet.
- Realprojekt-Builds prüfen, dass unbekannte Fashion-Mixins weiterhin übersprungen werden.
- `go test ./...`, `go vet ./...`, Modern-/Classic-Builds und `node --check`.

## Grenzen

Erkannt werden statisch geschriebene Mixin-Namen. Fashion-Mixins, dynamische Mixinnamen und Mixins aus nicht aufgelösten Imports bleiben unsupported.
