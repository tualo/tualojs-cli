# Schritt 20: Theme-Sass-Imports

## Beobachtete Abweichung

Theme-Packages deklarieren `sass.etc`-Einstiege, die lokale SCSS-Partials für Theme-Variablen und Mixins importieren. Diese Theme-Einstiege wurden bisher pauschal ausgelassen, obwohl Package-Variablen und Sass-Mixins inzwischen verfügbar sind.

## Verhalten

- Theme-`sass.etc`-Einstiege werden nach Package-`sass.var` und vor App-`sass.var` verarbeitet.
- Unterstützte lokale `@import`-/`@use`-/`@forward`-Partials werden aufgelöst, kompiliert und beobachtet.
- Definierte Package-Mixins aus Theme-Partials können von Class-SCSS verwendet werden.
- Unaufgelöste und Fashion-JavaScript-Imports werden gewarnt und ausgelassen. Theme-Kontexte, deren Variablen Fashion-Funktionen wie `material-color()` benötigen, werden ebenfalls ausgelassen.
- Partials mit verschachtelten `@mixin`-Definitionen werden gewarnt und ausgelassen, da Dart Sass diese Sencha-Fashion-Struktur nicht unterstützt.

## Prüfungen

- Fake-SDK-Integrationstest prüft Theme-`etc`, einen importierten Theme-Mixin, Sass-Variablen und Watch-Pfade.
- Modern-/Classic-Builds des `default`-Realprojekts sowie `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Fashion-JavaScript-Funktionen, Fashion-Farbpaletten wie `material-color()`, verschachtelte Mixins, UI-Mixins, dynamisch erzeugte Theme-Symbole und weitere Fashion-spezifische Sass-Semantik sind nicht enthalten.
