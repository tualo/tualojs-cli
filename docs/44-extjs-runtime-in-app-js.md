# Schritt 44: Ext-JS-Runtime in `app.js`

## Ziel

`extjsc build -include-extjs` nimmt die für SDK und Toolkit aufgelöste Ext-JS-Runtime an den Anfang des erzeugten `app.js`-Bundles auf. Dadurch kann die Website statt einer separaten Ext-Runtime-Datei nur das Bundle laden.

## Verhalten

- Die Runtime stammt aus `FrameworkRuntimePath`, also aus der Toolkit-Konfiguration oder der SDK-Standarddatei.
- Der Include ist standardmäßig deaktiviert. Bei Aktivierung ohne SDK oder Toolkit bricht der Build mit einer klaren Fehlermeldung ab.
- Die Runtime wird unverändert eingebettet; ES5-Upgrade, `let`/`const`-Normalisierung und Minifizierung betreffen weiterhin nur Compiler-Anwendungssources.
- Der Runtime-Pfad wird als beobachtete Build-Abhängigkeit geführt und im Source-Map-Quellenverzeichnis ausgewiesen.
- Das JSON-Manifest kennzeichnet das Bundle mit `frameworkIncluded: true`; `app.js` bleibt ein einzelner JS-Eintrag.

## Prüfungen

- Integrationstest prüft Runtime-Reihenfolge, unveränderte Runtime-Inhalte, fortgesetzte Transformation des Anwendungscodes und die erforderliche SDK-Konfiguration.
- `go test -count=1 ./...` und `go vet ./...`.

## Verwendung

```sh
extjsc build -app ./app -out build-go -toolkit modern -include-extjs -json -base-path /static/example/
```

## Grenzen

Die Option ist für den einmaligen `build` vorgesehen.
