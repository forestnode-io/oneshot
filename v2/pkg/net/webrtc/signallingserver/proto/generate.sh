#!/bin/bash
SRC_DIR=proto/
DST_DIR=.
FILES=$(find proto -iname "*.proto")

protoc \
    -I=$SRC_DIR \
    --proto_path=$SRC_DIR \
    --go_opt=paths=source_relative \
    --go_out=$DST_DIR \
    --go-grpc_opt=paths=source_relative \
    --go-grpc_out=$DST_DIR \
    $FILES 2>&1 > /dev/null