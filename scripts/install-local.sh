#!/bin/sh
# Build gzy from source (with .env.local baked in) and install it to
# the user's local bin directory.
#
# Usage:
#   ./scripts/install-local.sh
#   PREFIX=/usr/local ./scripts/install-local.sh
set -eu

cd "$(dirname "$0")/.."

bin_dir="${PREFIX:-$HOME/.local/bin}"
mkdir -p "$bin_dir"

OUT="$bin_dir/gzy" ./scripts/build.sh

echo
echo "installed: $bin_dir/gzy"
case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) echo "Add this to your shell profile: export PATH=\"$bin_dir:\$PATH\"" ;;
esac
