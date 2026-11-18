# Schritt 39: Modern-Icon-Sass-Shim

## Beobachtete Lücke

Das Modern-Theme lädt `icon.js`, um `parseIconArgs` für das `icon`-Mixin zu registrieren. Der Compiler stellt diese Funktion bereits im Sass-Prelude bereit, warnte aber dennoch über den JavaScript-Import.

## Verhalten

- Der bekannte Import `icon.js` wird wie die vorhandenen `font-icon.js`- und `extensions.js`-Shims ohne Warnung entfernt.
- Das Sass-Prelude bleibt die Implementierung von `parseIconArgs`.
- Nicht nachgebildete Fashion-JavaScript-Imports bleiben gewarnt.

## Prüfungen

- Sass-Integrationstest importiert `icon.js`, ruft `parseIconArgs` auf und prüft die erzeugten Icon-Deklarationen ohne Warnungen.
- Modern-Build des `default`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Dies implementiert keine allgemeine Fashion-JavaScript-Laufzeit und adressiert nicht `functions.js` oder `intersect-arguments`.
