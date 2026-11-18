# Schritt 40: Verschachtelte Fashion-Dynamic-Werte

## Beobachtete Lücke

Einige Fashion-Theme-Variablen sind Sass-Maps in einem äußeren `dynamic()`-Wrapper und enthalten zusätzlich einzelne `dynamic()`-Wrapper an Map-Werten. Bisher wurden solche Zuweisungen abgelehnt, obwohl nach Entfernen der Fashion-Marker ein gültiger Sass-Ausdruck übrig bleibt.

## Verhalten

- Innere `dynamic()`-Wrapper innerhalb eines bereits unterstützten Variablen-Ausdrucks werden rekursiv entfernt.
- Balancierte Klammern und Strings/Kommentare werden beim Finden der Wrapper berücksichtigt.
- Andere nicht unterstützte `dynamic()`-Formen außerhalb einfacher Variablenzuweisungen bleiben abgelehnt.

## Prüfungen

- Fixture prüft mehrzeilige Sass-Maps mit verschachtelten `dynamic()`-Werten und das resultierende CSS.
- Modern-Build des `default`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Dies entfernt ausschließlich verschachtelte Fashion-Wrapper. Fashion-Ausdrucksauswertung, bedingte Theme-Semantik und nicht-Sass-kompatible Werte bleiben offen.
