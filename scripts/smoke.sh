#!/usr/bin/env bash
# Full CLI integration smoke test: fresh temp pb_data + local mock provider.
#
# Flow (mirrors AGENTS.md "Smoke test reference flow"):
#   config template -> config import -> provider list -> provider from-preset glm
#   -> provider model add glm glm-5.3 -> start claude glm --model glm-5.3 --dry-run
#   -> provider add local (mock on 127.0.0.1:18990) -> test provider local
#   -> test models local -> profile add codex-local
#   -> start codex codex-local --dry-run -> config export --include-secrets
#
# Assertions match the current lipgloss card output (provider list cards,
# "Launch Plan" dry-run card). Every failed step exits non-zero immediately.
#
# Usage:
#   task smoke              # builds bin/aisw first, then runs this script
#   ./scripts/smoke.sh      # standalone; expects bin/aisw (run task build first)
#
# Env:
#   SMOKE_ADDR   mock provider listen address (default 127.0.0.1:18990)

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/.." && pwd)"
cd "${repo_root}"

aisw="${repo_root}/bin/aisw"
if [[ ! -x "${aisw}" ]]; then
  echo "bin/aisw not found — run 'task build' first (or use 'task smoke')" >&2
  exit 1
fi

addr="${SMOKE_ADDR:-127.0.0.1:18990}"
tmp="$(mktemp -d)"
data_dir="${tmp}/pb_data"
mock_pid=""

cleanup() {
  if [[ -n "${mock_pid}" ]]; then
    kill "${mock_pid}" 2>/dev/null || true
    wait "${mock_pid}" 2>/dev/null || true
  fi
  rm -rf "${tmp}"
}
trap cleanup EXIT

step() { printf '\n== %s ==\n' "$*"; }

run() { "${aisw}" --dir "${data_dir}" "$@"; }

assert_contains() { # <label> <haystack> <needle>
  if [[ "$2" != *"$3"* ]]; then
    echo "ASSERT FAILED [$1]: expected output to contain: $3" >&2
    echo "--- output ---" >&2
    echo "$2" >&2
    exit 1
  fi
  echo "ok: $1"
}

# ── mock provider ──────────────────────────────────────────────────────────
step "mock provider (${addr})"
if curl -sf "http://${addr}/health" >/dev/null 2>&1; then
  echo "port already serving a healthy mock — reusing it (not killed on exit)"
else
  go build -o "${tmp}/mock-provider" ./cmd/mock-provider
  "${tmp}/mock-provider" -http "${addr}" &
  mock_pid=$!
  ready=0
  for _ in $(seq 1 50); do
    if curl -sf "http://${addr}/health" >/dev/null 2>&1; then
      ready=1
      break
    fi
    sleep 0.1
  done
  if [[ "${ready}" != "1" ]]; then
    echo "mock provider failed to start on ${addr} (port occupied?)" >&2
    exit 1
  fi
  echo "mock provider up (pid ${mock_pid})"
fi

# ── smoke flow ─────────────────────────────────────────────────────────────
step "config template"
run config template --path "${tmp}/smoke.toml"

step "config import"
out="$(run config import --path "${tmp}/smoke.toml" --no-backup)"
echo "${out}"
assert_contains "config import" "${out}" "imported"

step "provider list"
out="$(run provider list)"
assert_contains "provider list (saved cards)" "${out}" "Saved Providers"
assert_contains "provider list (imported glm)" "${out}" "glm"
assert_contains "provider list (preset cards)" "${out}" "Built-in Provider Presets"

step "provider from-preset glm"
out="$(run provider from-preset glm --api-key smoke-glm-key)"
echo "${out}"
assert_contains "from-preset glm" "${out}" "saved provider glm"

step "provider model add glm glm-5.3"
out="$(run provider model add glm glm-5.3)"
echo "${out}"
assert_contains "model add" "${out}" "added model glm-5.3"

step "start claude glm --model glm-5.3 --dry-run"
out="$(run start claude glm --model glm-5.3 --dry-run)"
assert_contains "claude dry-run (plan card)" "${out}" "Launch Plan"
assert_contains "claude dry-run (model)" "${out}" "glm-5.3"

step "provider add local (mock)"
out="$(run provider add local \
  --base-url "http://${addr}/v1" \
  --api-key smoke-local-key \
  --protocol openai_chat \
  --model gpt-task)"
echo "${out}"
assert_contains "provider add local" "${out}" "saved provider local"

step "test provider local"
out="$(run test provider local)"
echo "${out}"
assert_contains "test provider (status line)" "${out}" "ok"
assert_contains "test provider (http 200)" "${out}" "200"

step "test models local"
out="$(run test models local)"
echo "${out}"
assert_contains "test models (mock model id)" "${out}" "gpt-task"

step "profile add codex-local"
out="$(run profile add codex-local --agent codex --provider local)"
echo "${out}"
assert_contains "profile add" "${out}" "saved profile codex-local"

step "start codex codex-local --dry-run"
out="$(run start codex codex-local --dry-run)"
assert_contains "codex dry-run (plan card)" "${out}" "Launch Plan"
assert_contains "codex dry-run (provider)" "${out}" "local"

step "config export --include-secrets"
out="$(run config export --include-secrets --path "${tmp}/export.toml")"
echo "${out}"
assert_contains "config export" "${out}" "exported"
assert_contains "export contains secret" "$(cat "${tmp}/export.toml")" "smoke-local-key"

step "SMOKE PASSED"
