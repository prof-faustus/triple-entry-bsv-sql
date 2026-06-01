#!/usr/bin/env bash
# Phase 0 / SYS-NODE-003 smoke test: chain reports state, generates blocks on demand,
# and the generated blocks are retrievable. Run inside WSL:
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/lib/smoke-regtest.sh
set -euo pipefail
RPC=/mnt/d/claude/SQL/node-docker/rpc.sh

echo "=== getblockchaininfo (chain + height) ==="
bash "$RPC" getblockchaininfo | tr ',' '\n' | grep -E '"chain"|"blocks"|"bestblockhash"' || true

echo "=== generate 2 blocks ==="
gen=$(bash "$RPC" generate 2)
echo "$gen"
h0=$(echo "$gen" | sed -E 's/.*\["([0-9a-f]+)".*/\1/')

echo "=== getblockchaininfo (height after) ==="
bash "$RPC" getblockchaininfo | tr ',' '\n' | grep -E '"blocks"' || true

echo "=== fetch one generated block by hash ($h0) ==="
bash "$RPC" getblock "\"$h0\"" | tr ',' '\n' | grep -E '"hash"|"height"|"merkleroot"|"tx"' | head -4 || true
