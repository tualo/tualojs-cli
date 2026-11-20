# tualojs – Ziele und Umsetzungsstand

## Festgelegte Ziele

- Einen eigenständigen Compiler in Go entwickeln, der Ext-JS-Anwendungen anhand ihres Projekt- und SDK-Aufbaus baut.
- Schrittweise ausgewählte Build-Verhaltensweisen von Sencha Cmd nachbilden. Das Kompatibilitätsziel ist Ext JS 8.0.0; als Referenz dient Sencha Cmd 8.0.0.
- Anwendungen für Classic und Modern verarbeiten und JavaScript-Klassen, Abhängigkeiten, Overrides und weitere statische Ext-JS-Referenzen erkennen.
- Neben JavaScript auch Stylesheets, Sass und statische Ressourcen in den Build einbeziehen.
- Verhalten mit kleinen Fixtures und – soweit vorhanden – mit Sencha-Cmd-Referenz-Builds prüfen. Entscheidungen, Grenzen und Abweichungen in `docs/` dokumentieren.
- Sencha Cmd nicht als Laufzeitabhängigkeit des Go-Compilers voraussetzen.

## Erreichte Ziele

- **Eigenständige Go-CLI:** `tualojs build` erstellt einen Build; `tualojs watch` überwacht Eingaben und stellt eine lokale Preview bereit.
- **Ext-JS-Projektauflösung:** `app.json`, `workspace.json` und SDK-Manifeste werden ausgewertet. Ext-SDK-Kompatibilitätsversionen und Package-Abhängigkeiten werden geprüft und aufgelöst.
- **JavaScript-Bundling:** Statische Klassenmetadaten werden erkannt und Abhängigkeiten geordnet. Unterstützt sind unter anderem `requires`, `uses`, Overrides, Aliase und Xtypes sowie ausgewählte Sencha-Präprozessoren und Quelltransformationen.
- **Build-Artefakte:** Lokale Scripts, Stylesheets und Ressourcen können übernommen werden. Sass-Verarbeitung, Source Maps, optionale Minifizierung und ein JSON-Asset-Manifest sind vorhanden.
- **Entwicklungsfunktionen:** Watch-/Preview-Server und die optionale Einbindung der aufgelösten Ext-JS-Runtime in `app.js` sind umgesetzt.
- **Nachvollziehbare Entwicklung:** Die Umsetzungsschritte und bekannten Grenzen sind in `docs/` dokumentiert und durch Go-Tests abgesichert. `build.sh` erstellt Binärdateien für Linux, Windows und macOS auf amd64 und arm64.

## Umfang und Grenzen

tualojs bildet **nicht den vollständigen Funktionsumfang von Sencha Cmd** nach. Unterstützt werden die in den Schrittdokumentationen beschriebenen Fälle; dynamisch berechnete Klassenreferenzen, die vollständige Sencha-Fashion-Semantik und zahlreiche Sencha-spezifische Optimierungen bleiben außerhalb des aktuellen Umfangs. Die jeweilige Dokumentation hält weitere Einschränkungen fest.

Die Apache-2.0-Lizenz gilt für tualojs. Sie erteilt keine Rechte an Sencha Ext JS oder anderen Sencha-Produkten; dafür gelten deren jeweilige Lizenzbedingungen. Details stehen in `LICENSE` und `NOTICE`.
