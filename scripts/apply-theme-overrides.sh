#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
OVERRIDE_ROOT="${ROOT_DIR}/theme-overrides/theme-Joe3/templates"
TARGET_ROOT="${ROOT_DIR}/halo-data/themes/theme-Joe3/templates"

if [ ! -d "${OVERRIDE_ROOT}" ]; then
  echo "[theme-overrides] no overrides found, skip."
  exit 0
fi

if [ ! -d "${TARGET_ROOT}" ]; then
  echo "[theme-overrides] target theme directory not found: ${TARGET_ROOT}"
  echo "[theme-overrides] skip syncing."
  exit 0
fi

copy_file() {
  local relative_path="$1"
  local src="${OVERRIDE_ROOT}/${relative_path}"
  local dst="${TARGET_ROOT}/${relative_path}"

  if [ ! -f "${src}" ]; then
    echo "[theme-overrides] source missing, skip: ${relative_path}"
    return
  fi

  mkdir -p "$(dirname "${dst}")"
  cp -f "${src}" "${dst}"
  echo "[theme-overrides] applied: ${relative_path}"
}

copy_file "assets/js/post.js"
copy_file "assets/js/journals.js"
copy_file "modules/macro/tail.html"
copy_file "post.html"
copy_file "moments.html"
copy_file "moment.html"

echo "[theme-overrides] done."

