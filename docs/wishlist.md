# Wishlist

## ES5 → ES6 JavaScript-Uplift (in Arbeit; erster Kern in Schritt 42)

Optionales Modernisieren älterer ES5-Quellen für ES6-Ausgabe, insbesondere sichere Umwandlung von `var`-Bindings in `let` beziehungsweise `const`.

### Anforderungen für die Umsetzung

- Nur mit expliziter Option aktivieren; vorhandene Quellen standardmäßig unverändert lassen.
- Scope- und Closure-Semantik erhalten; keine globalen Text-Ersetzungen.
- `const` nur verwenden, wenn keine spätere Zuweisung möglich ist; andernfalls `let`.
- Abdeckung für Parameter, Block-Scope, Schleifen, Closures und Mehrfachdeklarationen durch Fixtures und Tests.
- Ergebnis mit Sencha-Cmd-Referenzbuilds vergleichen.
- Der initiale Schritt 42 setzt den opt-in-Schalter und einen konservativen Kern für einfache Funktions-lokale `var`-Deklarationen um.

### Abgrenzung

Die aktuelle Option `-normalize-let-const` implementiert die Gegenrichtung für ausgewählte Funktions-Scope-Deklarationen (`let`/`const` → `var`). Sie bleibt davon unabhängig.
