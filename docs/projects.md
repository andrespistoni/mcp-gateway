# Proyectos y projectDir

## Registro por proyecto

```bash
cd /ruta/al/proyecto
mcp-gateway register-project
```

El gateway crea o actualiza:

```text
<proyecto>/.mcp.json
```

La entrada administrada tiene esta forma:

```json
{
  "mcpServers": {
    "mcp-gateway": {
      "type": "sse",
      "url": "http://localhost:3333/sse?projectDir=%2Fruta%2Fal%2Fproyecto"
    }
  }
}
```

El comando conserva las demás claves y servidores de `.mcp.json`. También añade
`.mcp.json` a `.gitignore` sin duplicar la entrada.

## Cómo se determina el proyecto

1. El cliente abre la URL almacenada en `.mcp.json`.
2. El gateway valida y asocia `projectDir` a la sesión SSE.
3. Cada llamada conserva ese contexto durante toda la sesión.
4. Si el downstream tiene `inject_project`, el gateway añade la ruta al
   argumento configurado.

El gateway nunca sobrescribe un valor de proyecto enviado por el cliente.

## Cuándo es necesario

El contexto es útil para MCPs que operan sobre:

- archivos;
- repositorios Git;
- herramientas de compilación;
- configuraciones locales;
- bases de datos o recursos seleccionados por proyecto.

Servicios globales como búsqueda web o APIs remotas pueden no necesitarlo.

## Configurar inyección

```bash
mcp-gateway add filesystem \
  --prefix filesystem__ \
  --binary /ruta/al/filesystem-mcp \
  --inject-project projectDir
```

`projectDir` aporta contexto, pero el downstream sigue ejecutándose con los
permisos normales del usuario.

## Varios proyectos

Ejecute `register-project` en cada repositorio. Cada `.mcp.json` contiene su
propia ruta y produce sesiones independientes.
