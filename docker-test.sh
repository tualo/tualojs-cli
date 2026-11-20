#!/bin/bash
project_dir="$(cd -- "$(dirname -- "$0")" && pwd)"
rm -rf testdata/fb_wvd

docker cp wvd_local:/home/tualo/www/html/server/ext-build/fb_wvd testdata/
rm -rf "$project_dir/testdata/fb_wvd/build/production/Tualo/ext"
docker cp wvd_local:/var/sencha/sdk/ext-8.0.0 testdata/fb_wvd/build/production/Tualo/ext

time go run ./cmd/tualojs build --app testdata/fb_wvd/build/production/Tualo --toolkit classic --mode production --minify --out ./tmp/default-classic-go
echo "/* Building default classic Go app...*/" > testdata/fb_wvd/build/production/Tualo/tmp/default-classic-go/app.all.js
cat "$project_dir/testdata/fb_wvd/build/production/Tualo/ext/build/ext-all.js" \
    testdata/fb_wvd/build/production/Tualo/tmp/default-classic-go/app.js \
    >> testdata/fb_wvd/build/production/Tualo/tmp/default-classic-go/app.all.js
docker cp testdata/fb_wvd/build/production/Tualo/tmp/default-classic-go/app.all.js  wvd_local:/home/tualo/www/html/server/ext-cache/fb_wvd/classic/app.js

docker exec wvd_local chown -R tualo:tualo ext-cache


dist/tualojs-macos-arm64 build -app ./testdata/sample -out build-go -json -base-path /static/example/
