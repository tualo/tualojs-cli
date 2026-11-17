# Schritt 13: Toolkit-fremde Package-Abhängigkeiten

## Beobachteter Fall

Das Universal-App-Fixture `testdata/ext-build/default/build/production/Tualo/` deklariert `font-awesome` in `app.json.requires`. Das SDK-Package `font-awesome` ist im Manifest als Classic-only markiert. Ein Modern-Build muss diese Package-Abhängigkeit auslassen, statt sie als nicht gefunden abzubrechen; andere Abhängigkeiten aus demselben Manifest bleiben davon unberührt.

## Verhalten

- Package-Metadaten werden unabhängig vom ausgewählten Toolkit indiziert.
- Ist ein explizit angefordertes Package vorhanden, aber für ein anderes Toolkit markiert, wird es für diesen Build ausgelassen. Seine transitiven Abhängigkeiten und Quelldateien werden nicht geladen.
- Das Manifest eines ausgelassenen Packages bleibt ein Watch-Pfad, damit Änderungen an seiner Toolkit-Zuordnung einen Neubau auslösen.
- Packages ohne Toolkit-Markierung sowie Packages für das ausgewählte Toolkit werden weiterhin transitiv aufgelöst.
- Ein Package-Name, der im SDK überhaupt nicht existiert, bleibt ein Fehler.
- Wird eine Klasse aus einem ausgelassenen Package im JavaScript statisch angefordert, wird sie nicht aus diesem Package gebündelt; die Auflösung bleibt dem Laufzeit-Loader überlassen.

## Prüfungen

- Fake-SDK-Integrationstest: Eine globale `requires`-Liste enthält ein Classic-only Package; Modern lässt es aus und baut weiter, Classic lädt es.
- Regressionstest: Ein tatsächlich unbekannter Package-Name wird weiterhin mit einer Diagnose abgewiesen.
- Der Modern-Production-Build des `default`-Realprojekts läuft durch und listet `font-awesome` nicht unter den geladenen Packages.

## Grenzen

Toolkit-fremde Package-Abhängigkeiten werden ausgelassen, auch wenn App-Code Klassen daraus statisch anfordert; solche Klassen können zur Laufzeit weiterhin unerfüllt sein. Bedingte Package-Abhängigkeiten und plattformabhängige Package-Metadaten sind nicht Teil dieses Schritts.
