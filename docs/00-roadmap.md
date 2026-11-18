# Roadmap: Sencha-Ext-JS-Compiler in Go

## Ziel

Ein eigenständiger Compiler in Go, der Sencha-Ext-JS-Projekte reproduzierbar baut. Der Umfang soll schrittweise anhand beobachtbaren Verhaltens und realer Projekt-Fixtures festgelegt werden. Als Referenz für Kompatibilität dienen Builds mit der jeweils festgelegten Sencha-Cmd-Version.

Die Implementierung startet mit einem kleinen, durch Tests abgesicherten vertikalen Schnitt statt mit einer vorschnellen Nachbildung aller Sencha-Cmd-Funktionen.

## Arbeitsregeln

- Für jeden Umsetzungsschritt eine Markdown-Datei in `docs/` anlegen, bevor oder während der Schritt umgesetzt wird.
- Verhalten zuerst anhand eines minimalen Fixtures beschreiben und mit der Referenz vergleichen, sofern Sencha Cmd verfügbar ist.
- Jede Funktion als klar abgegrenzten Schritt implementieren; Schnittstellen und Entscheidungen in der jeweiligen Schrittdokumentation festhalten.
- Referenz-Builds und Go-Builds mit denselben Eingaben vergleichen. Nichtdeterministische Metadaten und bekannte Abweichungen explizit dokumentieren.
- Keine Abhängigkeit von Sencha Cmd zur Laufzeit des Go-Compilers einführen.

## Vorgeschlagene Schritte

1. **Kompatibilitätsziel und Fixtures festlegen** (`docs/01-kompatibilitaetsziel.md`): Ext JS 8.0.0, Sencha Cmd 8.0.0 und ein erfolgreich mit Cmd gebautes Modern-Toolkit-Fixture.
2. **Erster Go-Build** (`docs/02-erster-go-build.md`): Go-Modul und CLI, minimales `app.json`-Modell, statische Ext-Klassenmetadaten, Abhängigkeitsordnung und einfaches JavaScript-Bundle.
3. **Klassen-Overrides** (`docs/03-overrides.md`): `app.json`-Override-Pfade erfassen und Overrides nur bei geladenem Ziel an der richtigen Stelle einordnen.
4. **Erstes Realprojekt** (`docs/04-default-realprojekt.md`): `default` als Universal-App einlesen, Toolkit-Klasspfade auswählen und reale Abweichungen von einem gültigen Sencha-Build erfassen.
5. **Xtype- und Alias-Auflösung** (`docs/05-xtype-aufloesung.md`): Klassen aus `alias`, `xtype`, `controller`, `viewModel`, `plugins` und `Ext.widget()` statisch finden.
6. **Projektmodell erweitern** (`docs/06-projektmodell.md`): `workspace.json`, Ext-SDK-Version und transitive Package-Classpaths.
7. **Watch-Preview-Server** (`docs/07-watch-preview.md`): Go-Build lokal ausliefern, Quellen beobachten und Browser bei erfolgreichem Neubau aktualisieren.
8. **JavaScript-Parsing erweitern** (`docs/08-javascript-parser.md`): Lexer/Parser und eng begrenzte Sencha-kompatible Quelltransformationen ausbauen.
9. **Abhängigkeitssemantik** (`docs/09-abhaengigkeiten.md`): dynamische Anforderungen, Loader-Pfade und Framework-Klassen ergänzen.
10. **Sencha-spezifische Transformationen** (`docs/10-transformationen.md`): Präprozessoren, Makros und Framework-Konventionen implementieren.
11. **Assets und Build-Ausgabe** (`docs/11-assets-und-ausgabe.md`): zusätzliche JS-, CSS- und Resource-Einträge aus `app.json` deterministisch ausgeben und über die Preview ausliefern.
12. **Minifizierung und Source Maps** (`docs/12-minifizierung.md`): optionale token-aware Minifizierung und token-/spaltenbasierte Source Maps; Sencha-kompatible Optimierungen weiter ergänzen.
13. **Kompatibilitätsausbau** (`docs/13-kompatibilitaet.md`): weitere App-, Workspace-, Paket-, Theme- und Produktions-Build-Fälle ergänzen.
14. **Theme-Package-Abhängigkeiten** (`docs/14-theme-packages.md`): das ausgewählte Theme-Package und seine transitiven SDK-Package-Abhängigkeiten in den Build aufnehmen.
15. **Fashion-Variablenreferenzen** (`docs/15-fashion-variablen.md`): einfache `dynamic($variable)`-Zuweisungen in Sass-Variablen normalisieren und deren Package-Dateien dependency-geordnet laden.
16. **Package-Sass-`etc`** (`docs/16-package-sass-etc.md`): `sencha.sass.etc`-Einstiegsdateien und unterstützte lokale Imports in den Sass-Kontext aufnehmen.
17. **Framework-Class-SCSS** (`docs/17-framework-sass-src.md`): `sass.src`-Quellen der toolkit- und core-spezifischen Framework-Packages in die Stylesheet-Pipeline aufnehmen.
18. **Package-Sass-Mixins** (`docs/18-package-sass-mixins.md`): `@include`-Aufrufe zulassen, wenn das Mixin in den geladenen Package-Sass-Quellen definiert ist.
19. **Fashion-Dynamic-Sass-Ausdrücke** (`docs/19-fashion-dynamic-ausdruecke.md`): `dynamic()`-Wrapper um einzeilige Sass-kompatible Variablen-Ausdrücke entfernen und mit Dart Sass auswerten.
20. **Theme-Sass-Imports** (`docs/20-theme-sass-imports.md`): Theme-`sass.etc`-Einstiege und ihre unterstützten Imports nach den Package-Variablen verarbeiten.
21. **Theme-Function-Kontext** (`docs/21-theme-function-context.md`): Theme-Funktionen vor Sass-Variablen verfügbar machen und Fashion-Kompatibilitätsfälle für importierte Theme-Quellen absichern.
22. **SDK-Root-Sass-Kontext** (`docs/22-sdk-root-sass.md`): `sass.etc`, `sass.var` und `sass.src` aus dem Ext-SDK-Root-Manifest in die Compiler-Pipeline einbeziehen.
23. **Theme-Bildpfade** (`docs/23-theme-background-images.md`): einfache `theme-background-image()`-Aufrufe auf kopierte Package-Bilder unter `resources/shared` abbilden.
24. **CSS-Basis-Mixins** (`docs/24-css-basis-mixins.md`): gebräuchliche Sass-Mixins wie `user-select` und `box-shadow` für Framework-Class-SCSS bereitstellen.
25. **CSS-Transform-Mixins** (`docs/25-css-transform-mixins.md`): einfache `rotate()`- und `scale()`-Mixin-Aufrufe in Framework-SCSS kompilieren.
26. **Mehrzeilige Fashion-Variablen** (`docs/26-mehrzeilige-dynamic-werte.md`): balancierte mehrzeilige `dynamic()`-Zuweisungen, insbesondere Sass-Maps, normalisieren.
27. **Fashion-Null-Aliase** (`docs/27-fashion-null-aliase.md`): direkte Theme-Variablenreferenzen auf `null`-Werte im Sass-Kontext erhalten.
28. **Fashion-Dart-Sass-Kompatibilität** (`docs/28-fashion-dart-sass-kompatibilitaet.md`): ausgewählte Fashion-Syntax aus Modern-Theme-Sass vor der Kompilierung in Dart-Sass-kompatible Funktions- und Mixin-Aufrufe überführen.
29. **Package-Sass-Einstiegsdateien** (`docs/29-package-sass-einstiegsdateien.md`): explizite `.scss`-Dateien aus `sencha.sass.src` als eigenständige Package-Stylesheets verarbeiten und ihre Imports beobachten.
30. **Statische Fashion-Ressourcenpfade** (`docs/30-fashion-ressourcenpfade.md`): literale `get-resource-path()`-Aufrufe für nach `resources/shared` kopierte Package-Ressourcen in relative CSS-Pfade umwandeln.
31. **Compass-Sass-Kompatibilität** (`docs/31-compass-sass-kompatibilitaet.md`): bekannte Compass-Importpfade auf die vom Compiler bereitgestellten CSS-Kompatibilitätsmixins abbilden.
32. **Package-Sass-Variablenimports** (`docs/32-package-sass-variablenimports.md`): auflösbare lokale Sass-Imports aus `sass.var`-Dateien in der Variablenabhängigkeitsauflösung berücksichtigen.
33. **Verschachtelte CSS-Imports** (`docs/33-verschachtelte-css-imports.md`): CSS-`@import url(...)` innerhalb von Sass-Mixins unverändert an Dart Sass weitergeben.
34. **Bekannte Fashion-JavaScript-Sass-Shims** (`docs/34-fashion-javascript-sass-shims.md`): JavaScript-Imports, deren Funktionen bereits im Sass-Prelude nachgebildet sind, ohne Warnung überspringen.
35. **Lokale Sass-Parameter** (`docs/35-lokale-sass-parameter.md`): Funktions- und Mixin-Parameter bei der Prüfung auf fehlende Package-Variablen als lokale Namen behandeln.
36. **Unbenutzte Fashion-JavaScript-Imports** (`docs/36-unbenutzte-fashion-js-imports.md`): `functions.js` ohne Warnung auslassen, wenn die registrierten Funktionen im geladenen Sass nicht verwendet werden.

## Architekturleitplanken

- **CLI-Schicht:** nimmt Optionen entgegen und delegiert an den Build-Ablauf.
- **Projektmodell:** normalisiert Konfiguration, Pfade und Eingabedateien.
- **Compiler-Pipeline:** getrennte Phasen für Auflösung, Transformation und Ausgabe; Phasen sollen ohne CLI testbar sein.
- **Dateisystemgrenze:** Builds arbeiten über explizite Eingabe- und Ausgabe-Optionen, damit Tests temporäre Verzeichnisse verwenden können.
- **Determinismus:** stabile Sortierung, keine implizite Abhängigkeit von Verzeichnis-Aufzählungsreihenfolgen und nachvollziehbare Diagnosen.
- **Kompatibilitätstests:** kleine Fixtures pro Feature plus End-to-End-Vergleich von Build-Artefakten.

## Festlegungen und offene Punkte

- Erstes Referenzprojekt: `testdata/fixtures/minimal-modern/`; erster Modus: Development.
- Reale Projekte liegen unter `testdata/ext-build/`; `default` verwendet aktuell Ext JS 8.0.0.43 und dient als erstes größeres Zielprojekt.
- Compilerlogik und JavaScript-Metadatenscanner werden in Go implementiert; das Go-Modul hat bisher keine externen Abhängigkeiten.
- Vorgemerkte spätere Erweiterungen stehen in [`docs/wishlist.md`](wishlist.md).
- Unterstützte Zielbetriebssysteme und der vollständige Kompatibilitätsumfang müssen noch festgelegt werden.

## Definition eines abgeschlossenen Schritts

Ein Schritt gilt als abgeschlossen, wenn seine Dokumentation Verhalten und Grenzen festhält, ein passendes Fixture oder einen begründeten Test enthält und die zugehörigen Prüfungen erfolgreich laufen. Bekannte Abweichungen von Sencha Cmd werden nicht stillschweigend als Kompatibilität behandelt.
