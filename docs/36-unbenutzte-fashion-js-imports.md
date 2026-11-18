# Schritt 36: Unbenutzte Fashion-JavaScript-Imports

## Beobachtete Lücke

Das Core-Stylesheet importiert `functions.js`, das Fashion-Funktionen wie `map_merge` und `intersect_arguments` registriert. In Apps, deren geladenes Sass diese Funktionen nicht aufruft, wurde der Import dennoch als Warnung gemeldet.

## Verhalten

- Der Compiler ermittelt Funktionsaufrufe im geladenen Sass inklusive aufgelöster Imports.
- `functions.js` wird ohne Warnung ausgelassen, wenn weder `map_merge` noch `intersect-arguments` verwendet wird.
- Sobald eine dieser Funktionen verwendet wird, bleibt der Import als nicht unterstützte Fashion-JavaScript-Abhängigkeit sichtbar.
- Bereits vorhandene Shims für `font-icon.js` und `extensions.js` bleiben davon unabhängig.

## Prüfungen

- Fixture prüft den unbenutzten `functions.js`-Import sowie das Erkennen beider Funktionsaufrufe.
- Classic-Build von `testdata/sample` sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Dies implementiert `map_merge` oder `intersect_arguments` nicht. Verwendet ein Build diese Fashion-Funktionen, bleibt die Laufzeitabhängigkeit bestehen.
