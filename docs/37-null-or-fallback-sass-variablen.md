# Schritt 37: Null-Or-Fallback in Sass-Variablen

## Beobachtete Lücke

Modern-Themes definieren optionale Werte als `null` und verwenden sie in direkten Alias-Zuweisungen oder `or`-Fallbacks. Bisher wurde oft die gesamte `sass.var`-Datei ausgelassen, sobald irgendeine Zuweisung auf einen solchen Wert verwies.

## Verhalten

- Direkte Aliase auf zuvor als `null` markierte Theme-Variablen bleiben auch dann erhalten, wenn weitere Variablen in derselben Datei definiert sind.
- Ein direkter null-valued Operand in einer Sass-`or`-Fallback-Zuweisung wird als sicher behandelt; Dart Sass wertet den Ausdruck aus.
- Andere Berechnungen mit null-valued Variablen bleiben weiterhin gewarnt und ausgelassen, sofern sie nicht durch einen späteren gezielten Kompatibilitätsschritt abgedeckt sind.

## Prüfungen

- Fixtures prüfen lokale Aliase und `or`-Fallbacks, die dependency-geordnete Verarbeitung sowie das weiterhin ausgelassene unsichere Rechenbeispiel.
- Modern-Build des `default`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Null-Semantik wird nur für direkte Aliase und direkte `or`-Operanden berücksichtigt. Weitere spezifische Fälle sind in separaten Schritten dokumentiert.
