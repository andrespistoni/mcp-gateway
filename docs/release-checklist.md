# Checklist de publicación v1

## Enmienda de alcance de verificación

Linux, macOS y Windows siguen dentro del alcance funcional. Este flujo solo dispone de Linux bajo WSL/Ubuntu: la suite se ejecuta allí y no ejecuta pruebas nativas de Windows ni macOS. Los cross-builds de macOS/Windows siguen siendo útiles para compilación, pero no equivalen a listener, persistencia/ACL, árbol de procesos ni gestor de servicios nativos.

Las casillas siguientes son seguimientos de release y de aprobación humana, no tareas bloqueantes de implementación ni de fases posteriores de este flujo. No se consideran completadas por código, fakes o cross-builds.

- [ ] Archivar matriz exacta Linux/macOS/Windows con versiones, arquitecturas y runners nativos aprobados.
- [ ] Archivar listener, persistencia/ACL, árbol de procesos y gestor de servicio nativos para cada plataforma que se vaya a publicar.
- [ ] Ejecutar y archivar en Windows la prueba de Job Object desde creación con hijo/nieto para cancelación, timeout, crash y shutdown (seguimiento retirado de T-013).
- [ ] Aprobar y archivar versiones y argv reales de CodeGraph, codebase-memory-mcp y Engram.
- [ ] Aprobar y comprobar de forma separada la disponibilidad real del puerto 3333.
- [ ] Aprobar la política corporativa del prompt adicional para ejecutar binarios.
- [ ] Archivar versión, sintaxis `get/add` y prueba de Claude real con dos downstreams por SSE localhost.
- [ ] Conseguir reproducción independiente de checksums y contenido exclusivo del binario de producto.
- [ ] Obtener revisión humana de la evidencia redactada y una autorización explícita de publicación.

`scripts/check-release-gates.sh` verifica la presencia de estos documentos y advierte cuando conserva marcadores `Estado: PENDIENTE` o casillas abiertas. Su advertencia no bloquea este flujo ni afirma que una publicación esté autorizada.
