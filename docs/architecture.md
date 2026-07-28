# Arquitectura

## Componentes

```text
Agente MCP
    │ HTTP/SSE localhost
    ▼
Servidor SSE ── sesiones y projectDir
    │
    ▼
Proxy ── catálogo, prefijos, routing y backpressure
    │
    ├── proceso MCP stdio A
    ├── proceso MCP stdio B
    └── proceso MCP stdio N
```

## Configuración y startup

1. La aplicación carga `mcp-downstreams.yaml`.
2. El servidor reserva el listener loopback.
3. El proxy inicia downstreams habilitados.
4. Cada downstream completa `initialize` y `tools/list`.
5. El proxy crea un catálogo inmutable.
6. El servidor comienza a aceptar sesiones.

Reservar el listener antes del startup evita iniciar procesos cuando el puerto
no puede utilizarse.

## Catálogo

Cada herramienta recibe el prefijo de su downstream:

```text
filesystem + read_file → filesystem__read_file
```

El catálogo mantiene la relación entre nombre público, downstream y nombre
original.

## Flujo de tools/call

1. SSE valida la solicitud JSON-RPC.
2. El proxy busca la ruta por nombre público.
3. Restaura el nombre original.
4. Inyecta `projectDir` cuando corresponde.
5. Reserva capacidad sin bloquear el servidor HTTP.
6. Asigna un ID interno y envía la solicitud por stdio.
7. Correlaciona la respuesta y restaura el ID del cliente.
8. Entrega el resultado mediante la sesión SSE.

## Cancelación y backpressure

Cada downstream procesa una llamada activa y mantiene una cola acotada. Una
solicitud que no puede reservar capacidad se rechaza antes de ser admitida.

Las cancelaciones y timeouts liberan créditos de cola. Si una escritura parcial
puede haber alterado el stream stdio, el proceso se termina para preservar la
integridad del protocolo.

## Shutdown

El cierre coordinado pone el servidor en quiesce, cierra sesiones y detiene
HTTP y downstreams en paralelo. Un deadline limita el tiempo total y activa el
cierre forzado cuando es necesario.
