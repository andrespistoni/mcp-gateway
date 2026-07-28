# Evidencia nativa por plataforma

Estado: PENDIENTE — seguimiento no bloqueante de este flujo

El alcance funcional conserva Linux, macOS y Windows. Este flujo solo ejecutó la suite en Linux bajo WSL/Ubuntu y no ejecutó runners nativos macOS ni Windows. Linux tampoco aporta todavía una validación real de gestor de servicios ni una matriz corporativa aprobada.

| Plataforma | Estado de este flujo | Evidencia que falta para una validación nativa futura |
|---|---|---|
| Linux | Suite Go ejecutada bajo WSL/Ubuntu | versión/arquitectura aprobadas, listener, persistencia, árbol y systemd de usuario reales |
| macOS | No verificado nativamente | versión/arquitectura aprobadas, listener, persistencia, árbol y LaunchAgent reales |
| Windows | No verificado nativamente | versión/arquitectura aprobadas, listener, ACL, Task Scheduler y Job Object desde creación con hijo/nieto en cancelación, timeout, crash y shutdown |

Los cross-builds no sustituyen estas ejecuciones. La prueba Windows de Job Object antes asociada a T-013 fue retirada del conjunto de tareas ejecutables por la decisión explícita de la persona usuaria; sigue como riesgo aceptado y seguimiento no bloqueante, no como evidencia realizada.
