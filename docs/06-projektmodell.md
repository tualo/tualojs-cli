# Schritt 6: Workspace, SDK und Packages

## Workspace und Ext-SDK

Der Compiler liest `workspace.json` im App-Verzeichnis und folgt `frameworks.ext.path`. `package.json` am SDK-Root muss das Framework `ext` und die Kompatibilitätsversion **8.0.0** ausweisen. Ist in `workspace.json` eine SDK-Version eingetragen, muss sie mit der tatsächlichen SDK-Version übereinstimmen.

Pfadlisten wie `classpath`, `overrides` und `requires` akzeptieren sowohl JSON-Arrays als auch Sencha-übliche komma-separierte Strings.

`classpath`, `overrides` sowie lokale JavaScript-, CSS- und Ressourcenpfade expandieren `${app.dir}`, `${app.name}` und `${toolkit.name}` aus App-Konfiguration und ausgewähltem Toolkit. Aufgelöste Quellpfade bleiben auf das App-Verzeichnis begrenzt.

`workspace.json` darf Sencha-übliche `//`- und `/* ... */`-Kommentare enthalten. Fehlt die Workspace-Datei bei einem App ohne Package-Abhängigkeiten, kann der erste App-Quellen-Build weiterhin laufen. Bei konfigurierten Packages ist ein auflösbares Ext-SDK erforderlich.

## Package-Abhängigkeiten

- Package-Namen aus `app.json.requires` werden gegen Metadaten unter `packages/`, `modern/` und `classic/` im SDK aufgelöst.
- Packages mit `sencha.toolkit: "classic"` oder `"modern"` werden nur für das passende ausgewählte Toolkit aufgelöst.
- Zusätzlich werden `requires` aus dem ausgewählten `classic`-/`modern`-Abschnitt und dem gleichnamigen `builds`-Profil berücksichtigt.
- `sencha.name` (oder bei Standard-SDK-Manifests das oberste Feld `name`), Verzeichnisname und `alternateName` sind gültige Namen.
- Transitive Package-Abhängigkeiten aus `sencha.requires` oder dem obersten Feld `requires` werden zyklusfrei und deterministisch sortiert.
- `sencha.classpath`-/`sencha.overrides`-Pfade sowie die entsprechenden obersten Manifestfelder expandieren `${package.dir}` und `${toolkit.name}`. Fehlende toolkit-spezifische Package-Pfade sind optional.
- `${package.locale}` wird aus `app.json.locale` expandiert, damit locale-spezifische Package-Overrides gefunden werden.
- Aufgelöste Package-Quellen nehmen an der gleichen Klassen-, Alias-, Override- und Abhängigkeitsauflösung wie App-Quellen teil.

## Prüfungen

- Integrationstest mit einem Fake-SDK für SDK-Version, Workspace-JSONC, Package-Aliase, transitive Abhängigkeiten und Classpath-Reihenfolge.
- Integrationstest für SDK-Package-Manifeste mit Standardfeldern auf Root-Ebene, einschließlich `exporter` und transitivem `core`.
- Integrationstest für App-Pfadmakros in Classpath, Overrides und Assets.
- Inkompatible SDK-Kompatibilitätsversionen werden vor dem Build abgewiesen.
- Das echte Ext-JS-8-Fixture unter `testdata/fixtures/minimal-modern/` löst sein lokales SDK erfolgreich auf.

## Grenzen

Package-JavaScript-Klassen werden aufgelöst und gebündelt; deklarierte `sencha.resources` transitive Packages werden in denselben Ressourcen-Ausgabeordner kopiert und beobachtet. Ext-Framework-Klassen werden für den Preview-Loader indiziert und aus dem vorgebauten SDK-Runtime geladen, aber noch nicht selbst kompiliert. Theme/Sass-Verarbeitung und Package-Build-Profile sind noch nicht implementiert.
