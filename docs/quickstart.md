# Inicio rápido

## 1. Instalar

Siga la [guía de instalación](installation.md) y compruebe:

```bash
mcp-gateway version
```

## 2. Inicializar

```bash
mcp-gateway setup
```

Este comando crea la configuración, incorpora discovery y habilita el daemon en
`localhost:3333`.

Para usar otro puerto:

```bash
mcp-gateway setup --port 4444
```

## 3. Revisar downstreams

```bash
mcp-gateway list
mcp-gateway doctor
```

Añada cualquier MCP adicional antes de registrar proyectos:

```bash
mcp-gateway add ejemplo \
  --prefix ejemplo__ \
  --binary /ruta/al/servidor-mcp
```

## 4. Registrar el proyecto

Desde la raíz del repositorio:

```bash
mcp-gateway register-project
```

O indicando una ruta:

```bash
mcp-gateway register-project --project-dir /ruta/al/proyecto
```

El comando crea o actualiza `.mcp.json`, conserva otros servidores registrados
y añade el archivo a `.gitignore`.

Repita este paso en cada proyecto que usará el gateway.

## 5. Reiniciar el cliente

Cierre y vuelva a abrir el agente o su sesión para que cargue `.mcp.json`. Las
herramientas aparecerán con los prefijos configurados.

## 6. Verificar

```bash
mcp-gateway doctor --verbose
mcp-gateway list
```

La integración global con Claude Code es opcional:

```bash
mcp-gateway install-claude
```

Ese registro global no contiene `projectDir`; para contexto de proyecto use
siempre `register-project`.
