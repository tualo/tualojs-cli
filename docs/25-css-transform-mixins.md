# Schritt 25: CSS-Transform-Mixins

## Beobachtete Abweichung

Classic-Framework-SCSS aus `core` verwendet `@include rotate(...)` und `@include scale(...)`. Diese einfachen Transform-Mixins sind im ausgewählten Classic-Kontext nicht immer als Theme-Quelle vorhanden und die Stylesheets wurden deshalb ausgelassen.

## Verhalten

- Fehlende `rotate($value)`- und `scale($value)`-Mixins werden als `transform`- und `-webkit-transform`-Deklarationen bereitgestellt.
- Eine gleichnamige Definition aus geladenen SDK-Sass-Quellen hat Vorrang.
- Andere nicht bekannte Mixins werden weiterhin gewarnt und ausgelassen.

## Prüfungen

- Dart-Sass-Integrationstest prüft CSS-Ausgabe für beide Transform-Mixins.
- Classic-/Modern-Builds des `default`-Realprojekts sowie `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Komplexe transform-/Compass-Mixins und vollständige Browserpräfix-Semantik sind nicht Teil dieses Schritts.
