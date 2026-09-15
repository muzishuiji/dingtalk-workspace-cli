#!/usr/bin/env bash
set -euo pipefail

layer="source"
binary=""
out=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --layer) layer="${2:?missing --layer value}"; shift 2 ;;
    --binary) binary="${2:?missing --binary value}"; shift 2 ;;
    --out) out="${2:?missing --out value}"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [[ -z "$out" ]]; then
  out="$(mktemp -d "${TMPDIR:-/tmp}/dws-card-acceptance.XXXXXX")"
fi
mkdir -p "$out"
out="$(cd "$out" && pwd)"

run_source() {
  cd "$repo_root"
  DWS_PACKAGE_VERSION=0.0.0-test go test ./internal/card/a2ui/... -count=1
  DWS_PACKAGE_VERSION=0.0.0-test go test ./internal/helpers -run '^TestCrossPlatformCoverageCard' -count=1
  printf '{"layer":"source","status":"PASS"}\n' > "$out/source.json"
}

ensure_binary() {
  if [[ -n "$binary" ]]; then
    binary="$(cd "$(dirname "$binary")" && pwd)/$(basename "$binary")"
    return
  fi
  binary="$out/dws"
  cd "$repo_root"
  DWS_PACKAGE_VERSION=0.0.0-test go build -o "$binary" ./cmd
}

run_preview() {
  ensure_binary
  local spec="$repo_root/internal/helpers/testdata/a2ui-approval-spec.json"
  "$binary" card compose --file "$spec" --output "$out/messages.json" --format json > "$out/compose.json"
  "$binary" card lint --file "$out/messages.json" --mode create --format json > "$out/lint.json"
  "$binary" card preview --file "$out/messages.json" --output "$out/preview.html" --format json > "$out/preview.json"
  printf '{"layer":"preview","status":"PASS","previewKind":"reference_preview","realRenderer":false}\n' > "$out/preview-status.json"
}

case "$layer" in
  source) run_source ;;
  preview) run_preview ;;
  all) run_source; run_preview ;;
  *) echo "unsupported layer: $layer (use source, preview, or all)" >&2; exit 2 ;;
esac

echo "A2UI acceptance artifacts: $out"
