#!/bin/bash
rm -rf /home/tualo/www/html/server/ext-build/fb_wvd

docker cp \
wvd_local:/home/tualo/www/html/server/ext-build/fb_wvd \
testdata/ext-build/fb_wvd

rm -rf /Users/thomashoffmann/Documents/Projects/go/extjs-compiler/testdata/ext-build/fb_wvd/fb_wvd/build/production/Tualo/ext
docker cp \
wvd_local:/var/sencha/sdk/ext-8.0.0 \
testdata/ext-build/fb_wvd/fb_wvd/build/production/Tualo/ext

go run ./cmd/extjsc build --app testdata/ext-build/fb_wvd/build/production/Tualo --toolkit classic --mode production --minify --out ./tmp/default-classic-go
cat /Users/thomashoffmann/Documents/Projects/go/extjs-compiler/testdata/ext-build/fb_wvd/build/production/Tualo/ext/build/ext-all.js testdata/ext-build/fb_wvd/build/production/Tualo/tmp/default-classic-go/app.js > testdata/ext-build/fb_wvd/build/production/Tualo/tmp/default-classic-go/app.all.js
docker cp testdata/ext-build/fb_wvd/build/production/Tualo/tmp/default-classic-go/app.all.js  wvd_local:/home/tualo/www/html/server/ext-cache/fb_wvd/classic/app.js

docker exec wvd_local chown -R tualo:tualo ext-cache