#!/bin/bash

DIST_DIR="${DIST_DIR:-/dist}"
mkdir -v "$DIST_DIR"

for i in ./build/* ./offensive-tor-toolkit*.tar.gz
do
  cp -v "$i" "$DIST_DIR"
done
