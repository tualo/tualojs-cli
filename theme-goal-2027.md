# tualo-2027

Ziel ist es ein Theme für classic zu derstellen, welches 
sass funktionen erneuert, welche auf deprecated stehen.

das ergebnis soll dem Neumorhphic-Flat entsprechen,
jedoch, wo möglich neue css methoden nutzen.

gleich lautende attribut-eigenschaften, sollen 
wenn möglich in root variablen central untergebracht 
werden.

Basis: /Users/thomashoffmann/Documents/Projects/go/extjs-compiler/testdata/ext-8.0.0/classic/theme-neutral

`theme-basic` is not present in the local Ext JS 8.0.0 SDK. `theme-neutral` was selected as the abstract Classic base.

Ziel: /Users/thomashoffmann/Documents/Projects/go/extjs-compiler/testdata/ext-8.0.0/classic/theme-neumorhphic-2027


Zum Testen:
Links zu: /Users/thomashoffmann/Documents/Projects/go/extjs-compiler/testdata/sample/ext/classic/theme-neumorhphic-2027

## Aktueller Stand

- Das Theme ist als `theme-neumorhphic-2027` angelegt und erweitert `theme-neutral`.
- Die Palette ist in Sass-Theme-Variablen zentral definiert und als CSS Custom Properties verfügbar.
- Legacy-Farb-Funktionsnamen aus dem Ext-JS-8-Theme werden über `sass:color`-Funktionen kompatibel gehalten.
- `$menu-component-margin` ist im Kind-Theme explizit gesetzt, da `theme-base` die Variable in `Menu.scss` direkt benötigt.
- Die Sample-App verwendet das neue Theme; ihre lokale SDK-Kopie enthält das Theme-Package.
