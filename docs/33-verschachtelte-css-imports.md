# Schritt 33: Verschachtelte CSS-Imports

## Beobachtete Lücke

Das Theme-Base-Mixin `google-webfont` enthält ein verschachteltes CSS-`@import url(...)`. Der Import ist gültig und kann von Dart Sass zur Include-Stelle ausgegeben werden, wurde aber bisher wie ein verschachtelter Sass-Dateiimport behandelt und entfernt.

## Verhalten

- Verschachtelte `@import url(...)`-Direktiven bleiben unverändert im Sass-Quelltext.
- Verschachtelte Sass-Dateiimporte werden weiterhin übersprungen und gewarnt.
- Top-Level-CSS-Imports verhalten sich wie bisher.

## Prüfungen

- Sass-Integrationstest kompiliert ein Mixin mit verschachteltem CSS-URL-Import und prüft den ausgegebenen CSS-Import ohne Warnung.
- Classic-Build von `testdata/sample` sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Dies unterstützt keine Sass-Dateiimporte innerhalb von Mixins und verändert keine Import-URLs.
