#!/usr/bin/env bash
set -euo pipefail

root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
threshold="${MCP_GATEWAY_COVERAGE_THRESHOLD:-80}"
profile="$(mktemp)"
trap 'rm -f "$profile"' EXIT HUP INT TERM

cd "$root"
go test ./... -count=1 -covermode=atomic -coverprofile="$profile"

awk -v threshold="$threshold" '
  NR > 1 &&
  $1 !~ /\/cmd\// &&
  $1 !~ /\/internal\/testsupport\// {
    total += $2
    if ($3 > 0) {
      covered += $2
    }
  }
  END {
    if (total == 0) {
      print "No se encontraron statements de producción." > "/dev/stderr"
      exit 1
    }
    percentage = 100 * covered / total
    printf "Cobertura de producción: %.2f%% (%d/%d statements; requerido > %.2f%%)\n",
      percentage, covered, total, threshold
    if (percentage <= threshold) {
      exit 1
    }
  }
' "$profile"
