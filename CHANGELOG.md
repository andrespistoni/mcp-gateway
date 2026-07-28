# Changelog

Todos los cambios relevantes de `mcp-gateway` se documentan en este archivo.

El formato está basado en [Keep a Changelog](https://keepachangelog.com/es-ES/1.1.0/)
y el proyecto utiliza [Versionado Semántico](https://semver.org/lang/es/).

## [Unreleased]

## [0.1.1] - 2026-07-28

### Fixed

- El runner del daemon drena stdout y stderr antes de recolectar el proceso para
  no perder salidas cortas bajo concurrencia.

## [0.1.0] - 2026-07-28

### Added

- Transporte HTTP/SSE sobre loopback para exponer un catálogo unificado de
  herramientas MCP stdio.
- Enrutamiento de `tools/call`, prefijos por downstream e inyección opcional de
  `projectDir`.
- Comandos `setup`, `serve`, `doctor`, `enable-daemon`, `disable-daemon` y
  `restart`.
- Servicios de usuario mediante systemd, launchd y Task Scheduler.
- Registro por proyecto mediante `.mcp.json` e integración opcional con Claude
  Code.
- Instaladores y desinstaladores con verificación SHA-256 para Linux, macOS y
  Windows.
- Builds de release para `amd64` y `arm64`, checksums y publicación automática
  desde tags semánticos.
- Changelog versionado y reglas de repositorio para mantenerlo de forma
  consistente.
- Gate de cobertura de producción superior al 80% y tests de componente para
  protocolo MCP, daemon, SSE, CLI, persistencia y diagnósticos.
- Documentación pública de instalación, configuración, operación, seguridad,
  actualización y desinstalación.

### Changed

- Las mutaciones de configuración reinician automáticamente el daemon cuando
  está instalado y en ejecución.
- El workflow de calidad aplica permisos mínimos, concurrencia controlada y
  referencias inmutables para las Actions utilizadas.
- El empaquetado normaliza el directorio de salida para generar correctamente
  archivos ZIP desde sus directorios temporales.
- Los commits del repositorio siguen Conventional Commits con reglas
  compartidas para agentes.

### Security

- Validación estricta de Host y Origin para conexiones locales.
- Límites de sesiones, solicitudes, colas, mensajes y tiempos de ejecución.
- Redacción de secretos estructurados en diagnósticos y archivos de servicio
  restringidos al usuario.

[Unreleased]: https://github.com/andrespistoni/mcp-gateway/compare/v0.1.1...HEAD
[0.1.1]: https://github.com/andrespistoni/mcp-gateway/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/andrespistoni/mcp-gateway/releases/tag/v0.1.0
