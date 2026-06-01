#!/usr/bin/env bash
# Run the Phase-2 stream e2e inside WSL (Linux), avoiding Windows-exe launch issues.
# The Teranode regtest node is local to WSL (localhost:9292).
#   wsl -u root -- bash /mnt/d/claude/SQL/node-docker/lib/run-e2e-wsl.sh
set -euo pipefail
# self-redirect all output to a Windows-readable file regardless of caller capture
exec > /mnt/d/claude/SQL/services-go/bin/e2e-wsl.out 2>&1

# Ensure a recent Go toolchain (>=1.23) is available in WSL.
if ! command -v go >/dev/null 2>&1 || ! go version | grep -qE 'go1\.(2[3-9]|[3-9][0-9])'; then
  echo ">>> installing Go toolchain in WSL"
  VER=$(curl -fsSL https://go.dev/VERSION?m=text | head -1)
  curl -fsSL "https://go.dev/dl/${VER}.linux-amd64.tar.gz" -o /tmp/go.tgz
  rm -rf /usr/local/go && tar -C /usr/local -xzf /tmp/go.tgz
fi
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
go version

cd /mnt/d/claude/SQL/services-go
echo ">>> go run ./cmd/streame2e"
go run ./cmd/streame2e --log /mnt/d/claude/SQL/services-go/bin/te_e2e.log
