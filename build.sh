#!/usr/bin/env bash

case $(uname -s) in
    Darwin*)   ext="dylib";;
    *)         ext="so";;
esac

set -x

go build -buildmode=c-shared -o kom.$ext kom.go plugins.go inventree.go
