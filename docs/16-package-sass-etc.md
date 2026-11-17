# Schritt 16: Package-Sass-`etc`

## Beobachtete Abweichung

Ext-SDK-Packages wie `core` deklarieren in `sencha.sass.etc` einen Einstieg mit globalen Sass-Variablen und Imports für Funktionen und Mixins. Bisher wurden nur App-`sass.etc`-Einstiege verarbeitet; dadurch fehlten unter anderem `$prefix` und Package-Sass-Hilfsdateien.

## Verhalten

- `sencha.sass.etc` geladener Packages wird als SCSS-Einstieg aufgelöst; Package-Pfade müssen innerhalb des Package-Verzeichnisses liegen.
- `etc`-Einstiege aus Nicht-Theme-Packages laufen nach App-`sass.etc` und vor Package-`sass.var`; einfache Theme-`etc`-Einstiege laufen nach Package-`sass.var` und vor App-`sass.var`.
- Lokale `@import`-/`@use`-/`@forward`-Partials werden beim Watchen berücksichtigt. Beim Kompilieren werden relative Imports auf ihre Quelldateien aufgelöst.
- Theme-`etc`-Einstiege mit Imports in den Fashion-Kontext werden mit Warnung ausgelassen. Unaufgelöste Imports und Fashion-JavaScript-Imports werden ebenfalls ausgelassen und gewarnt.
- Direkte Variablendeklarationen aus Package-`etc` stehen der Package-Variablenauflösung und Class-SCSS-Prüfung zur Verfügung.
- Verarbeitete Package-`etc`-Einstiege und ihre unterstützten lokalen Imports bleiben Watch-Pfade.

## Prüfungen

- Fake-SDK-Integrationstest prüft Package-`etc`, `dynamic()`-Normalisierung, relative Imports, Variablenverwendung in Class-SCSS und Watch-Pfade.
- Dart Sass kompiliert das erzeugte CSS.
- `go test ./...`, `go vet ./...` und Modern-/Classic-Realprojekt-Builds.

## Grenzen

Das Laden der Package-`etc`-Einstiege stellt noch keinen vollständigen Sencha-Fashion-Kontext her. Fashion-spezifische `@include`-Aufrufe, Theme-`etc`-Imports, Fashion-JavaScript-Funktionen und dynamisch berechnete Variablen bleiben begrenzt.
