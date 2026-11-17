# Schritt 21: Theme-Function-Kontext

## Beobachtete Abweichung

Theme-Variablen rufen Funktionen aus ihren eigenen `sass.etc`-Imports auf. Werden die Theme-Imports erst nach den Variablen ausgewertet, fehlen Funktionen wie `material-color()`. Weitere Fashion-Annahmen betreffen dynamische Wrapper in importierten Partials, optionale Box-Funktionsargumente und `null`-Werte.

## Verhalten

- Statische Sass-`@function`- und `@mixin`-Hilfsquellen aus Theme-Imports werden vor Package-Variablen geladen; dieselben Hilfsimports werden beim späteren Theme-`etc`-Einstieg nicht doppelt ausgegeben.
- Importierte `dynamic()`-Werte verwenden eine Sass-Identitätsfunktion. Geladene Theme-Funktionen wie `material-color()` können dadurch von Package-Variablen genutzt werden.
- Klassische Box-Funktionen mit Fashion-typischem `$fallbackBox` erhalten einen optionalen `null`-Standardwert.
- Variablenquellen, deren Ausdrücke von einer als `null` definierten Theme-Variablen abhängen, werden gewarnt und ausgelassen.
- Verschachtelte Mixins und Theme-Kontexte mit nicht unterstützten Imports bleiben diagnostiziert und ausgelassen.

## Prüfungen

- Integrationstest für importierte dynamische Theme-Werte, lokale Hilfsfunktionen und Mixins.
- Fake-SDK-Test für Variablenabhängigkeiten auf `null`.
- Modern-/Classic-Realprojekt-Builds, `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Sencha-Fashion-JavaScript, UI-Mixins wie `extjs-button-small-ui`, verschachtelte Sass-Mixins und die vollständige dynamische Theme-Semantik sind nicht implementiert.
