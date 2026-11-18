# Schritt 30: Statische Fashion-Ressourcenpfade

## Beobachtete Lücke

Package-Einstiegsdateien wie `font-ext/sass/src/all.scss` verwenden `get-resource-path('fonts')`, um Dateien aus Package-Ressourcen einzubinden. Die Ressourcen werden bereits nach `resources/shared` kopiert, aber Sass-Quellen mit diesem Fashion-Aufruf wurden bisher ausgelassen.

## Verhalten

- Statische `get-resource-path()`-Aufrufe mit String-Pfad und optionalem `path`-/`shared`-Pool werden in relative URLs unter `../../shared/` übersetzt.
- Query- und Fragmentteile im Pfad bleiben erhalten.
- Aufrufe mit dynamischem Pfad oder `packageName`-Kontext werden nicht umgeschrieben und bleiben als Fashion-Abweichung erkennbar.
- Die Übersetzung wird sowohl auf Package-`sass.src`-Einstiegsdateien als auch auf weitere kompilierte Sass-Quellen angewendet.

## Prüfungen

- Unit-/Integrationstest prüft CSS-URL, statische Pfadnormalisierung und Package-Einstiegsdatei.
- Classic-Build des `testdata/sample`-Realprojekts sowie `go test -count=1 ./...`, `go vet ./...` und `node --check`.

## Grenzen

Fashion-`output-paths`, dynamische Pfade und `packageName`-Auflösung werden nicht simuliert. Die Übersetzung setzt voraus, dass die referenzierte Ressource in `resources/shared` liegt.
