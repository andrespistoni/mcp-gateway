# Operación

## Estado y diagnóstico

```bash
mcp-gateway list
mcp-gateway doctor
mcp-gateway doctor --verbose
```

`doctor` comprueba configuración, ejecutables, handshake MCP, daemon, endpoint
local y disponibilidad de Claude.

## Ciclo de vida del daemon

```bash
mcp-gateway restart
mcp-gateway disable-daemon
mcp-gateway enable-daemon
```

- `restart` reinicia el servicio existente.
- `disable-daemon` lo detiene y elimina su definición de usuario.
- `enable-daemon` vuelve a crear la definición y lo inicia.

Gestores utilizados:

| Sistema | Gestor |
| --- | --- |
| Linux | systemd de usuario |
| macOS | launchd LaunchAgent |
| Windows | Task Scheduler |

## Ejecución en primer plano

```bash
mcp-gateway serve --port 3333
```

El proceso atiende `GET /sse` y `POST /message` exclusivamente en
`localhost:<puerto>`.

## Apagado

Ante una señal de terminación, el gateway:

1. deja de admitir sesiones nuevas;
2. cierra las sesiones activas;
3. solicita shutdown al servidor HTTP y a los downstreams;
4. fuerza el cierre si vence el plazo.

## Límites

- 128 sesiones SSE simultáneas.
- 16 solicitudes pendientes por sesión.
- Una llamada activa y 32 llamadas en cola por downstream.
- Mensajes HTTP y stdio de hasta 1 MiB.
- Timeout de 60 segundos por llamada.

## Resolución de problemas

### El comando no aparece

Abra una terminal nueva y compruebe que la carpeta de instalación esté en
`PATH`.

### El puerto está ocupado

```bash
mcp-gateway setup --port 4444
```

Vuelva a registrar los proyectos con el mismo puerto.

### Un downstream no está disponible

```bash
mcp-gateway doctor --verbose
mcp-gateway list
```

Compruebe que el ejecutable exista, tenga permisos y use stdout exclusivamente
para mensajes MCP.

### El agente no muestra herramientas

Confirme que existe `.mcp.json` en la raíz, vuelva a ejecutar
`register-project` y reinicie la sesión del agente.
