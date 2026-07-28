# Configuración

## Ubicación

La configuración principal se guarda en:

```text
~/.mcp-gateway/mcp-downstreams.yaml
```

El archivo usa esquema estricto con `version: 1`.

## Ejemplo

```yaml
version: 1
port: 3333
downstreams:
  - name: filesystem
    prefix: filesystem__
    binary: /ruta/al/filesystem-mcp
    args:
      - --root
      - /ruta/permitida
    enabled: true
    env:
      LOG_LEVEL: info
    inject_project: true
    project_argument: projectDir
```

## Campos

### Documento

- `version`: versión del esquema; debe ser `1`.
- `port`: puerto local entre `1024` y `65535`; el valor predeterminado es
  `3333`.
- `downstreams`: lista de servidores MCP stdio.

### Downstream

- `name`: identificador único.
- `prefix`: prefijo único terminado en `__`.
- `binary`: ruta o nombre de ejecutable.
- `args`: elementos `argv`, cada uno como valor independiente.
- `enabled`: habilita o deshabilita el servidor.
- `env`: variables añadidas al entorno del proceso.
- `inject_project`: activa la inyección de contexto.
- `project_argument`: nombre del argumento que recibe la ruta del proyecto.

## Discovery

```bash
mcp-gateway discover
mcp-gateway discover --write
```

Sin `--write`, discovery solo informa resultados. Con `--write`, añade
servidores nuevos y conserva entradas existentes.

`setup` ejecuta el flujo de discovery durante la inicialización.

## Añadir un downstream

```bash
mcp-gateway add nombre \
  --prefix nombre__ \
  --binary /ruta/al/binario \
  --arg "--opcion" \
  --arg "valor con espacios" \
  --env KEY=VALUE
```

El gateway valida `initialize` y todas las páginas de `tools/list` antes de
guardar. `--skip-validation` permite guardar sin esa validación.

Opciones adicionales:

- `--disabled`: guarda el downstream deshabilitado.
- `--inject-project <argumento>`: configura inyección de `projectDir`.

## Administrar downstreams

```bash
mcp-gateway list
mcp-gateway disable nombre
mcp-gateway enable nombre
mcp-gateway remove nombre
```

Las mutaciones reinician el daemon cuando está instalado y activo.

## Variables y secretos

Use referencias de entorno cuando el proceso necesite credenciales:

```yaml
env:
  API_TOKEN: ${API_TOKEN}
```

Evite secretos en `args`, nombres, rutas o logs.
