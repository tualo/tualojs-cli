# Schritt 15: Fashion-Variablenreferenzen

## Beobachtete Abweichung

Die Theme-Packages in Ext JS definieren Sass-Variablen häufig als `dynamic($andereVariable)`. Bisher wurden nur `dynamic(<Literal>)`-Zuweisungen normalisiert. Theme-Variablendateien mit Referenzen auf Variablen aus anderen Dateien wurden deshalb ausgelassen.

## Verhalten

- Eine alleinstehende Variablenzuweisung der Form `$name: dynamic($andereVariable);` wird zu `$name: $andereVariable;` normalisiert.
- Package-Variablendateien mit noch nicht verfügbaren Variablen werden zunächst zurückgestellt. Nachgelagerte Definitionen werden zuerst geladen, danach die abhängigen Dateien.
- Referenzen innerhalb derselben Datei und bereits bekannte App-/Package-Variablen werden unterstützt.
- Alle entdeckten SCSS-Variablendateien bleiben Watch-Pfade, auch wenn ihre Referenzen nicht aufgelöst werden können.

## Prüfungen

- Unit-Test prüft die Sortierung einer vorwärts referenzierenden Variablendatei sowie die Watch-Pfade.
- Dart-Sass-Integrationstest prüft das kompilierte CSS einer Theme-Variablenreferenz.
- `dynamic()` außerhalb einer einfachen Variablenzuweisung bleibt unsupported.

## Grenzen

Fashion-Funktionen, dynamisch berechnete Werte, Imports, Interpolation, Mixins und zyklische oder unbekannte Variablenreferenzen werden weiterhin nicht ausgewertet; betroffene Package-Variablendateien werden mit Warnung übersprungen.
