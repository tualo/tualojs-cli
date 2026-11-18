# Schritt 31: Compass-Sass-Kompatibilität

## Beobachtete Lücke

Das Ext-SDK importiert `compass`, `compass/css3` und `compass/css3/images`. Compass ist im Compiler nicht installiert; die Imports wurden deshalb als nicht auflösbar gewarnt, obwohl die für die geladenen Theme-Quellen benötigten CSS-Basis-Mixins bereits durch den Sass-Prelude abgedeckt sind.

## Verhalten

- Die bekannten Compass-Importpfade werden als virtuelle, leere Importmodule behandelt; ihre Importe werden ohne Warnung entfernt.
- Die Sass-Basis-Mixins des Compilers (`border-radius`, einzelne Ecken, `box-shadow`, `opacity`, `user-select`, `rotate` und `scale`) bleiben die tatsächlich bereitgestellte Kompatibilitätsschicht.
- Andere nicht auflösbare Sass-Imports werden weiterhin gewarnt.

## Prüfungen

- Sass-Integrationstest kompiliert einen Compass-Import mit einem bereitgestellten Mix-in und prüft, dass kein Import-Warning entsteht.
- Classic-/Modern-Builds des `default`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Dies implementiert Compass nicht vollständig. Compass-Funktionen und Mixins außerhalb des expliziten Compiler-Preludes bleiben ununterstützt.
