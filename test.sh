#!/usr/bin/env bash

set -x

./build.sh && sqlite3 < test.sql
