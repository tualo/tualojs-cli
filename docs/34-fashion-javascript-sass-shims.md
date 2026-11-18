# Schritt 34: Bekannte Fashion-JavaScript-Sass-Shims

## Beobachtete Lücke

Das SDK lädt einzelne Fashion-JavaScript-Dateien per Sass-`@import`. Zwei davon registrieren Funktionen, die der Compiler bereits in seinem Sass-Prelude bereitstellt: `font-icon.js` (`parseFontIconArgs`) und `extensions.js` (`file_join`). Diese Importe wurden trotzdem als nicht unterstützte JavaScript-Imports gewarnt.

## Verhalten

- Die bekannten Imports `font-icon.js` und `extensions.js` werden ohne Warnung entfernt, da ihre Sass-Entsprechungen im Compiler-Prelude vorhanden sind.
- Andere Fashion-JavaScript-Imports bleiben gewarnt und werden ausgelassen.

## Prüfungen

- Sass-Integrationstest importiert beide Dateien, verwendet die Sass-Funktionen und prüft die CSS-Ausgabe ohne Import-Warnungen.
- Classic-Build von `testdata/sample` sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Dies ist eine explizite Liste vorhandener Sass-Shims und keine Fashion-JavaScript-Laufzeit. `functions.js` mit dynamischer Mixin-Argumentauflösung bleibt ununterstützt.
