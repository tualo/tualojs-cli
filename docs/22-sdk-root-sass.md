# Schritt 22: SDK-Root-Sass-Kontext

## Beobachtete Abweichung

Das mitgelieferte Ext-SDK enthält globale Sass-Einstiege unter `ext/sass/etc`. Diese laden unter anderem Variablen wie `$fa-var-caret-down` aus SDK-Package-Partials. Die bisherigen Sass-Pipelines lasen nur App-, Package- und Theme-Manifeste, nicht aber das Sass-Kontextfeld im Root-`ext/package.json`.

## Verhalten

- `sass.etc`, `sass.var` und `sass.src` des durch `workspace.json` ausgewählten SDK-Root-Manifests werden aufgelöst und innerhalb des SDK-Verzeichnisses gehalten.
- SDK-Root-`etc` und unterstützte lokale Imports werden vor Package-`etc` und Package-Variablen geladen; importierte Definitionen stehen dem Variablen- und Mixin-Resolver zur Verfügung.
- Variablenzuweisungen mit einfachen Vorwärtsreferenzen innerhalb derselben Datei werden vor dem Kompilieren dependency-geordnet.
- Statische Sass-Helfer aus SDK-Packages, die über den Root-`etc`-Einstieg importiert werden, werden ebenfalls vor Package-Variablen geladen; so stehen etwa Font-Awesome-Variablen wie `$fa-var-caret-down` zur Verfügung.
- Fehlende Feature-Flags `$include-*` erhalten standardmäßig `false`; deklarierte Werte aus den Sass-Quellen bleiben maßgeblich.
- Ein begrenzter Satz von Fashion-/CSS-Helfern (`map_*`, `file_join`, `parseIconArgs`, `parseFontIconArgs`, `border-radius`, `opacity`, `user-select`, `box-shadow`, `rotate`, `scale`) ermöglicht die Kompilierung bekannter Theme-Mixins. Der Classic-Realprojekt-Build erzeugt damit auch CSS für `@include extjs-button-small-ui`.
- Doppelte SDK-Root-Wurzeln aus App- und Framework-Auflösung werden dedupliziert und beobachtet.
- Fashion-JavaScript-Imports und `get-resource-path`-Auswertungen, die einen Sencha-Buildkontext voraussetzen, werden weiterhin diagnostiziert und ausgelassen.

## Prüfungen

- Fake-SDK-Integrationstest prüft SDK-Root-`sass.etc`, einen importierten Variablenpartial, Class-SCSS und Watch-Pfade.
- Modern-/Classic-Builds des `default`-Realprojekts; der Classic-Output enthält `.x-btn-pivot-iconbutton-small` aus `extjs-button-small-ui`.
- `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

SDK-Root-Sass wird in den Kontext aufgenommen. Sencha-Fashion-JavaScript, vollständige Ressourcenpfad-Expansion, alle Theme-Mixins und produktionsspezifische Sass-Konfigurationen bleiben offen.
