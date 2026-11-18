# Schritt 27: Fashion-Null-Aliase

## Beobachtete Abweichung

Theme-Dateien enthalten häufig Zuweisungen wie `$tab-color-over: dynamic($tab-color)`, wobei `$tab-color` absichtlich `null` ist. Diese direkten Alias-Zuweisungen wurden bisher wie fehlerhafte Sass-Ausdrücke behandelt und die gesamte Variablendatei ausgelassen.

## Verhalten

- Direkte Variablenaliase auf eine als `null` definierte Fashion-Variable bleiben im Sass-Kontext erhalten und werden selbst als `null` markiert.
- Abgeleitete Ausdrücke wie `darken($null-color, 10%)` werden weiterhin gewarnt und ausgelassen.
- Die Erkennung gilt für direkte einzeilige Variablenzuweisungen, einschließlich normalisierter `dynamic($variable)`-Formen.

## Prüfungen

- Unit-Test prüft die dependency-geordnete Auflösung eines Alias auf `null` und das Auslassen eines abgeleiteten Ausdrucks.
- Classic-/Modern-Builds des `default`-Realprojekts sowie `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Fashion-Wertvererbung, bedingte Theme-Overrides und berechnete Ausdrücke mit `null`-Werten werden nicht ausgewertet.
