# Schritt 28: Fashion-Dart-Sass-Kompatibilität

## Beobachtete Abweichung

Das Modern-Theme-Sass nutzt einige Fashion-Konstrukte, die Dart Sass nicht akzeptiert: doppelte benannte Mixin-Argumente ohne Doppelpunkt, lokal deklarierte Sass-Funktionen und Theme-Farb-Funktionen mit absichtlich ausgelassenem `$color`-Argument.

## Verhalten

- Doppelte Argumente der Form `$name $name` in `@include`-Aufrufen werden als `$name: $name` normalisiert.
- Die verschachtelten Modern-Theme-Funktionen `calcPos` und `calcOffset` werden zu Sass-Funktionen auf Dateiebene und ihre aufrufenden Stellen überführt; bisher implizite äußere Variablen werden explizit übergeben.
- `button-hovered-color`, `button-pressed-color`, `button-focused-color` und `button-disabled-color` erhalten für `$color` den Fashion-kompatiblen Standardwert `null`.

## Prüfungen

- Unit-Tests decken die einzelnen Sass-Normalisierungen ab.
- Classic- und Modern-Builds des `default`-Realprojekts kompilieren erfolgreich mit Dart Sass.
- `go test -count=1 ./...`, `go vet ./...` und `node --check` für die erzeugten Bundles.

## Grenzen

Dies ist eine gezielte Übersetzung der beobachteten SDK-Konstrukte, kein allgemeiner Sass-/Fashion-Parser. Weitere Fashion-spezifische Funktions-, Mixin- oder JavaScript-Semantik bleibt offen.
