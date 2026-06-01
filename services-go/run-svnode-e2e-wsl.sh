#!/usr/bin/env bash
# Fund + test the keystone e2e against the running SV Node regtest WALLET (:18443). Run:
#   wsl -u root -- bash /mnt/d/claude/SQL/services-go/run-svnode-e2e-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/services-go/svnode-e2e.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
cd /mnt/d/claude/SQL/services-go
go run ./cmd/svnodee2e --log /mnt/d/claude/SQL/services-go/bin/te_svnode.log
