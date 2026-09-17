#!/usr/bin/env bash
set -euo pipefail

binary=""
out=""

while [[ $# -gt 0 ]]; do
  case "$1" in
    --binary) binary="${2:?missing --binary value}"; shift 2 ;;
    --out) out="${2:?missing --out value}"; shift 2 ;;
    *) echo "unknown argument: $1" >&2; exit 2 ;;
  esac
done

repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
if [[ -z "$out" ]]; then
  out="$(mktemp -d "${TMPDIR:-/tmp}/dws-a2ui-recipe-gallery.XXXXXX")"
fi
mkdir -p "$out"
out="$(cd "$out" && pwd)"

if [[ -z "$binary" ]]; then
  binary="$repo_root/dws"
  (cd "$repo_root" && DWS_PACKAGE_VERSION=0.0.0-test make build)
else
  binary="$(cd "$(dirname "$binary")" && pwd)/$(basename "$binary")"
fi

run_dws() {
  GOMEMLIMIT="${DWS_CARD_ACCEPTANCE_GOMEMLIMIT:-512MiB}" \
    GOGC="${DWS_CARD_ACCEPTANCE_GOGC:-20}" \
    "$binary" "$@"
}

recipes=(notification information approval task schedule report form)
run_dws card recipe list --format json > "$out/recipe-list.json"
registered_count="$(grep -o '"name"' "$out/recipe-list.json" | wc -l | tr -d ' ')"
if [[ "$registered_count" != "${#recipes[@]}" ]]; then
  echo "gallery covers ${#recipes[@]} recipes, but the CLI registers $registered_count" >&2
  exit 1
fi

for recipe in "${recipes[@]}"; do
  grep -Eq "\"name\"[[:space:]]*:[[:space:]]*\"$recipe\"" "$out/recipe-list.json" || {
    echo "gallery recipe is not registered by the CLI: $recipe" >&2
    exit 1
  }
  spec="$repo_root/internal/helpers/testdata/a2ui-${recipe}-spec.json"
  messages="$out/${recipe}-messages.json"
  preview="$out/${recipe}-preview.html"

  [[ -f "$spec" ]] || { echo "missing gallery spec: $spec" >&2; exit 1; }
  run_dws card compose --file "$spec" --output "$messages" --format json > "$out/${recipe}-compose.json"
  run_dws card lint --file "$messages" --mode create --format json > "$out/${recipe}-lint.json"
  run_dws card preview --file "$messages" --output "$preview" --format json > "$out/${recipe}-preview.json"
  grep -Fq 'data-component="Card"' "$preview" || { echo "$recipe preview has no Card" >&2; exit 1; }
done

grep -Fq 'data-component-id="decision_title"' "$out/approval-preview.html" || { echo "approval recipe has no decision section" >&2; exit 1; }
grep -Fq '审批意见（选填）' "$out/approval-preview.html" || { echo "approval recipe has no approval comment field" >&2; exit 1; }
grep -Fq 'data-event-name="approval_submit"' "$out/approval-preview.html" || { echo "approval recipe has no approval submit event" >&2; exit 1; }
if grep -Fq 'data-component-id="form_fields"' "$out/approval-preview.html"; then
  echo "approval recipe unexpectedly uses the data-entry field container" >&2
  exit 1
fi
grep -Fq 'data-component-id="form_intro"' "$out/form-preview.html" || { echo "form recipe has no collection instruction" >&2; exit 1; }
grep -Fq 'data-component-id="form_fields"' "$out/form-preview.html" || { echo "form recipe has no grouped field container" >&2; exit 1; }
grep -Fq 'data-event-name="form_submit"' "$out/form-preview.html" || { echo "form recipe has no form submit event" >&2; exit 1; }
grep -Fq 'class="ButtonControl borderless" data-event-name="form_cancel"' "$out/form-preview.html" || { echo "form recipe cancel action is not low emphasis" >&2; exit 1; }

cp "$repo_root/scripts/card/recipe-gallery.html" "$out/index.html"

for recipe in "${recipes[@]}"; do
  grep -Fq "${recipe}-preview.html?embed=1" "$out/index.html" || {
    echo "gallery does not reference $recipe preview" >&2
    exit 1
  }
done

component_count=0
for component in Button File Markdown TextField Link Card Row Column Tag Divider CollapsiblePanel Image Text ChoicePicker; do
  if grep -Fq "data-component=\"$component\"" "$out"/*-preview.html; then
    component_count=$((component_count + 1))
  else
    echo "gallery preview matrix is missing component: $component" >&2
    exit 1
  fi
done

printf '{"status":"PASS","recipes":%d,"components":%d,"previewKind":"reference_preview","remoteSideEffects":false,"realRenderer":false}\n' "${#recipes[@]}" "$component_count" > "$out/gallery-status.json"
echo "A2UI Recipe gallery: $out/index.html"
