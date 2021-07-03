#!/bin/bash -e

cd "$(dirname "$0")"

set -x
for file in *.go
do
  # Build statically
  go build -ldflags="-extldflags=-static" -o build/ "$file"
done
