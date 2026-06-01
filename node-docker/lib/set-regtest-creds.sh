#!/usr/bin/env bash
# One-shot: set deterministic regtest RPC creds in the quickstart .env, recreate the
# rpc container, and prove block generation. Run inside WSL:
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/lib/set-regtest-creds.sh
set -euo pipefail

QS="${TERANODE_QS:-/root/teranode-quickstart}"
ENVF="$QS/.env"
RU=teranode
RP=regtestsecret

[ -f "$ENVF" ] || { echo "no .env at $ENVF" >&2; exit 1; }
sed -i "s/^rpc_user=.*/rpc_user=${RU}/" "$ENVF"
sed -i "s/^rpc_pass=.*/rpc_pass=${RP}/" "$ENVF"
echo "creds in .env:"; grep -nE '^rpc_user=|^rpc_pass=' "$ENVF"

cd "$QS"
docker compose up -d --force-recreate rpc >/dev/null 2>&1
sleep 4

rpc() {
  curl -sS -u "${RU}:${RP}" -H 'Content-Type: application/json' \
    --data "{\"jsonrpc\":\"1.0\",\"id\":\"q\",\"method\":\"$1\",\"params\":[${2:-}]}" \
    http://localhost:9292/
  echo
}

echo "=== getblockcount (before) ==="; rpc getblockcount
echo "=== generate 3 ===";            rpc generate 3
echo "=== getblockcount (after) ===";  rpc getblockcount
