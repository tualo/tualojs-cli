# Schritt 14: Theme-Package-Abhängigkeiten

## Beobachtete Abweichung

Das Sencha-Produktionsmanifest des Classic-Realprojekts listet `theme-triton` und dessen Abhängigkeiten `theme-neptune`, `theme-neutral`, `theme-base`, `font-ext` und `font-awesome` als eingebundene Packages. Der Go-Build wählt zwar `theme-triton` als Theme, fügt es bisher aber nicht dem Package-Graphen hinzu.

## Verhalten

- Das ausgewählte Theme aus `app.json.theme`, dem passenden `builds`-Profil oder dem Toolkit-Standard wird als optionale Package-Anforderung an das SDK übergeben.
- Ist das Theme-Package im SDK vorhanden und für das Toolkit geeignet, werden seine Theme-Vererbung (`sencha.extend`) und `sencha.requires` transitiv in Dependency-Reihenfolge aufgelöst.
- Klassenpfade, Overrides, Ressourcen und Sass-Eingaben der geladenen Theme-Packages verwenden die bereits vorhandenen Package-Pipelines.
- Fehlende Theme-Packages werden ignoriert, damit Apps ohne lokal verfügbares Theme-SDK weiterhin wie zuvor bauen können.
- Toolkit-fremde Theme-Packages werden entsprechend Schritt 13 ausgelassen.

## Prüfungen

- Fake-SDK-Build prüft die Package-Reihenfolge eines Theme-Vererbungs-/Requires-Graphen sowie kopierte Theme-Ressourcen und Watch-Pfade.
- Realprojekt-Builds prüfen die Theme-Ketten: Classic nimmt `theme-base`, `theme-neutral`, `theme-neptune` und `theme-triton` auf; Modern nimmt `theme-base`, `theme-neptune` und `theme-material` auf.
- `go test ./...`, `go vet ./...` und `node --check` der erzeugten Bundles.

## Grenzen

Die Package-Abhängigkeiten werden aufgelöst; Sencha-Fashion-Theme-Kontexte und deren Sass-Mixins werden damit noch nicht nachgebildet.
