#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
required=(
  "README.md"
  "CHANGELOG.md"
  "LICENSE"
  "docs/index.md"
)

for path in "${required[@]}"; do
  test -s "$root/$path" || { printf 'Falta documentación pública requerida: %s\n' "$path" >&2; exit 1; }
done

grep -q '^## \[Unreleased\]' "$root/CHANGELOG.md" || {
  printf 'CHANGELOG.md no contiene la sección Unreleased.\n' >&2
  exit 1
}
grep -q '<!-- flujo-e2e-verificado: inicio -->' "$root/README.md" || {
  printf 'README.md no contiene el flujo E2E verificado.\n' >&2
  exit 1
}

printf 'Documentación pública y marcadores de release presentes.\n'
