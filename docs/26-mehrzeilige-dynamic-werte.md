# Schritt 26: Mehrzeilige Fashion-Variablen

## Beobachtete Abweichung

Ext-JS-Theme-Packages definieren Sass-Maps und mehrzeilige Werte mit `dynamic(( ... ))`. Der bisherige Normalisierer verarbeitet nur einzeilige Zuweisungen und hat diese Theme-Variablendateien ausgelassen.

## Verhalten

- `dynamic()` in einfachen Sass-Variablenzuweisungen wird auch über Zeilenumbrüche hinweg bis zur passenden schließenden Klammer erkannt.
- Verschachtelte Klammern, zitierte Strings und Kommentare beeinflussen die Klammerzuordnung nicht.
- Der Wrapper wird entfernt und ein fehlendes Semikolon am Assignment-Ende ergänzt.
- `dynamic()` außerhalb einer Variablenzuweisung sowie nicht geschlossene oder verschachtelte Fashion-Ausdrücke bleiben unsupported.

## Prüfungen

- Unit-Test für eine mehrzeilige Map-Zuweisung mit verschachtelten Maps.
- Dart-Sass-Integrationstest prüft die Weiterverarbeitung der normalisierten Map.
- Modern-/Classic-Realprojekt-Builds, `go test ./...`, `go vet ./...` und `node --check`.

## Grenzen

Fashion-spezifische Map-Auflösung und spätere dynamische Theme-Overrides werden nicht nachgebildet; nur die Sass-kompatible Initialzuweisung wird an Dart Sass übergeben.
