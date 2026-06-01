#!/usr/bin/env bash
# Switch the quickstart node to TERATESTNET with the correct per-network knobs (Spec §14 / .env.example).
# GATED: this only WRITES the config. It will NOT bring the node up or broadcast unless TERATESTNET_CONFIRM=1
# is set, because moving off regtest is a STOP-AND-ASK step (public network, funded keys, miner-policy
# survey). See node-docker/TERATESTNET.md.
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/teratestnet-up.sh
set -euo pipefail
QS="${TERANODE_QS:-/root/teranode-quickstart}"
ENVF="$QS/.env"
PEER="${TERATESTNET_PEER:-57.130.17.176:38333}" # teratestnet has no DNS seeder; set an explicit peer

cp -n "$QS/.env.example" "$ENVF" 2>/dev/null || true
sed -i "s/^network=.*/network=teratestnet/" "$ENVF"
sed -i "s/^minminingtxfee=.*/minminingtxfee=0.00000001/" "$ENVF"        # 1 sat/kb
sed -i "s/^blockmaxsize=.*/blockmaxsize=1073741824/" "$ENVF"            # 1 GiB (teratestnet cap)
sed -i "s/^excessiveblocksize=.*/excessiveblocksize=1073741824/" "$ENVF"
sed -i "s/^COMPOSE_PROFILES=.*/COMPOSE_PROFILES=legacy,p2p/" "$ENVF"     # need peering on a shared network
sed -i "s|^legacy_config_ConnectPeers=.*|legacy_config_ConnectPeers=${PEER}|" "$ENVF"
echo "configured teratestnet .env (peer ${PEER}); profiles=legacy,p2p"
grep -E "^(network|minminingtxfee|blockmaxsize|excessiveblocksize|COMPOSE_PROFILES|legacy_config_ConnectPeers)=" "$ENVF"

if [ "${TERATESTNET_CONFIRM:-0}" != "1" ]; then
  echo
  echo "STOP-AND-ASK: not starting. Moving off regtest needs operator go-ahead + a FUNDED teratestnet key."
  echo "To proceed after providing funds: TERATESTNET_CONFIRM=1 bash $0  (then ./start.sh syncs the node)."
  exit 0
fi
cd "$QS" && ./start.sh
