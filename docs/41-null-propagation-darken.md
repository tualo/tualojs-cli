# Schritt 41: Null-Propagation für `darken()`

## Beobachtete Lücke

Das UX-Gauge-Theme leitet `$gauge-needle-fill` mit `darken($slider-track-border-color, 40%)` ab, obwohl diese Slider-Farbe im ausgewählten Modern-Theme absichtlich `null` ist. Fashion propagiert diesen optionalen Farbwert als `null`; Dart Sass lehnt die Funktion auf `null` ab.

## Verhalten

- Eine direkte Package-Sass-Variablenzuweisung `darken($name, ...)` wird als `null` behandelt, wenn `$name` bereits als null-valued bekannt ist.
- Die Sass-Variablendatei und ihre weiteren unabhängigen Werte bleiben dadurch ladbar.
- Andere Funktionen und beliebige Rechenausdrücke mit `null` bleiben außerhalb dieses Schritts.

## Prüfungen

- Tests prüfen die Null-Propagation, die resultierende Sass-Kompilierung und weiterhin abgelehnte nicht unterstützte Null-Ausdrücke.
- Modern-Build des `default`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Unterstützt wird nur `darken()` mit direkter null-valued Farbvariable als erstem Argument und als Package-Sass-Variablenzuweisung.
