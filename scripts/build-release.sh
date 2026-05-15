#!/bin/sh
set -eu

version="${VERSION:-dev}"
mkdir -p dist

build_one() {
  goos="$1"
  goarch="$2"
  name="$3"
  out="dist/gzy"
  if [ "$goos" = "windows" ]; then
    out="dist/gzy.exe"
  fi
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "-X main.version=$version" -o "$out" ./cmd/gzy
  if [ "$goos" = "windows" ]; then
    (cd dist && zip -q "$name" gzy.exe && rm gzy.exe)
  else
    (cd dist && tar -czf "$name" gzy && rm gzy)
  fi
}

build_one darwin amd64 gzy_Darwin_x86_64.tar.gz
build_one darwin arm64 gzy_Darwin_arm64.tar.gz
build_one linux amd64 gzy_Linux_x86_64.tar.gz
build_one linux arm64 gzy_Linux_arm64.tar.gz
build_one windows amd64 gzy_Windows_x86_64.zip
build_one windows arm64 gzy_Windows_arm64.zip
