#!/usr/bin/env bash
# Validate the SPV/BURI proof layer against LIVE BSV testnet (WhatsOnChain). Run:
#   wsl -u root -- bash /mnt/d/claude/SQL/services-go/run-spvtestnet-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/services-go/spvtestnet.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
cd /mnt/d/claude/SQL/services-go
go run ./cmd/spvtestnet
