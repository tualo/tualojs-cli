# Schritt 19: Sass-kompatible Fashion-`dynamic()`-Ausdrücke

## Beobachtete Abweichung

Theme-Variablen verwenden `dynamic()` nicht nur für Literale und direkte Variablenreferenzen, sondern auch für Sass-Ausdrücke wie `mix(...)`, `round(...)` und einfache Arithmetik. Diese Zuweisungen wurden bisher ausgelassen, obwohl Dart Sass die inneren Ausdrücke auswerten kann.

## Verhalten

- In einer alleinstehenden Variablenzuweisung wird der äußere `dynamic(...)`-Wrapper entfernt; verschachtelte Klammern und zitierte Strings werden beim Erkennen des Endes berücksichtigt.
- Der normalisierte Sass-Ausdruck wird von Dart Sass ausgewertet. Abhängigkeiten auf Variablen werden wie in Schritt 15 dependency-geordnet geladen.
- Fehlt am ursprünglichen Assignment das Semikolon, wird es bei der Normalisierung ergänzt.
- `dynamic()` innerhalb von CSS-Deklarationen, nicht geschlossene Ausdrücke, verschachtelte Fashion-`dynamic()`-Aufrufe sowie Fashion-spezifische Funktionen bleiben unsupported.

## Prüfungen

- Unit-Test für Literal-, Variablen-, Funktions- und Arithmetikausdrücke einschließlich fehlendem Semikolon.
- Dart-Sass-Integrationstest prüft die Auswertung von `mix()` und Arithmetik.
- Modern-/Classic-Builds des `default`-Realprojekts sowie `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Unterstützt werden Ausdrücke, die Dart Sass selbst auswerten kann. Sencha-Fashion-Funktionen, dynamische Theme-Vererbung und die vollständige Fashion-Wertauflösung werden weiterhin nicht nachgebildet.
