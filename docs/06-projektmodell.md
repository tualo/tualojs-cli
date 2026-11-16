# Schritt 6: Workspace, SDK und Packages

## Workspace und Ext-SDK

Der Compiler liest `workspace.json` im App-Verzeichnis und folgt `frameworks.ext.path`. `package.json` am SDK-Root muss das Framework `ext` und die Kompatibilitätsversion **8.0.0** ausweisen. Ist in `workspace.json` eine SDK-Version eingetragen, muss sie mit der tatsächlichen SDK-Version übereinstimmen.

`workspace.json` darf Sencha-übliche `//`- und `/* ... */`-Kommentare enthalten. Fehlt die Workspace-Datei bei einem App ohne Package-Abhängigkeiten, kann der erste App-Quellen-Build weiterhin laufen. Bei konfigurierten Packages ist ein auflösbares Ext-SDK erforderlich.

## Package-Abhängigkeiten

- Package-Namen aus `app.json.requires` werden gegen Metadaten unter `packages/`, `modern/` und `classic/` im SDK aufgelöst.
- `sencha.name`, Verzeichnisname und `sencha.alternateName` sind gültige Namen.
- Transitive Package-Abhängigkeiten aus `sencha.requires` werden zyklusfrei und deterministisch sortiert.
- `sencha.classpath`- und `sencha.overrides`-Pfade expandieren `${package.dir}` und `${toolkit.name}`. Fehlende toolkit-spezifische Package-Pfade sind optional.
- Aufgelöste Package-Quellen nehmen an der gleichen Klassen-, Alias-, Override- und Abhängigkeitsauflösung wie App-Quellen teil.

## Prüfungen

- Integrationstest mit einem Fake-SDK für SDK-Version, Workspace-JSONC, Package-Aliase, transitive Abhängigkeiten und Classpath-Reihenfolge.
- Inkompatible SDK-Kompatibilitätsversionen werden vor dem Build abgewiesen.
- Das echte Ext-JS-8-Fixture unter `testdata/fixtures/minimal-modern/` löst sein lokales SDK erfolgreich auf.

## Grenzen

Package-JavaScript-Klassen werden aufgelöst und gebündelt. Ext-Framework-Klassen werden für den Preview-Loader indiziert und aus dem vorgebauten SDK-Runtime geladen, aber noch nicht selbst kompiliert. Package-Ressourcen, Theme/Sass-Verarbeitung, Toolkit-spezifische Package-Filter und Package-Build-Profile sind noch nicht implementiert.
