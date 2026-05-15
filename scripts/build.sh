#!/bin/sh
# Build a single binary for the current platform with secrets baked in
# from .env.local (if present).
#
# Usage:
#   ./scripts/build.sh              # output: ./dist/gzy
#   OUT=/tmp/gzy ./scripts/build.sh
set -eu

cd "$(dirname "$0")/.."

env_file=".env.local"
client_id="${GZY_GITHUB_CLIENT_ID:-}"
if [ -z "$client_id" ] && [ -f "$env_file" ]; then
  # shellcheck disable=SC1090
  client_id="$(. "$env_file"; printf '%s' "${GZY_GITHUB_CLIENT_ID:-}")"
fi

version="${VERSION:-dev}"
out="${OUT:-./dist/gzy}"
mkdir -p "$(dirname "$out")"

ldflags="-X main.version=$version"
if [ -n "$client_id" ]; then
  ldflags="$ldflags -X main.defaultGitHubClientID=$client_id"
  echo "embedding GZY_GITHUB_CLIENT_ID from $env_file"
else
  echo "no GZY_GITHUB_CLIENT_ID found (browser auth will require runtime env var)"
fi

go build -ldflags "$ldflags" -o "$out" ./cmd/gzy
echo "built $out"
