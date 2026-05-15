#!/bin/sh
set -eu

cd "$(dirname "$0")/.."

env_file=".env.local"
client_id="${GZY_GITHUB_CLIENT_ID:-}"
if [ -z "$client_id" ] && [ -f "$env_file" ]; then
  # shellcheck disable=SC1090
  client_id="$(. "$env_file"; printf '%s' "${GZY_GITHUB_CLIENT_ID:-}")"
fi

version="${VERSION:-dev}"
ldflags="-X main.version=$version"
if [ -n "$client_id" ]; then
  ldflags="$ldflags -X main.defaultGitHubClientID=$client_id"
fi

mkdir -p dist

build_one() {
  goos="$1"
  goarch="$2"
  name="$3"
  out="dist/gzy"
  if [ "$goos" = "windows" ]; then
    out="dist/gzy.exe"
  fi
  GOOS="$goos" GOARCH="$goarch" go build -ldflags "$ldflags" -o "$out" ./cmd/gzy
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
