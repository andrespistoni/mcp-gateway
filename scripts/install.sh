#!/usr/bin/env sh
set -eu

repository="${MCP_GATEWAY_REPOSITORY:-andrespistoni/mcp-gateway}"
version="${MCP_GATEWAY_VERSION:-latest}"
install_dir="${MCP_GATEWAY_INSTALL_DIR:-}"

require() {
  command -v "$1" >/dev/null 2>&1 || {
    printf 'Error: se requiere %s.\n' "$1" >&2
    exit 1
  }
}

require curl
require tar

case "$(uname -s)" in
  Linux) os=linux ;;
  Darwin) os=darwin ;;
  *) printf 'Error: sistema operativo no soportado.\n' >&2; exit 1 ;;
esac

case "$(uname -m)" in
  x86_64|amd64) arch=amd64 ;;
  arm64|aarch64) arch=arm64 ;;
  *) printf 'Error: arquitectura no soportada.\n' >&2; exit 1 ;;
esac

if [ "$version" = latest ]; then
  latest_url="$(curl -fsSL -o /dev/null -w '%{url_effective}' "https://github.com/$repository/releases/latest")"
  version="${latest_url##*/}"
fi

case "$version" in
  v*) release="${version#v}" ;;
  *) release="$version"; version="v$version" ;;
esac

archive="mcp-gateway_${release}_${os}_${arch}.tar.gz"
checksums="mcp-gateway_${release}_checksums.txt"
base_url="https://github.com/$repository/releases/download/$version"
tmp_dir="$(mktemp -d)"
trap 'rm -rf "$tmp_dir"' EXIT HUP INT TERM

curl -fsSL "$base_url/$archive" -o "$tmp_dir/$archive"
curl -fsSL "$base_url/$checksums" -o "$tmp_dir/$checksums"

if command -v sha256sum >/dev/null 2>&1; then
  expected="$(awk -v file="$archive" '$2 == file { print $1 }' "$tmp_dir/$checksums")"
  actual="$(sha256sum "$tmp_dir/$archive" | awk '{ print $1 }')"
elif command -v shasum >/dev/null 2>&1; then
  expected="$(awk -v file="$archive" '$2 == file { print $1 }' "$tmp_dir/$checksums")"
  actual="$(shasum -a 256 "$tmp_dir/$archive" | awk '{ print $1 }')"
else
  printf 'Error: se requiere sha256sum o shasum para verificar la descarga.\n' >&2
  exit 1
fi

[ -n "$expected" ] && [ "$expected" = "$actual" ] || {
  printf 'Error: checksum SHA-256 inválido.\n' >&2
  exit 1
}

tar -xzf "$tmp_dir/$archive" -C "$tmp_dir"

if [ -z "$install_dir" ]; then
  if [ -d /usr/local/bin ] && [ -w /usr/local/bin ]; then
    install_dir=/usr/local/bin
  else
    install_dir="${HOME:?}/.local/bin"
  fi
fi

mkdir -p "$install_dir"
install -m 0755 "$tmp_dir/mcp-gateway" "$install_dir/mcp-gateway"

printf 'mcp-gateway %s instalado en %s/mcp-gateway\n' "$release" "$install_dir"
case ":${PATH:-}:" in
  *":$install_dir:"*) ;;
  *) printf 'Añada %s a PATH antes de continuar.\n' "$install_dir" ;;
esac
printf 'Siguiente paso: mcp-gateway setup\n'
