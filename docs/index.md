# Documentación de mcp-gateway

`mcp-gateway` agrega servidores MCP locales basados en stdio y los publica
mediante un único endpoint HTTP/SSE limitado a loopback.

## Guías

- [Instalación](installation.md): instalación automática, manual, actualización
  y desinstalación.
- [Inicio rápido](quickstart.md): recorrido ordenado desde un equipo nuevo hasta
  un proyecto operativo.
- [Configuración](configuration.md): formato YAML, discovery y administración
  de downstreams.
- [Proyectos y projectDir](projects.md): registro por repositorio y propagación
  del contexto.
- [Operación](operations.md): daemon, diagnóstico, límites y resolución de
  problemas.
- [Seguridad](security.md): modelo de confianza y recomendaciones operativas.
- [Arquitectura](architecture.md): componentes y flujo de una llamada MCP.
- [Desarrollo y releases](development.md): calidad, cobertura, builds y
  publicación.

## Conceptos principales

- Un **downstream** es un servidor MCP ejecutado como proceso hijo mediante
  stdio.
- El **catálogo** combina las herramientas de todos los downstreams habilitados
  y aplica un prefijo único a cada nombre.
- Una **sesión SSE** representa la conexión de un cliente con el gateway.
- `projectDir` identifica el proyecto asociado a una sesión registrada mediante
  `.mcp.json`.
- El **daemon** mantiene el gateway activo como servicio del usuario.
