# Schritt 38: Fashion-Box-Rotationen

## Beobachtete Lücke

Modern-Themes verwenden Fashion-Funktionen wie `rotate90($box)`, um Box-Listen für horizontale und vertikale UIs umzudrehen. Optionales `$box: null` wird von Fashion als `null` weitergereicht; ohne entsprechende Sass-Shims wird die Variablendatei vom Null-Validator ausgelassen.

## Verhalten

- Der Sass-Prelude stellt `rotate90`, `rotate180` und `rotate270` für Box-Listen mit ein bis vier Werten bereit und erhält `null` als `null`.
- Variablenzuweisungen, deren direkter Rotationsoperand als `null` bekannt ist, werden ebenfalls als `null` markiert und können dependency-geordnet weiterverwendet werden.
- Andere Berechnungen mit `null`-Werten bleiben weiterhin gewarnt und ausgelassen.

## Prüfungen

- Tests prüfen Box-Rotationen, null-sichere Funktionsauswertung und dependency-Auflösung einer rotierten Null-Variable.
- Modern-Build des `default`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Unterstützt werden nur die expliziten Rotationsfunktionen und direkte Variablenargumente. Beliebige Fashion-Funktionen mit Null-Eingaben werden nicht verallgemeinert.
