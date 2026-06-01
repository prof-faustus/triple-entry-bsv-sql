#!/usr/bin/env bash
# Re-apply schema (incl. te.render_pdf), anchor via tewriter, and show te.render_pdf output.
#   wsl -u root -- bash /mnt/d/claude/SQL/services-go/run-renderpdf-check-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/services-go/renderpdf-check.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
export PGPASSWORD=te
bash /mnt/d/claude/SQL/pg-fork/sql-surface/apply.sh >/dev/null 2>&1 || true
cd /mnt/d/claude/SQL/services-go
go run ./cmd/tewriter --log /mnt/d/claude/SQL/services-go/bin/te_writer.log >/dev/null 2>&1 || true
echo "=== te.render_pdf('public.accounts','1') (SYS-DOC-005) ==="
psql -h 127.0.0.1 -p 5433 -U te -d te -tAc "SELECT jsonb_pretty(te.render_pdf('public.accounts','1'));"
