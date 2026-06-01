#!/usr/bin/env bash
# Wipe the Teranode regtest to a clean chain (genesis) and bring it back to RUNNING.
# Does NOT touch the separate PG18 cluster on :5433. Run:
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/lib/reset-regtest.sh
set -uo pipefail
QS="${TERANODE_QS:-/root/teranode-quickstart}"
cd "$QS"
echo ">>> down -v (wipe teranode volumes)"
docker compose down -v 2>&1 | tail -3
echo ">>> start.sh"
./start.sh 2>&1 | tail -6
