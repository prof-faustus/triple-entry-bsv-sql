#!/usr/bin/env bash
# Capture image digests, Kafka topics, and node version to close VERIFY-LOG gates B7/B9/B10.
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/lib/inspect.sh
set -uo pipefail
QS="${TERANODE_QS:-/root/teranode-quickstart}"
cd "$QS"

echo "===== image digests (B9) ====="
docker compose images 2>/dev/null | awk 'NR==1 || /teranode|postgres|redpanda|aerospike/'

echo; echo "===== Kafka (Redpanda) topics (B7) ====="
KAFKA=$(docker compose ps --format '{{.Name}}' | grep -i kafka | head -1)
if [ -n "${KAFKA:-}" ]; then
  docker exec "$KAFKA" rpk topic list 2>/dev/null || docker exec "$KAFKA" rpk topic list --brokers localhost:9092 2>/dev/null || echo "rpk topic list unavailable"
else
  echo "no kafka container found"
fi

echo; echo "===== Teranode version / Chronicle (B10) ====="
BC=$(docker compose ps --format '{{.Name}}' | grep -E 'blockchain' | head -1)
docker inspect --format '{{ index .Config.Labels "org.opencontainers.image.version" }} {{ index .Config.Labels "org.opencontainers.image.revision" }}' "$BC" 2>/dev/null || true
docker exec "$BC" sh -lc 'teranode-cli version 2>/dev/null || /app/teranode-cli version 2>/dev/null || echo "(cli version n/a)"' 2>/dev/null || true
