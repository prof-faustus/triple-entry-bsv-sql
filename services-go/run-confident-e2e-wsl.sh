#!/usr/bin/env bash
# Confidential-field path (SYS-HMAC-009) against the SV Node wallet. Run:
#   wsl -u root -- bash /mnt/d/claude/SQL/services-go/run-confident-e2e-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/services-go/confident-e2e.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
cd /mnt/d/claude/SQL/services-go
go run ./cmd/confidente2e --log /mnt/d/claude/SQL/services-go/bin/te_confidential.log
echo "----- SQL te.render_pdf (SYS-DOC-005) -----"
PGPASSWORD=te psql -h 127.0.0.1 -p 5433 -U te -d te -tAc "SELECT te.render_pdf('public.accounts','1');" 2>&1 || true
