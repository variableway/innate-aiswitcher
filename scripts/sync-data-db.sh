#!/usr/bin/env bash
# Copy the bundled data.db into the default innate-aiswitcher PocketBase data dir.
#
# Source (default): <repo-root>/data.db
# Target (default): ~/.innate-aiswitcher/pb_data/data.db
#
# Usage:
#   ./scripts/sync-data-db.sh
#   ./scripts/sync-data-db.sh /path/to/data.db
#   DEST_DIR=~/.innate-aiswitcher/pb_data ./scripts/sync-data-db.sh

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"

source="${1:-${SOURCE:-${repo_root}/data.db}}"
dest_dir="${DEST_DIR:-${HOME}/.innate-aiswitcher/pb_data}"
dest_file="${dest_dir}/data.db"

if [[ ! -f "${source}" ]]; then
  echo "source database not found: ${source}" >&2
  exit 1
fi

mkdir -p "${dest_dir}"

for sidecar in data.db-journal data.db-wal data.db-shm; do
  if [[ -f "${dest_dir}/${sidecar}" ]]; then
    rm -f "${dest_dir}/${sidecar}"
    echo "removed stale ${sidecar}"
  fi
done

if [[ -f "${dest_file}" ]]; then
  stamp="$(date +%Y%m%d-%H%M%S)"
  backup="${dest_dir}/data.db.bak-${stamp}"
  cp -f "${dest_file}" "${backup}"
  echo "backed up existing database to ${backup}"
fi

tmp="${dest_dir}/data.db.tmp"
cp -f "${source}" "${tmp}"
mv -f "${tmp}" "${dest_file}"

echo "synced ${source}"
echo "    -> ${dest_file}"
