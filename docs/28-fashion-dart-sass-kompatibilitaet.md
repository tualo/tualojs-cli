# Schritt 28: Fashion-Dart-Sass-Kompatibilität

## Beobachtete Abweichung

Das Modern-Theme-Sass nutzt einige Fashion-Konstrukte, die Dart Sass nicht akzeptiert: doppelte benannte Mixin-Argumente ohne Doppelpunkt, lokal deklarierte Sass-Funktionen und Theme-Farb-Funktionen mit absichtlich ausgelassenem `$color`-Argument.

## Verhalten

- Doppelte Argumente der Form `$name $name` in `@include`-Aufrufen werden als `$name: $name` normalisiert.
- Die verschachtelten Modern-Theme-Funktionen `calcPos` und `calcOffset` werden zu Sass-Funktionen auf Dateiebene und ihre aufrufenden Stellen überführt; bisher implizite äußere Variablen werden explizit übergeben.
- `button-hovered-color`, `button-pressed-color`, `button-focused-color` und `button-disabled-color` erhalten für `$color` den Fashion-kompatiblen Standardwert `null`.
- Verschachtelte Sass-Mixins ohne äußere Variablen werden auf Dateiebene gehoben; benötigte lexikalische Variablen werden als Parameter ergänzt.
- Die vier `border-*-radius`-Mixins für einzelne Ecken werden als Dart-Sass-kompatible Präfix-Mixins bereitgestellt.
- `border-top-radius` und `border-bottom-radius` werden ebenfalls als Präfix-Mixins bereitgestellt.
- `is-null()` behandelt `null` und Sass-`none` als leere Werte; die Theme-`min()`-/`max()`-Kompatibilität normalisiert solche Werte vor Vergleichen.
- Bei Sass-Variablendateien werden lokale Überschreibungen berücksichtigt, bevor Referenzen auf vorherige `null`-Werte als ungültig bewertet werden.
- In Font-Awesome-7-Icon-Maps wird die dort absichtlich maskierte `\#{ $icon }`-Interpolation vor der Dart-Sass-Kompilierung aktiviert; Icon-Regeln wie `.fa-check::before` erhalten damit den Unicode-Glyphwert statt des Sass-Ausdrucks als Text.

## Prüfungen

- Unit-Tests decken die einzelnen Sass-Normalisierungen sowie lokale Variablenüberschreibungen ab.
- Dart-Sass-Test prüft `is-null()` und die Theme-`min()`-/`max()`-Fälle mit `none`.
- Dart-Sass-Integrationstest prüft die Font-Awesome-Icon-Codepoints für reguläre und Brand-Icons.
- Classic- und Modern-Builds des `default`-Realprojekts kompilieren erfolgreich mit Dart Sass.
- `go test -count=1 ./...`, `go vet ./...` und `node --check` für die erzeugten Bundles.

## Grenzen

Dies ist eine gezielte Übersetzung der beobachteten SDK-Konstrukte, kein allgemeiner Sass-/Fashion-Parser. Weitere Fashion-spezifische Funktions-, Mixin- oder JavaScript-Semantik bleibt offen.
