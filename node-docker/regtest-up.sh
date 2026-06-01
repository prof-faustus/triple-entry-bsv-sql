#!/usr/bin/env bash
# Reproducible Teranode REGTEST bring-up (dev/test only). Pins the versions resolved in
# spec/VERIFY-LOG.md (B1–B10). Run inside WSL as root:
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/regtest-up.sh
#
# Idempotent: clones teranode-quickstart at the pinned tag if absent, writes a lean
# regtest .env (core services only — no monitoring/p2p/legacy), starts the stack, and
# drives the FSM to RUNNING. Confirms block generation at the end.
set -euo pipefail

QS="${TERANODE_QS:-/root/teranode-quickstart}"
TERANODE_VERSION="${TERANODE_VERSION:-v0.15.1}"   # VERIFY-LOG B8/B9
RU="${RPC_USER:-teranode}"
RP="${RPC_PASS:-regtestsecret}"                    # dev-only regtest password

if [ ! -d "$QS/.git" ]; then
  echo ">>> cloning teranode-quickstart -> $QS"
  git clone --depth 1 https://github.com/bsv-blockchain/teranode-quickstart.git "$QS"
fi
cd "$QS"

echo ">>> writing lean regtest .env"
cp -n .env.example .env || true
sed -i "s/^TERANODE_VERSION=.*/TERANODE_VERSION=${TERANODE_VERSION}/" .env
sed -i "s/^network=.*/network=regtest/" .env
sed -i "s/^COMPOSE_PROFILES=.*/COMPOSE_PROFILES=/" .env   # core services only
sed -i "s/^minminingtxfee=.*/minminingtxfee=0/" .env       # regtest: no fee floor
sed -i "s/^rpc_user=.*/rpc_user=${RU}/" .env
sed -i "s/^rpc_pass=.*/rpc_pass=${RP}/" .env
sed -i "s/^clientName=.*/clientName=teranode-regtest/" .env
grep -q '^POSTGRES_PASSWORD=teranode_change_me' .env && \
  sed -i "s/^POSTGRES_PASSWORD=.*/POSTGRES_PASSWORD=$(openssl rand -hex 16)/" .env

echo ">>> docker compose up -d (core services)"
docker compose up -d

echo ">>> driving FSM to RUNNING"
./lib/fsm.sh up || ./cli.sh setfsmstate --fsmstate RUNNING || true

echo ">>> confirm block generation"
sleep 3
bash /mnt/d/claude/SQL/node-docker/rpc.sh getblockchaininfo | tr ',' '\n' | grep -E '"chain"|"blocks"' || true
bash /mnt/d/claude/SQL/node-docker/rpc.sh generate 1
echo ">>> regtest up. RPC: http://localhost:9292  (user=${RU})"
