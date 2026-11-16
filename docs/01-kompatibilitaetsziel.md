# Schritt 1: Kompatibilitätsziel und Fixtures

## Festgelegt

- **Ext JS:** 8.0.0
- **Sencha Cmd:** 8.0.0
- **Implementierungssprache:** Go (Projektziel aus `goals.md`)
- **Erstes Fixture:** `testdata/fixtures/minimal-modern/`, Modern Toolkit

## Erstes Fixture

Das kleine Fixture enthält `app.json`, den Einstieg `app.js`, eine Application-Klasse, eine Main-View mit Greeting-, Created-, Required- und WidgetOnly-Komponenten sowie einen Override für die Main-View. Es deckt Vererbung, `requires`, `mainView`, `Ext.create`, `Ext.require`, xtype-Aliase und `override` ab.

Die Klassenquellen liegen unter `app/` passend zu `classpath`. Framework-Klassen wie `Ext.app.Application`, `Ext.Panel` und `Ext.Component` stammen aus dem Ext-JS-SDK und werden nicht im Fixture dupliziert.

Das SDK liegt lokal unter `testdata/ext-8.0.0/`; dessen `package.json` meldet Ext JS **8.0.0.43** (Kompatibilitätsversion 8.0.0). `workspace.json` referenziert es relativ, ohne eine SDK-Kopie im Fixture abzulegen. Das SDK-Verzeichnis wird von Git ignoriert.

## Referenzwerkzeug-Status

Sencha Cmd **8.0.0.48** startet regulär; `sencha help app` listet den Build-Befehl. Die Version wird im Cmd-Banner angezeigt.

Der Referenz-Build läuft mit `sencha app build --dev` im Fixture-Verzeichnis erfolgreich. `bootstrap.json` enthält die App-Klassen `GoCompilerFixture.Application`, `GoCompilerFixture.view.main.Main` und `GoCompilerFixture.view.main.Greeting`. Der erste Versuch vor dem Eintragen des SDK-Pfads schlug mit `Unable to load required framework: ext@null` fehl.

## Noch festzulegen

- Welche Betriebssysteme und Architekturen zunächst unterstützt werden
- Ob Parser-/Minifier-Abhängigkeiten zulässig sind oder sämtliche Bestandteile ausschließlich in Go geschrieben werden sollen

## Vorgehen für Kompatibilität

Für jedes unterstützte Verhalten wird ein minimales Fixture mit Eingaben, Referenz-Aufruf und erwarteten Ausgaben dokumentiert. Sencha-Cmd-Builds dienen als Referenz für die entsprechenden End-to-End-Tests. Abweichungen werden pro Funktion festgehalten.

## Abnahmekriterium dieses Schritts

Ext-JS-Version 8.0.0, Sencha-Cmd-Version 8.0.0 und ein minimales Modern-Toolkit-Fixture sind festgelegt. Der Development-Referenzbuild ist erfolgreich und die App-Klassen sind im erzeugten Bootstrap nachweisbar.
