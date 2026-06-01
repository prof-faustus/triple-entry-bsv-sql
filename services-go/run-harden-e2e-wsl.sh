#!/usr/bin/env bash
# Phase-7 hardening e2e in WSL (node local). Run:
#   wsl -u root -- bash /mnt/d/claude/SQL/services-go/run-harden-e2e-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/services-go/harden-e2e.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
cd /mnt/d/claude/SQL/services-go
go run ./cmd/hardene2e --log /mnt/d/claude/SQL/services-go/bin/te_harden.log
