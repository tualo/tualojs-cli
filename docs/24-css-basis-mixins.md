# Schritt 24: CSS-Basis-Mixins

## Beobachtete Abweichung

Framework-Class-SCSS und Package-SCSS verwenden einfache CSS-Mixins wie `user-select` und `box-shadow`. Diese Namen fehlen je nach ausgewähltem Toolkit in den geladenen Theme-Partials, sodass sonst unterstützbare Stylesheets übersprungen werden.

## Verhalten

- `user-select($value)` erzeugt die gängigen vendor-präfixierten und standardisierten CSS-Deklarationen.
- `box-shadow($shadow...)` erzeugt die WebKit- und Standard-Deklarationen.
- Ein Sass-Mixin aus dem SDK hat Vorrang; die eingebauten Versionen ergänzen nur fehlende Definitionen.
- Andere unbekannte Mixins werden weiterhin mit Warnung ausgelassen.

## Prüfungen

- Dart-Sass-Integrationstest prüft die erzeugten Deklarationen für beide Mixins.
- Modern-/Classic-Builds des `default`-Realprojekts, `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Komplexe Compass-/Fashion-Mixins, Theme-UI-Generierung und weitere vendor-spezifische Mixins bleiben außerhalb dieses Schritts.
