# Desarrollo y releases

## Requisitos

- Go 1.25 o posterior.
- Bash y herramientas estándar de empaquetado para builds de release.

## Preparar el repositorio

```bash
git clone https://github.com/andrespistoni/mcp-gateway.git
cd mcp-gateway
go test ./... -count=1
```

## Validación

```bash
test -z "$(gofmt -l .)"
bash scripts/check-coverage.sh
go test -race ./... -count=1
go vet ./...
bash -n scripts/*.sh
bash scripts/build-release.sh --dry-run
```

La cobertura de statements de producción debe ser estrictamente superior al
80%. El gate excluye el entrypoint mínimo y la infraestructura interna de MCPs
falsos.

## Builds

```bash
bash scripts/build-release.sh \
  --version v0.1.2 \
  --output /tmp/mcp-gateway-release
```

El script genera archivos para Linux, macOS y Windows en `amd64` y `arm64`,
verifica su contenido y crea el manifiesto SHA-256.

## Commits

Use Conventional Commits:

```text
feat(sse): add session admission limit
fix(daemon): preserve unit during failed restart
docs(config): explain project injection
```

Los cambios incompatibles usan `!` y un footer `BREAKING CHANGE:`.

## Preparar una release

1. Mueva las entradas de `CHANGELOG.md` desde `Unreleased` a
   `## [X.Y.Z] - YYYY-MM-DD`.
2. Deje una sección `Unreleased` vacía.
3. Ejecute la validación completa.
4. Cree y publique el tag:

```bash
git tag -a v0.1.2 -m "v0.1.2"
git push origin v0.1.2
```

El workflow valida el tag y el changelog, ejecuta calidad, construye los
artefactos y publica la GitHub Release con notas automáticas.
