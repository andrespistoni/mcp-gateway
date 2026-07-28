# Matriz técnica de release

## Estado

Linux, macOS y Windows siguen siendo plataformas funcionales en alcance. Esta es una matriz **candidata de compilación**, no una aprobación de plataformas publicables. En este flujo solo se ejecutó la suite en Linux bajo WSL/Ubuntu; no se ejecutaron pruebas nativas de macOS ni Windows. Los cross-builds detectan errores de compilación, pero no aportan evidencia nativa de listener, persistencia/ACL, proceso descendiente ni gestor de servicios.

| SO | Arquitectura candidata | Build sin CGO | Prueba nativa requerida | Estado de evidencia real |
|---|---|---:|---|---|
| Linux | amd64 | Sí | listener loopback, persistencia, árbol Unix y systemd de usuario | Suite Go ejecutada solo bajo WSL/Ubuntu; matriz y validación real de gestor siguen pendientes |
| macOS | amd64 | Sí | listener loopback, persistencia, árbol y LaunchAgent | No verificado nativamente; seguimiento no bloqueante de este flujo |
| Windows | amd64 | Sí | listener, ACL, Task Scheduler y Job Object con hijo/nieto | No verificado nativamente; seguimiento no bloqueante de este flujo |

La workflow `calidad` compila estos tres objetivos; no programa pruebas nativas macOS/Windows en este flujo. Un resultado futuro de CI tampoco reemplaza el archivo de evidencia archivada: debe registrar SO, versión, arquitectura, runner, comandos, resultados redactados y revisión humana.

## Reglas de empaquetado

`scripts/build-release.sh` construye únicamente el binario de producto con `CGO_ENABLED=0`, `-trimpath`, `-buildvcs=false`, build ID vacío y versión/commit por linker. Genera:

- `mcp-gateway_<versión>_<goos>_<goarch>.tar.gz` para Linux/macOS;
- `mcp-gateway_<versión>_windows_amd64.zip` para Windows;
- `mcp-gateway_<versión>_checksums.txt` ordenado con SHA-256.

El script inspecciona `go version -m`, verifica que cada archivo comprimido contiene solo el binario de producto y vuelve a calcular los checksums. No distribuye fake MCP, fixtures, configuración, scripts ni runtime adicional.
