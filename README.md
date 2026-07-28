# mcp-gateway

`mcp-gateway` agrupa servidores MCP locales de stdio detrás de una sola conexión HTTP/SSE en `http://localhost:<puerto>/sse`. El binario sirve exclusivamente en loopback y usa el literal `localhost` en sus URL generadas.

> **Estado de verificación:** Linux, macOS y Windows siguen en alcance funcional. Este flujo ejecutó la suite solo en Linux bajo WSL/Ubuntu y no ejecutó pruebas nativas de Windows ni macOS. Los cross-builds de esas plataformas no equivalen a pruebas nativas. La evidencia ausente —incluidos Job Object Windows, Claude real, recetas/argv aprobados, disponibilidad de 3333 y política de prompt— es seguimiento no bloqueante de este flujo y no autoriza una publicación por sí sola; véase [`docs/release-checklist.md`](docs/release-checklist.md).

## Instalación de una candidata verificada

1. Descargue el archivo de su plataforma y `mcp-gateway_<versión>_checksums.txt` desde una candidata autorizada.
2. Verifique el SHA-256 antes de descomprimir. En Linux:

   ```bash
   sha256sum --ignore-missing --check mcp-gateway_<versión>_checksums.txt
   tar -xzf mcp-gateway_<versión>_linux_amd64.tar.gz
   install -m 0755 mcp-gateway "$HOME/.local/bin/mcp-gateway"
   ```

3. Compruebe la identidad del binario:

   ```bash
   mcp-gateway version
   ```

Los artefactos se generan con `scripts/build-release.sh`; el script revisa el contenido del tar/zip, `go version -m` y checksums. Su salida no autoriza publicación por sí sola.

## Flujo operativo

<!-- flujo-e2e-verificado: inicio -->
El siguiente flujo está cubierto por `TestREADMEWorkflow` en un HOME, PATH, proyecto, daemon y downstreams **falsos** aislados. La prueba usa un puerto dinámico; no toca 3333, no ejecuta Claude ni discovery real.

```bash
export PUERTO=4444                 # Elija un puerto aprobado; 3333 es el default contractual.
export PROJECT_DIR="$PWD"

mcp-gateway setup --port "$PUERTO"
mcp-gateway discover --write
mcp-gateway register-project --project-dir "$PROJECT_DIR" --port "$PUERTO"
```

`setup` crea o actualiza `~/.mcp-gateway/mcp-downstreams.yaml`, fusiona discovery sin sobrescribir entradas existentes y configura el servicio de usuario nativo. En producción requiere que el gestor y sus permisos estén aprobados. `register-project` actualiza exclusivamente `mcpServers.mcp-gateway` en `<proyecto>/.mcp.json` y añade `.mcp.json` una vez a `<proyecto>/.gitignore`.

Agregue un MCP que no pertenezca a las recetas conocidas sin recompilar. Cada `--arg` es un elemento argv separado: no use una cadena de shell ni incluya secretos en argumentos.

```bash
mcp-gateway add <nombre> --prefix <nombre>__ --binary /ruta/absoluta/al/servidor \
  --arg "--opcion" --arg "valor con espacios" --inject-project projectDir
mcp-gateway list
mcp-gateway doctor --verbose
```

Por defecto `add` valida `initialize` y todas las páginas `tools/list` antes de guardar. `--skip-validation` solo debe usarse de forma excepcional: conserva `enabled: true`, avisa que el servidor puede no estar disponible y no sustituye una validación posterior.

El daemon ejecuta el equivalente a `mcp-gateway serve --port <puerto>`. Para operar manualmente:

```bash
mcp-gateway serve --port "$PUERTO"
mcp-gateway restart
mcp-gateway disable-daemon
```

Para registrar Claude Code, primero valide la versión y sintaxis aprobadas en la evidencia de release y después ejecute `mcp-gateway install-claude --port "$PUERTO"`. No se afirma compatibilidad con una versión de Claude no documentada por esa evidencia.
<!-- flujo-e2e-verificado: fin -->

## Operación y límites

- El puerto configurado debe estar entre 1024 y 65535; la precedencia es `--port`, configuración y `3333`.
- `GET /sse` abre una sesión y anuncia un endpoint relativo de mensajes. `POST /message` acepta JSON-RPC MCP `2024-11-05`.
- Solo se aceptan Host `localhost:<puerto>` y Origin ausente o `http://localhost:<puerto>`; no hay CORS permisivo ni bind remoto.
- Cada sesión admite hasta 16 solicitudes pendientes; cada downstream admite una llamada activa más 32 en cola. Los mensajes HTTP/stdin tienen un máximo de 1 MiB.
- `tools/list` se agrega con prefijos; `tools/call` restaura el nombre downstream exacto y puede inyectar `projectDir` sin sobrescribir argumentos del caller.

## Seguridad y riesgos residuales

El gateway ejecuta binarios directamente con argv separado, limita protocolo/recursos, conserva stdout downstream para protocolo y redacta campos estructurados sensibles antes de sus sinks. No registra IDs de sesión, argumentos/resultados MCP, valores de entorno ni `projectDir` normal.

Los riesgos aceptados de v1 siguen vigentes:

1. no hay autenticación ni TLS: loopback, Host/Origin e IDs aleatorios reducen pero no eliminan ataques de procesos locales;
2. los downstreams no están sandboxed y heredan permisos de la persona usuaria;
3. una configuración explícita puede ejecutar binarios arbitrarios;
4. recetas sin firma y PATH/fallback por basename pueden seleccionar un binario no deseado en un entorno comprometido;
5. `projectDir` es contexto, no autorización ni confinamiento;
6. resultados de tools pueden contener datos sensibles y se preservan para el cliente, no se inspeccionan ni redactan;
7. existe riesgo TOCTOU entre validar y usar rutas, binarios o symlinks.

La redacción no descubre secretos arbitrarios escritos como texto libre. Revise con cuidado la configuración, los binarios y cualquier salida verbose.

## Backout

Para volver a una candidata v1 anterior verificada: deshabilite el daemon, restaure el binario cuyo checksum fue validado, ejecute `enable-daemon --port <puerto>` con ese binario y ejecute `doctor`. La configuración v1 no tiene migración ni estado de sesión persistente que revertir.

## Desarrollo y release

```bash
gofmt -w cmd internal test/e2e
go test ./... -count=1
go test -race ./... -count=1
go vet ./...
bash scripts/build-release.sh --dry-run
```

Para construir una candidata inspeccionable sin publicar:

```bash
bash scripts/build-release.sh --version 0.0.0-rc.1 --output /tmp/mcp-gateway-release
sha256sum --check /tmp/mcp-gateway-release/mcp-gateway_0.0.0-rc.1_checksums.txt
```

`scripts/check-release-gates.sh` advierte sobre evidencia pendiente, sin bloquear este flujo ni declarar una publicación autorizada. No marque una casilla por un cross-build o un fake; una publicación real requiere evidencia archivada y autorización humana explícita.
