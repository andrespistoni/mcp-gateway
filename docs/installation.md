# Instalación

## Plataformas y artefactos

Las releases publican binarios de 64 bits:

| Sistema | Arquitecturas | Formato |
| --- | --- | --- |
| Linux | `amd64`, `arm64` | `.tar.gz` |
| macOS | `amd64`, `arm64` | `.tar.gz` |
| Windows | `amd64`, `arm64` | `.zip` |

Todos los artefactos se acompañan de un archivo de checksums SHA-256.

## Linux y macOS

```bash
curl -fsSL https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.sh | sh
```

El instalador:

1. detecta sistema y arquitectura;
2. localiza la última GitHub Release;
3. descarga el archivo y su manifiesto;
4. verifica el checksum SHA-256;
5. instala `mcp-gateway` en `/usr/local/bin` cuando es escribible o en
   `~/.local/bin`.

Para fijar versión y ubicación:

```bash
curl -fsSL https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.sh -o install.sh
MCP_GATEWAY_VERSION=v0.1.1 \
MCP_GATEWAY_INSTALL_DIR="$HOME/bin" \
sh install.sh
```

## Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.ps1 | iex
```

El ejecutable se instala en
`%LOCALAPPDATA%\Programs\mcp-gateway` y esa carpeta se añade al `PATH` del
usuario.

Para fijar una versión:

```powershell
irm https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.ps1 -OutFile install.ps1
$env:MCP_GATEWAY_VERSION = "v0.1.1"
.\install.ps1
```

## Instalación manual

1. Descargue el artefacto correspondiente desde
   [GitHub Releases](https://github.com/andrespistoni/mcp-gateway/releases).
2. Descargue `mcp-gateway_<versión>_checksums.txt`.
3. Verifique SHA-256.
4. Extraiga el ejecutable en una carpeta incluida en `PATH`.
5. Compruebe la versión:

```bash
mcp-gateway version
```

## Actualización

Ejecute nuevamente el instalador. El binario se reemplaza y la configuración
existente se conserva. Después compruebe:

```bash
mcp-gateway version
mcp-gateway doctor
```

## Desinstalación

Linux y macOS:

```bash
curl -fsSL https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/uninstall.sh | sh
```

Windows:

```powershell
irm https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/uninstall.ps1 | iex
```

El desinstalador retira primero el servicio y después elimina el binario.
Windows también elimina del `PATH` la carpeta administrada.

La configuración se conserva por defecto. Para eliminar los archivos de
configuración propios use `--purge` en Unix o `-Purge` en PowerShell.
