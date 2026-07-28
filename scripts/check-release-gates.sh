#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
required=(
  "docs/evidence/claude.md"
  "docs/evidence/native-matrix.md"
  "docs/evidence/environment.md"
  "docs/release-checklist.md"
)

for path in "${required[@]}"; do
  test -f "$root/$path" || { printf 'Falta evidencia requerida: %s\n' "$path" >&2; exit 1; }
done

if grep -R -qE '^Estado: PENDIENTE|^- \[ \]' "${required[@]/#/$root/}"; then
  printf 'Advertencia de release: queda evidencia ambiental pendiente; no bloquea este flujo de implementación ni la construcción de candidatas.\n' >&2
  exit 0
fi

printf 'Los documentos no declaran pendientes. Una publicación real todavía requiere revisión y autorización humana explícitas.\n'
