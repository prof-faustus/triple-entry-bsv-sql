#!/usr/bin/env bash
# Phase-4 token e2e in WSL (node local). Run:
#   wsl -u root -- bash /mnt/d/claude/SQL/tokenisation/run-token-e2e-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/tokenisation/token-e2e.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
cd /mnt/d/claude/SQL/services-go
go run ./cmd/tokene2e --log /mnt/d/claude/SQL/services-go/bin/te_token.log
