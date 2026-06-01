#!/usr/bin/env bash
# Phase-5 EDI/logistics e2e in WSL (node local). Run:
#   wsl -u root -- bash /mnt/d/claude/SQL/edi-dfa/run-edi-e2e-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/edi-dfa/edi-e2e.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
cd /mnt/d/claude/SQL/services-go
go run ./cmd/edie2e --log /mnt/d/claude/SQL/services-go/bin/te_edi.log
