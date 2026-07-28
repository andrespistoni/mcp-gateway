# mcp-gateway

[![Calidad](https://github.com/andrespistoni/mcp-gateway/actions/workflows/quality.yml/badge.svg)](https://github.com/andrespistoni/mcp-gateway/actions/workflows/quality.yml)
[![Release](https://img.shields.io/github/v/release/andrespistoni/mcp-gateway)](https://github.com/andrespistoni/mcp-gateway/releases/latest)
[![Licencia MIT](https://img.shields.io/badge/licencia-MIT-blue.svg)](LICENSE)

`mcp-gateway` reúne varios servidores MCP locales basados en stdio detrás de un
único endpoint HTTP/SSE. Expone un catálogo unificado en
`http://localhost:3333/sse`, añade prefijos para evitar colisiones y enruta cada
`tools/call` al proceso correspondiente.

El gateway está diseñado para ejecutarse como servicio de usuario en Linux,
macOS y Windows. Solo escucha en loopback y no expone el servicio a la red.

## Características

- agrega herramientas de varios servidores MCP stdio;
- descubre configuraciones conocidas sin sobrescribir entradas existentes;
- registra el endpoint en proyectos y en Claude Code;
- puede inyectar el directorio del proyecto en una llamada;
- gestiona un servicio de usuario con systemd, launchd o Task Scheduler;
- aplica límites de tamaño, sesiones, cola y tiempo de ejecución;
- valida Host y Origin y redacta secretos estructurados en diagnósticos;
- incluye pruebas unitarias, E2E, de carrera y fuzzing.

## Documentación

La documentación completa está disponible en [docs/index.md](docs/index.md):
instalación, inicio rápido, configuración, proyectos, operación, seguridad,
arquitectura, desarrollo y releases.

## Plataformas soportadas

| Sistema | Arquitecturas | Servicio de usuario |
| --- | --- | --- |
| Linux | `amd64`, `arm64` | systemd |
| macOS | Intel (`amd64`), Apple Silicon (`arm64`) | launchd |
| Windows | `amd64`, `arm64` | Task Scheduler |

Se requiere un sistema de 64 bits. Para compilar desde el código fuente se
requiere Go 1.25 o posterior.

## Instalación rápida

Los instaladores descargan la última GitHub Release, verifican su checksum
SHA-256 y copian el binario a una ubicación del usuario. Es recomendable
inspeccionar cualquier script remoto antes de ejecutarlo.

### Linux y macOS

```bash
curl -fsSL https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.sh | sh
```

El binario se instala en `/usr/local/bin` cuando es escribible; en caso
contrario usa `~/.local/bin`. Para elegir una ubicación o versión:

```bash
curl -fsSL https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.sh -o install.sh
MCP_GATEWAY_INSTALL_DIR="$HOME/bin" MCP_GATEWAY_VERSION="v0.1.1" sh install.sh
```

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.ps1 | iex
```

Instala el ejecutable en
`%LOCALAPPDATA%\Programs\mcp-gateway` y añade esa carpeta al `PATH` del usuario.
Para revisar el script o fijar una versión antes de ejecutarlo:

```powershell
irm https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/install.ps1 -OutFile install.ps1
$env:MCP_GATEWAY_VERSION = "v0.1.1"
.\install.ps1
```

### Instalación manual

1. Descarga el archivo de tu sistema y
   `mcp-gateway_<versión>_checksums.txt` desde
   [GitHub Releases](https://github.com/andrespistoni/mcp-gateway/releases/latest).
2. Verifica el checksum SHA-256.
3. Extrae `mcp-gateway` o `mcp-gateway.exe` en una carpeta incluida en `PATH`.
4. Comprueba la instalación:

```bash
mcp-gateway version
```

## Primer uso

Después de instalar el binario, siga este orden.

### 1. Inicializar el gateway

```bash
mcp-gateway setup
```

`setup` crea `~/.mcp-gateway/mcp-downstreams.yaml`, incorpora los servidores
que puede descubrir sin sobrescribir entradas existentes y habilita el servicio
de usuario en el puerto `3333`.

Para elegir otro puerto, hágalo en este primer paso y use el mismo valor al
registrar proyectos:

```bash
mcp-gateway setup --port 4444
```

### 2. Revisar y configurar los MCPs

```bash
mcp-gateway list
mcp-gateway doctor
```

Si necesita volver a examinar el equipo después de instalar otro servidor MCP:

```bash
mcp-gateway discover
mcp-gateway discover --write
```

Los MCPs que no puedan descubrirse se añaden con `mcp-gateway add`, como se
explica en [Configurar servidores MCP](#configurar-servidores-mcp).

### 3. Registrar cada proyecto

Ejecute este comando dentro de cada proyecto que vaya a usar con el gateway:

```bash
cd /ruta/al/proyecto
mcp-gateway register-project
```

También puede indicar la ruta explícitamente:

```bash
mcp-gateway register-project --project-dir /ruta/al/proyecto
```

En PowerShell:

```powershell
mcp-gateway register-project --project-dir (Get-Location).Path
```

El comando crea o actualiza `<proyecto>/.mcp.json`, modifica únicamente
`mcpServers.mcp-gateway` y añade `.mcp.json` una sola vez a `.gitignore`. La URL
registrada incluye `projectDir`, por lo que cada conexión queda asociada al
proyecto correcto.

Registrar un proyecto no configura automáticamente los demás: repita este paso
en cada repositorio que utilice.

<!-- flujo-e2e-verificado: inicio -->
El equivalente con puerto explícito, comprobado por la suite E2E, es:

```bash
export PUERTO=4444
export PROJECT_DIR="$PWD"

mcp-gateway setup --port "$PUERTO"
mcp-gateway discover --write
mcp-gateway register-project --project-dir "$PROJECT_DIR" --port "$PUERTO"
```
<!-- flujo-e2e-verificado: fin -->

### 4. Integrar el cliente

Para Claude Code, el `.mcp.json` creado en el paso anterior ya es la integración
recomendada cuando el MCP necesita conocer el proyecto. Cierre y vuelva a abrir
el agente o la sesión para que relea el archivo.

Opcionalmente, puede registrar también un endpoint de usuario disponible fuera
de proyectos registrados:

```bash
mcp-gateway install-claude
```

Ese registro global no incluye `projectDir`. No sustituye a
`register-project` para MCPs que necesitan contexto del repositorio y puede
omitirse en un flujo exclusivamente basado en proyectos.

### 5. Verificar

Desde el proyecto registrado:

```bash
mcp-gateway doctor --verbose
mcp-gateway list
```

Si el cliente muestra las herramientas con sus prefijos, la instalación quedó
operativa.

## Configurar servidores MCP

Examina los servidores que pueden descubrirse:

```bash
mcp-gateway discover
mcp-gateway discover --write
mcp-gateway list
```

También puede añadirse cualquier servidor MCP stdio sin recompilar:

```bash
mcp-gateway add <nombre> \
  --prefix <nombre>__ \
  --binary /ruta/absoluta/al/servidor \
  --arg "--root" \
  --arg "/ruta/con espacios"
```

Cada `--arg` representa un elemento independiente de `argv`; no es una cadena
de shell. Las variables de entorno se añaden con `--env KEY=VALUE`. No conviene
poner secretos en los argumentos porque pueden ser visibles para otros procesos.

Para pasar el directorio del proyecto cuando el cliente no haya enviado ya ese
argumento:

```bash
mcp-gateway add ejemplo \
  --prefix ejemplo__ \
  --binary /ruta/al/servidor \
  --inject-project projectDir
```

Por defecto, `add` valida `initialize` y todas las páginas de `tools/list` antes
de guardar. `--skip-validation` omite esa comprobación y debe reservarse para
casos excepcionales.

Administración:

```bash
mcp-gateway disable filesystem
mcp-gateway enable filesystem
mcp-gateway remove filesystem
mcp-gateway restart
```

Una mutación reinicia automáticamente el daemon cuando está instalado y en
ejecución.

## Operación del servicio

```bash
mcp-gateway doctor --verbose
mcp-gateway restart
mcp-gateway disable-daemon
mcp-gateway enable-daemon
```

También puede ejecutarse en primer plano:

```bash
mcp-gateway serve --port 3333
```

El endpoint MCP es `http://localhost:<puerto>/sse`. `GET /sse` crea la sesión y
`POST /message` recibe JSON-RPC conforme a MCP `2024-11-05`.

## Límites y seguridad

- solo acepta Host `localhost:<puerto>` y Origin ausente o
  `http://localhost:<puerto>`;
- no implementa autenticación ni TLS porque el listener es exclusivamente
  loopback;
- cada sesión admite hasta 16 solicitudes pendientes;
- cada downstream admite una llamada activa y hasta 32 en cola;
- los mensajes HTTP y stdio tienen un máximo de 1 MiB;
- una llamada tiene un timeout de 60 segundos;
- los downstreams heredan los permisos del usuario y no se ejecutan en sandbox;
- `projectDir` aporta contexto, pero no autorización ni confinamiento;
- los resultados MCP se entregan sin inspección ni redacción.

No exponga el puerto mediante proxies, túneles o reglas de red sin añadir una
capa de autenticación adecuada.

## Actualización y desinstalación

Ejecutar de nuevo el instalador sustituye el binario por la última versión. La
configuración se conserva.

### Linux y macOS

```bash
curl -fsSL https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/uninstall.sh | sh
```

Para eliminar también `mcp-downstreams.yaml`:

```bash
curl -fsSL https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/uninstall.sh -o uninstall.sh
sh uninstall.sh --purge
```

Si se instaló en una ruta personalizada, use
`--install-dir /ruta` o `MCP_GATEWAY_INSTALL_DIR`.

### Windows PowerShell

```powershell
irm https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/uninstall.ps1 | iex
```

Para eliminar también la configuración:

```powershell
irm https://raw.githubusercontent.com/andrespistoni/mcp-gateway/main/scripts/uninstall.ps1 -OutFile uninstall.ps1
.\uninstall.ps1 -Purge
```

Los desinstaladores detienen y eliminan el servicio de usuario antes de borrar
el ejecutable. Windows también retira del `PATH` la carpeta añadida por el
instalador. Por seguridad, la configuración se conserva salvo que se solicite
`--purge`/`-Purge`.

Los registros de proyecto no se eliminan automáticamente porque el instalador
no mantiene una lista de proyectos y no debe modificar repositorios
arbitrariamente. En cada proyecto registrado, retire
`mcpServers.mcp-gateway` de `.mcp.json`; puede conservar `.mcp.json` en
`.gitignore` si contiene otras configuraciones.

Si se ejecutó `install-claude`, retire el registro de usuario con:

```bash
claude mcp remove mcp-gateway
```

## Compilar desde el código fuente

```bash
git clone https://github.com/andrespistoni/mcp-gateway.git
cd mcp-gateway
go test ./... -count=1
go build -o mcp-gateway ./cmd/mcp-gateway
```

Verificación completa para contribuir:

```bash
test -z "$(gofmt -l .)"
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
bash scripts/build-release.sh --dry-run
```

## Publicación de releases

El workflow de calidad se ejecuta en cada pull request y push a `main`. Una
release se publica automáticamente al subir un tag semántico:

```bash
git tag -a v1.0.0 -m "v1.0.0"
git push origin v1.0.0
```

El workflow vuelve a ejecutar calidad, compila Linux, macOS y Windows para
`amd64` y `arm64`, genera checksums y crea la GitHub Release con notas
automáticas. Crear un commit o hacer push a `main` no crea una release por sí
solo; el tag es la autorización explícita de publicación.

Los cambios destinados a la siguiente versión se mantienen en
[CHANGELOG.md](CHANGELOG.md). Antes de crear un tag, la sección `Unreleased`
debe convertirse en la versión y fecha que se publicarán.

## Solución de problemas

- Ejecute `mcp-gateway doctor --verbose` para validar configuración,
  downstreams, daemon, endpoint y Claude.
- Si el comando no se encuentra tras instalar, abra una terminal nueva o añada
  la carpeta indicada por el instalador a `PATH`.
- Si el puerto está ocupado, ejecute `mcp-gateway setup --port <otro-puerto>`.
- Si un downstream falla, ejecútelo directamente y revise que hable MCP por
  stdio y no escriba logs en stdout.

## Licencia

Distribuido bajo la [licencia MIT](LICENSE).
