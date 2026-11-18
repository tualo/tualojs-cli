# Schritt 35: Lokale Sass-Parameter

## Beobachtete Lücke

Die Prüfung von Package-`sass.src`-Dateien behandelte alle `$name`-Referenzen wie Package-Variablen. Parameter lokaler Sass-Funktionen und Mixins, beispielsweise `$value1` und `$value2` in `add-if($value1, $value2)`, wurden dadurch fälschlich als fehlende Variablen gemeldet.

## Verhalten

- Parameter von `@function`- und `@mixin`-Definitionen gelten bei der Referenzprüfung als lokal definiert.
- Andere Referenzen bleiben weiterhin gegen lokale Deklarationen und verfügbare Package-/Theme-Variablen geprüft.
- Sass-Parameter-Standardwerte behalten ihre Referenzen auf externe Variablen und werden weiterhin validiert.
- Das `TreeItem.scss`-Framework-Stylesheet wird dadurch nicht länger wegen seiner `add-if($value1, $value2)`-Parameter ausgelassen.

## Prüfungen

- Unit-Test prüft lokale Funktions-/Mixin-Parameter sowie weiterhin erkannte undefinierte globale Referenzen.
- Classic-Build von `testdata/sample` sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Lexikalische Variablen aus Sass-Kontrollstrukturen wie `@each` und `@for` werden nicht allgemein analysiert.
