# Schritt 20: Theme-Sass-Imports

## Beobachtete Abweichung

Theme-Packages deklarieren `sass.etc`-Einstiege, die lokale SCSS-Partials für Theme-Variablen und Mixins importieren. Diese Theme-Einstiege wurden bisher pauschal ausgelassen, obwohl Package-Variablen und Sass-Mixins inzwischen verfügbar sind.

## Verhalten

- Theme-Sass-Funktionen und -Mixins aus lokalen Imports werden vor Package-`sass.var` geladen; der verbleibende Theme-`sass.etc`-Einstieg läuft danach und vor App-`sass.var`.
- Unterstützte lokale `@import`-/`@use`-/`@forward`-Partials werden aufgelöst, kompiliert und beobachtet.
- Definierte Package-Mixins aus Theme-Partials können von Class-SCSS verwendet werden.
- Eingebundene `dynamic()`-Werte ohne direkte Normalisierung werden über eine Sass-Identitätsfunktion ausgewertet.
- Unterstützte Box-Funktionen erhalten den von Fashion implizit erwarteten `null`-Fallback für `$fallbackBox`.
- Unaufgelöste und Fashion-JavaScript-Imports werden gewarnt und ausgelassen.
- Partials mit verschachtelten `@mixin`-Definitionen werden gewarnt und ausgelassen, da Dart Sass diese Sencha-Fashion-Struktur nicht unterstützt.

## Prüfungen

- Fake-SDK-Integrationstest prüft Theme-`etc`, einen importierten Theme-Mixin, Sass-Variablen und Watch-Pfade.
- Modern-/Classic-Builds des `default`-Realprojekts sowie `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Fashion-JavaScript-Funktionen, verschachtelte Mixins, generierte UI-Mixins, dynamisch erzeugte Theme-Symbole und weitere Fashion-spezifische Sass-Semantik sind nicht enthalten.
