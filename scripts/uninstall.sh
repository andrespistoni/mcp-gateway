#!/usr/bin/env sh
set -eu

purge=false
install_dir="${MCP_GATEWAY_INSTALL_DIR:-}"

while [ "$#" -gt 0 ]; do
  case "$1" in
    --purge) purge=true ;;
    --install-dir)
      [ "$#" -ge 2 ] || { printf 'Error: --install-dir requiere una ruta.\n' >&2; exit 2; }
      install_dir=$2
      shift
      ;;
    *) printf 'Uso: %s [--purge] [--install-dir RUTA]\n' "$0" >&2; exit 2 ;;
  esac
  shift
done

binary=""
if [ -n "$install_dir" ]; then
  binary="$install_dir/mcp-gateway"
elif command -v mcp-gateway >/dev/null 2>&1; then
  binary="$(command -v mcp-gateway)"
elif [ -x "${HOME:?}/.local/bin/mcp-gateway" ]; then
  binary="$HOME/.local/bin/mcp-gateway"
elif [ -x /usr/local/bin/mcp-gateway ]; then
  binary=/usr/local/bin/mcp-gateway
fi

disable_without_binary() {
  case "$(uname -s)" in
    Linux)
      unit_base="${XDG_CONFIG_HOME:-${HOME:?}/.config}"
      unit="$unit_base/systemd/user/mcp-gateway.service"
      if [ -e "$unit" ]; then
        systemctl --user disable --now mcp-gateway
        rm -f "$unit"
        systemctl --user daemon-reload
      fi
      ;;
    Darwin)
      plist="${HOME:?}/Library/LaunchAgents/mcp-gateway.plist"
      if [ -e "$plist" ]; then
        launchctl bootout "gui/$(id -u)" "$plist" >/dev/null 2>&1 || true
        rm -f "$plist"
      fi
      ;;
    *) printf 'Error: sistema operativo no soportado.\n' >&2; exit 1 ;;
  esac
}

if [ -n "$binary" ] && [ -x "$binary" ]; then
  "$binary" disable-daemon
else
  disable_without_binary
fi

if [ -n "$binary" ]; then
  case "$binary" in
    */mcp-gateway) rm -f "$binary" ;;
    *) printf 'Error: ruta de binario inesperada: %s\n' "$binary" >&2; exit 1 ;;
  esac
fi

if "$purge"; then
  config_dir="${HOME:?}/.mcp-gateway"
  rm -f "$config_dir/mcp-downstreams.yaml" "$config_dir/mcp-downstreams.yaml.lock"
  rmdir "$config_dir" 2>/dev/null || true
  printf 'Configuración propia eliminada.\n'
else
  printf 'Configuración conservada en %s/.mcp-gateway.\n' "$HOME"
fi

printf 'mcp-gateway desinstalado.\n'
printf 'Los registros .mcp.json de proyectos y Claude se conservan; consulte el README para retirarlos.\n'
