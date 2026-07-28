# Seguridad

## Modelo de red

El listener usa exclusivamente loopback y las URL generadas contienen el
literal `localhost`. El gateway valida:

- Host `localhost:<puerto>`;
- Origin ausente o `http://localhost:<puerto>`.

No exponga el puerto mediante túneles, proxies o reglas de red sin incorporar
autenticación y controles acordes al nuevo límite de confianza.

## Procesos downstream

Los servidores MCP:

- se ejecutan directamente con `argv` separado;
- no pasan por una cadena de shell;
- heredan los permisos del usuario;
- no se ejecutan dentro de un sandbox.

Configure únicamente binarios y argumentos confiables.

## Datos sensibles

El gateway redacta claves estructuradas asociadas a tokens, secretos,
contraseñas, claves y autorización antes de escribir diagnósticos.

No registra:

- IDs de sesión;
- argumentos o resultados MCP;
- variables de entorno;
- valores normales de `projectDir`.

Los resultados MCP se entregan al cliente sin inspección ni redacción porque
forman parte de la respuesta funcional.

## projectDir

`projectDir` es contexto para una herramienta. No concede autorización, no
limita el filesystem y no sustituye un sandbox.

## Archivos

- La configuración se escribe de forma atómica.
- Los archivos de configuración y definiciones de servicio se restringen al
  usuario.
- Los registros de proyecto conservan claves ajenas al gateway.

## Recomendaciones

- Mantenga el sistema y los servidores MCP actualizados.
- Verifique los checksums de release.
- Use referencias de entorno para credenciales.
- Evite secretos en argumentos de proceso.
- Revise los MCPs antes de habilitarlos.
- Ejecute `doctor` después de cambios operativos.
