#!/usr/bin/env bash
set -euo pipefail

# Genera candidatos reproducibles en un runner Unix con GNU tar, gzip, zip y
# sha256sum. No publica ni evalúa los gates ambientales de release.
root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="0.0.0-dev"
commit="$(git -C "$root" rev-parse HEAD)"
output="$root/dist"
dry_run=false

while (($#)); do
  case "$1" in
    --version) version="$2"; shift 2 ;;
    --commit) commit="$2"; shift 2 ;;
    --output) output="$2"; shift 2 ;;
    --dry-run) dry_run=true; shift ;;
    *) printf 'uso: %s [--version X.Y.Z] [--commit SHA] [--output DIR] [--dry-run]\n' "$0" >&2; exit 2 ;;
  esac
done

release="${version#v}"
if [[ "$output" != /* ]]; then
  output="$PWD/$output"
fi
source_date_epoch="${SOURCE_DATE_EPOCH:-$(git -C "$root" show -s --format=%ct "$commit")}"
targets=(
  "linux/amd64"
  "linux/arm64"
  "darwin/amd64"
  "darwin/arm64"
  "windows/amd64"
  "windows/arm64"
)

run() {
  if "$dry_run"; then
    printf '+ '
    printf '%q ' "$@"
    printf '\n'
    return
  fi
  "$@"
}

if "$dry_run"; then
  printf 'Candidato: versión=%s commit=%s SOURCE_DATE_EPOCH=%s\n' "$release" "$commit" "$source_date_epoch"
  for target in "${targets[@]}"; do
    IFS=/ read -r goos goarch <<<"$target"
    extension=""
    [[ "$goos" == windows ]] && extension=".exe"
    run env CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -buildvcs=false -ldflags "-buildid= -X mcp-gateway/internal/version.release=$release -X mcp-gateway/internal/version.commit=$commit" -o "STAGE/mcp-gateway$extension" ./cmd/mcp-gateway
  done
  printf 'No se crearon artefactos ni se evaluaron gates ambientales.\n'
  exit 0
fi

command -v tar >/dev/null
command -v gzip >/dev/null
command -v zip >/dev/null
command -v unzip >/dev/null
command -v sha256sum >/dev/null
tar --version | grep -q 'GNU tar'

rm -rf "$output"
mkdir -p "$output"
export SOURCE_DATE_EPOCH="$source_date_epoch"

for target in "${targets[@]}"; do
  IFS=/ read -r goos goarch <<<"$target"
  stage="$output/.stage-$goos-$goarch"
  mkdir -p "$stage"
  extension=""
  [[ "$goos" == windows ]] && extension=".exe"
  binary="$stage/mcp-gateway$extension"
  env CGO_ENABLED=0 GOOS="$goos" GOARCH="$goarch" go build -trimpath -buildvcs=false \
    -ldflags "-buildid= -X mcp-gateway/internal/version.release=$release -X mcp-gateway/internal/version.commit=$commit" \
    -o "$binary" ./cmd/mcp-gateway
  touch -d "@$SOURCE_DATE_EPOCH" "$binary"
  go version -m "$binary"
  if [[ "$goos" == windows ]]; then
    archive="$output/mcp-gateway_${release}_${goos}_${goarch}.zip"
    (cd "$stage" && zip -X -q "$archive" "mcp-gateway$extension")
    [[ "$(unzip -Z1 "$archive")" == "mcp-gateway$extension" ]]
  else
    archive="$output/mcp-gateway_${release}_${goos}_${goarch}.tar.gz"
    tar --sort=name --mtime="@$SOURCE_DATE_EPOCH" --owner=0 --group=0 --numeric-owner -C "$stage" -cf - "mcp-gateway$extension" | gzip -n >"$archive"
    [[ "$(tar -tzf "$archive")" == "mcp-gateway$extension" ]]
  fi
  rm -rf "$stage"
done

(cd "$output" && LC_ALL=C sha256sum mcp-gateway_*.tar.gz mcp-gateway_*.zip | LC_ALL=C sort >"mcp-gateway_${release}_checksums.txt" && sha256sum --check "mcp-gateway_${release}_checksums.txt")
printf 'Candidatos creados y verificados en %s.\n' "$output"
