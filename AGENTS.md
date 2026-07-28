# Repository instructions

## Product invariants

No cambies estos contratos sin una petición explícita y una entrada destacada
en el changelog:

- El servidor escucha exclusivamente en loopback y las URL generadas usan el
  literal `localhost`; nunca abras el bind a interfaces remotas por defecto.
- Host y Origin se validan de forma estricta. Loopback no debe presentarse como
  sustituto de autenticación cuando se atraviesa un proxy o túnel.
- Los procesos MCP y gestores nativos se ejecutan con `argv` separado, nunca
  mediante una cadena de shell.
- No registres IDs de sesión, argumentos o resultados MCP, variables de
  entorno, secretos ni valores normales de `projectDir`.
- La inyección de `projectDir` no sobrescribe un argumento enviado por el
  cliente y no debe describirse como sandbox o mecanismo de autorización.
- Las mutaciones de configuración deben conservar escritura atómica, permisos
  privados y contenido existente que el gateway no administra.
- Una desinstalación debe retirar el daemon antes que el binario y conservar
  configuración y registros de proyectos salvo solicitud explícita de purge.
- Linux, macOS y Windows siguen siendo plataformas soportadas. Mantén aislado
  el código específico mediante build tags y evita asumir herramientas Unix en
  rutas ejecutadas por Windows.

## Code navigation

Si existe `.codegraph/`, usa CodeGraph antes de búsquedas textuales o lectura
manual para localizar símbolos, comprender flujos y evaluar el impacto de un
cambio. No crees el índice automáticamente cuando no exista.

## Conventional Commits

Todos los commits deben seguir Conventional Commits:

```text
<type>[optional scope][!]: <descripción>
```

- Usa uno de estos tipos: `feat`, `fix`, `docs`, `test`, `refactor`, `perf`,
  `build`, `ci`, `chore` o `revert`.
- La descripción debe ser breve, en minúscula, en modo imperativo y sin punto
  final.
- Usa un scope solo cuando aporte contexto estable, por ejemplo `sse`,
  `daemon`, `proxy`, `cli`, `docs` o `release`.
- Marca cambios incompatibles con `!` y explica la migración en un footer
  `BREAKING CHANGE:`.
- Mantén cada commit enfocado en una unidad lógica. No mezcles correcciones o
  funcionalidades independientes.
- Para cambios sustanciales, añade un body con apartados `Funcional`,
  `Técnico` y `Verificación`, describiendo resultados y comandos realmente
  ejecutados.
- No incluyas secretos, rutas privadas, datos personales ni output sensible en
  asuntos o bodies.

## Validation

Ejecuta validación proporcional al cambio y comunica con precisión qué se
ejecutó y qué no:

- Todo cambio: `git diff --check` y `test -z "$(gofmt -l .)"`.
- Código Go: `go test ./... -count=1` y `go vet ./...`.
- Concurrencia, procesos, proxy, SSE o shutdown:
  `go test -race ./... -count=1`.
- README o workflows:
  `go test ./test/e2e -count=1`.
- Scripts shell: `bash -n scripts/*.sh`.
- Antes de finalizar código de producción: `bash scripts/check-coverage.sh`;
  la cobertura debe ser estrictamente mayor a 80%.
- Packaging, targets o release: `bash scripts/build-release.sh --dry-run`;
  usa un output temporal para validar artefactos reales cuando cambien nombres,
  checksums o instaladores.

Las pruebas que abren sockets deben usar puertos dinámicos y loopback. Un
cross-build demuestra compilación, no funcionamiento nativo. No declares
verificados Windows, macOS, Claude real, systemd, launchd o Task Scheduler sin
evidencia ejecutada en ese entorno.

## Changelog

Este repositorio mantiene `CHANGELOG.md` siguiendo Keep a Changelog y
Versionado Semántico. Todo agente que realice cambios debe revisar estas reglas
antes de finalizar su trabajo.

- Añade en `## [Unreleased]` cualquier cambio visible para usuarios:
  funcionalidades, correcciones, seguridad, compatibilidad, instalación,
  actualización, configuración, CLI, protocolo, operación o despliegue.
- Usa exclusivamente las categorías `Added`, `Changed`, `Deprecated`,
  `Removed`, `Fixed` y `Security`. Crea una categoría solo cuando tenga
  contenido.
- Escribe cada entrada en español, como una viñeta breve que describa el efecto
  para el usuario. No copies asuntos de commits ni detalles internos sin valor
  operativo.
- No añadas entradas para refactors, formato, tests o documentación interna que
  no cambien el comportamiento o las instrucciones públicas.
- Agrupa cambios relacionados en una sola entrada y evita duplicados.
- No inventes versiones, fechas, compatibilidad ni resultados de pruebas.
- No modifiques secciones de versiones publicadas salvo que el usuario pida
  corregir un error histórico.

Al preparar una release:

1. Mantén una sección `## [Unreleased]` vacía al principio.
2. Mueve sus entradas a `## [X.Y.Z] - YYYY-MM-DD`, sin prefijo `v`.
3. Usa la fecha UTC real de publicación.
4. Aplica SemVer: `major` para incompatibilidades, `minor` para funcionalidad
   compatible y `patch` para correcciones compatibles.
5. Añade o actualiza al final los enlaces de comparación entre tags.
6. Verifica que el tag `vX.Y.Z`, el changelog, los assets y el README describan
   la misma versión y plataformas.

## Public documentation

- Si cambia el flujo de instalación, configuración, registro por proyecto,
  daemon, release o desinstalación, actualiza también `README.md`.
- Conserva los marcadores `flujo-e2e-verificado` del README y mantén ejecutables
  los comandos contenidos entre ellos.
- Distingue claramente entre un push a `main` y una publicación mediante tag:
  solo un tag semántico `vX.Y.Z` autoriza la GitHub Release.
- Mantén en español los textos de CLI y la documentación para usuarios; usa
  nombres e identificadores idiomáticos en inglés dentro del código Go.
- No documentes comandos, plataformas, assets o compatibilidad que el código y
  la automatización actuales no produzcan.

## Release discipline

- No crees ni publiques tags, releases o assets salvo solicitud explícita.
- Antes de un tag, comprueba que `CHANGELOG.md`, README, versión, plataformas,
  nombres de assets e instaladores coincidan.
- Las acciones de GitHub deben tener permisos mínimos y estar fijadas a un SHA
  completo revisado.
- Nunca publiques desde un árbol sucio ni omitas una comprobación fallida. Si
  una validación depende de acceso nativo no disponible, comunica la limitación
  de forma explícita en lugar de inferir éxito.
