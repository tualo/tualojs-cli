# Schritt 9: Eager- und Deferred-Abhängigkeiten

## Verhalten

`requires` und `mixins` sind eager Abhängigkeiten: ihre Klassen werden vor der verwendenden Klasse angeordnet. `uses` bezeichnet optionale bzw. später genutzte Klassen; diese werden nach der deklarierenden Klasse in das Bundle aufgenommen.

Statische `Ext.Loader.setPath`-Mappings auf vorhandene Verzeichnisse innerhalb der App werden ebenfalls durchsucht. Klassen aus diesen Pfaden können so über statische `Ext.create`-Aufrufe oder Wildcards in die Klassenauflösung gelangen. Nicht vorhandene und externe Serverpfade bleiben zur Laufzeit dynamisch.

Die Trennung verhindert, dass ein `uses`-Verweis fälschlich einen Ladezyklus erzeugt. Der Test deckt etwa `Main uses Optional` und `Optional requires Main` ab: `Main` wird zuerst ausgegeben, `Optional` danach.

## Prüfungen

- Parser-Test für `uses`-Metadaten.
- Build-Test für die Reihenfolge und das Zusammenspiel mit einer rückwärts gerichteten eager `requires`-Kante.
- Parser- und Build-Test für direkte sowie objektbasierte `Ext.Loader.setPath`-Mappings.
- `go test ./...` und `go vet ./...`.

## Grenzen

Nur statische Klassennamen werden aufgelöst. Bedingt oder dynamisch berechnete `uses`-Einträge und die vollständige Laufzeitsemantik des Ext-Classloaders bleiben zukünftige Schritte.
