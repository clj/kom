#!/usr/bin/env bash

set -e
set -x

./build.sh
trap "trap - SIGTERM && kill -- -$$" SIGINT SIGTERM EXIT
python3 test_server.py &
sleep 1
sqlite3 < test.sql
