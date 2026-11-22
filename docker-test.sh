#!/bin/bash
./build.sh
docker cp dist/linux/arm64/tualojs ${1:-wvd_local}:/usr/local/bin/tualojs
docker cp testdata/sample/ext/classic/theme-neumorhphic-2027 ${1:-wvd_local}:/home/tualo/www/html/server/ext-build/fb_wvd/build/production/Tualo/ext/classic
