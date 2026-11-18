# Schritt 23: Theme-Bildpfade

## Beobachtete Abweichung

Class-SCSS-Dateien aus SDK-Packages verwenden `theme-background-image('pfad', 'png')`. Ohne Sencha Fashion wurde das Stylesheet bisher ausgelassen, obwohl die Package-Ressource vorhanden ist und in `resources/shared` kopiert wird.

## Verhalten

- Statische Stringargumente für Pfad und optionale Dateiendung werden in relative CSS-URLs unter `../../shared/images/` umgewandelt; UX-Package-Bilder verwenden `../../shared/ux/images/`.
- Wenn `app.json.resources` auf `${build.id}` verweist und der vorherige Sencha-Build Toolkit-Ressourcen enthält, werden diese zusätzlich nach `resources/shared` kopiert und beobachtet.
- Sass-Interpolation im Pfad bleibt für spätere Mixinauswertung erhalten.
- Nichtliterale Argumente werden weiterhin diagnostiziert und das Stylesheet wird ausgelassen.
- Die Funktion `theme_image_exists` und Sencha-spezifische Theme-Suchpfade werden nicht nachgebildet.

## Prüfungen

- Fake-SDK-Integrationstest prüft die CSS-URL, Package-Bilder unter `resources/shared/images` und vorhandene Toolkit-Bilder aus `${build.id}` unter `resources/shared/...`.
- Das `default`-Realprojekt prüft erzeugte UX-Bild-URLs und übernimmt vorhandene Toolkit-Bildressourcen aus dem Sencha-Build.
- Classic-/Modern-Builds des `default`-Realprojekts, `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Nur Package-Bilder mit dem konfigurierten `output: "shared"`-Ziel werden direkt adressiert. Pfadpräfixe aus `get-resource-path`, Theme-Fallbacks und Fashion-Bildexistenzprüfungen bleiben offen.
