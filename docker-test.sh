#!/bin/bash
./build.sh
docker cp dist/linux/arm64/tualojs ${1:-wvd_local}:/usr/local/bin/tualojs