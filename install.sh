#!/bin/sh
set -eu

repo="${GZY_REPO:-frimpsss/gzy}"
version="${GZY_VERSION:-latest}"
bin_dir="${GZY_BIN_DIR:-$HOME/.local/bin}"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT

os="$(uname -s)"
arch="$(uname -m)"
case "$os" in
  Darwin) os_name="Darwin" ;;
  Linux) os_name="Linux" ;;
  *) echo "gzy: unsupported OS: $os" >&2; exit 1 ;;
esac
case "$arch" in
  x86_64|amd64) arch_name="x86_64" ;;
  arm64|aarch64) arch_name="arm64" ;;
  *) echo "gzy: unsupported architecture: $arch" >&2; exit 1 ;;
esac

artifact="gzy_${os_name}_${arch_name}.tar.gz"
if [ "$version" = "latest" ]; then
  url="https://github.com/${repo}/releases/latest/download/${artifact}"
else
  url="https://github.com/${repo}/releases/download/${version}/${artifact}"
fi

mkdir -p "$bin_dir"
if command -v curl >/dev/null 2>&1; then
  curl -fsSL "$url" -o "$tmp_dir/$artifact"
elif command -v wget >/dev/null 2>&1; then
  wget -qO "$tmp_dir/$artifact" "$url"
else
  echo "gzy: install requires curl or wget" >&2
  exit 1
fi

tar -xzf "$tmp_dir/$artifact" -C "$tmp_dir"
install "$tmp_dir/gzy" "$bin_dir/gzy"
echo "gzy installed to $bin_dir/gzy"
case ":$PATH:" in
  *":$bin_dir:"*) ;;
  *) echo "Add this to your shell profile: export PATH=\"$bin_dir:\$PATH\"" ;;
esac
