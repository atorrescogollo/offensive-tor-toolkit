#!/bin/bash -xe

for f in *.go
do
  # Build statically
  go build -ldflags="-extldflags=-static" -o ./build/${f/.go} $f

  # Convert to base64
  base64 ./build/${f/.go} > ./build/${f/.go}.b64
done

# Compress files
tar -cvzf offensive-tor-toolkit.tar.gz --transform s/build/offensive-tor-toolkit/ ./build/
