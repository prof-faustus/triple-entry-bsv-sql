#!/usr/bin/env bash
# Run the FULL stack against the running SV Node wallet (:18443): token (P4), EDI/logistics (P5),
# PostgreSQL triple-entry (P3), hardening (P7). Each funds from the wallet and broadcasts to the SV Node.
#   wsl -u root -- bash /mnt/d/claude/SQL/services-go/run-all-svnode-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/services-go/all-svnode.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
B=/mnt/d/claude/SQL/services-go/bin
cd /mnt/d/claude/SQL/services-go

echo "##### PHASE 4: TOKEN #####"
go run ./cmd/tokene2e --log $B/te_token.log; echo "token rc=$?"

echo "##### PHASE 5: EDI + LOGISTICS #####"
go run ./cmd/edie2e --log $B/te_edi.log; echo "edi rc=$?"

echo "##### PHASE 3: POSTGRESQL TRIPLE-ENTRY #####"
bash /mnt/d/claude/SQL/pg-fork/sql-surface/apply.sh >/dev/null 2>&1 || true
go run ./cmd/tewriter --log $B/te_writer.log; echo "pg rc=$?"

echo "##### PHASE 7: HARDENING #####"
go run ./cmd/hardene2e --log $B/te_harden.log; echo "harden rc=$?"

echo "##### ALL DONE #####"
