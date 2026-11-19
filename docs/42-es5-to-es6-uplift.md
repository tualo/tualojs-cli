# Schritt 42: ES5→ES6-Uplift, sicherer Funktions-Scope-Kern

## Ziel

`-upgrade-es5` aktiviert optional eine konservative Modernisierung von ES5-`var`-Deklarationen. Ohne diese Option bleiben Eingaben unverändert. Die bestehende Gegenrichtung `-normalize-let-const` bleibt separat verfügbar.

## Unterstützter Fall

- Berücksichtigt werden ausschließlich einfache, initialisierte Deklarationen eines einzelnen Bezeichners direkt im Körper einer klassischen `function`.
- `const` wird gewählt, wenn nach der Initialisierung keine direkte Zuweisung, Inkrementierung oder Dekrementierung auf die Bindung erkannt wird; sonst `let`.
- Globale Variablen, `for`-Kopfdeklarationen, Block-deklarierte `var`, mehrfache Deklarationen, Parameter-Re-deklarationen und Bindungen mit Lesezugriff vor der Deklaration bleiben unverändert.
- Bindungen, die von inneren Funktionen erfasst werden könnten, werden konservativ ausgelassen. Funktionen mit Arrow-Funktionen, `eval()` oder `with` werden ebenfalls nicht modernisiert.
- Deklarationen ohne explizites Semikolon oder über mehrere Zeilen bleiben zunächst unverändert.

## Prüfungen

- Transformationstests prüfen die Auswahl von `const`/`let` und wesentliche Fälle, die unverändert bleiben müssen.
- Ein Build-Test prüft sowohl die explizite Aktivierung als auch das unveränderte Standardverhalten.
- `go test -count=1 ./...`, `go vet ./...` und ein Node-Syntaxcheck der Beispielausgabe.

## Grenzen und nächste Erweiterungen

Dies ist der erste sichere Kern, kein vollständiger Scope- oder AST-Parser. Arrow-Function-Scopes, Closure-Analyse, Parameter- und Block-Scope-Fixtures sowie Referenzvergleiche mit Sencha Cmd bleiben Erweiterungen dieses Uplift-Schritts.
