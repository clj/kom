#!/usr/bin/env bash

src=$1
version=$2

cd $src
for lib in kom-*; do
    ext=${lib##*.}
    filename=${lib%%.*}
    filename=${filename/darwin/macos}
    zip=${filename/kom/kicad-odbc-middleware-$version}.zip
    cp $lib kom.$ext
    zip $zip kom.$ext
    rm kom.$ext
done
