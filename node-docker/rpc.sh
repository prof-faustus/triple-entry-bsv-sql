#!/usr/bin/env bash
# Minimal Teranode JSON-RPC client for the regtest stack. Reads creds from the
# quickstart .env. Usage (inside WSL):
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/rpc.sh <method> [json-params...]
# Examples:
#   rpc.sh getblockchaininfo
#   rpc.sh generate 3
#   rpc.sh getblockbyheight 1
set -euo pipefail

QS="${TERANODE_QS:-/root/teranode-quickstart}"
ENVF="$QS/.env"
RU=$(grep -E '^rpc_user=' "$ENVF" | head -1 | cut -d= -f2-)
RP=$(grep -E '^rpc_pass=' "$ENVF" | head -1 | cut -d= -f2-)
PORT="${RPC_PORT:-9292}"

method="${1:?usage: rpc.sh <method> [params...]}"; shift || true
params=""
for p in "$@"; do params+="${p},"; done
params="${params%,}"

curl -sS -u "${RU}:${RP}" -H 'Content-Type: application/json' \
  --data "{\"jsonrpc\":\"1.0\",\"id\":\"q\",\"method\":\"${method}\",\"params\":[${params}]}" \
  "http://localhost:${PORT}/"
echo
