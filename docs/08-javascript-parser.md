# Schritt 8: JavaScript-Parsing und erste Transformation

## Scanner

Der Go-Scanner erfasst Zeichenpositionen und unterscheidet Identifier, Strings, Template-Literale, Regex-Literale und Satzzeichen. Klammern in Kommentaren, Strings und Regex-Ausdrücken beeinflussen dadurch nicht mehr die `Ext.define`-Konfigurationsanalyse. Er unterstützt statische Objekt- und Factory-Funktionsformen von `Ext.define`.

## Transformation

Mit `-normalize-let-const` senkt der Go-Compiler direkte `let`-/`const`-Deklarationen im Funktions-Scope zu `var`. Das ist standardmäßig ausgeschaltet. Sencha Cmds kompiliertes Classic-Bundle enthält im Realprojekt eine Funktionssignatur mit Parameter `tablename` und eine gleichnamige `let`-Deklaration im Funktionsrumpf; die Option normalisiert diesen Konflikt. Deklarationen in inneren Blöcken und `for`-Köpfen sowie Strings und Kommentare bleiben unangetastet.

## Prüfungen

- Unit-Tests decken Funktions-Scope, Parameterkonflikte, `for`-Köpfe, Strings und verschachtelte Blöcke ab.
- Der generierte Classic-Build von `testdata/ext-build/default/` besteht `node --check`.
- `go test ./...` und `go vet ./...` laufen erfolgreich.

## Grenzen

Das ist keine allgemeine ES6-zu-ES5-Transpilation und kein vollständiger Sencha-Klassencompiler. Arrow-Funktionen, Destructuring, Block-Scope-Semantik, async/await und Ext-spezifische Klassen-Transformationen benötigen weitere Parser- und Compilerphasen.
