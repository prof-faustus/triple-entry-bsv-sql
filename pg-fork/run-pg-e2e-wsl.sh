#!/usr/bin/env bash
# Phase-3 end-to-end in WSL: ordinary SQL (capture) -> tewriter (broadcast third entries) ->
# cold-rebuild == live DB, plus te.verify(). Node + PG18 are both local to WSL.
#   wsl -u root -- bash /mnt/d/claude/SQL/pg-fork/run-pg-e2e-wsl.sh
set -uo pipefail
exec > /mnt/d/claude/SQL/pg-fork/pg-e2e.out 2>&1
export PATH=/usr/local/go/bin:$PATH
export GOTOOLCHAIN=local
export PGPASSWORD=te
PSQL="psql -h 127.0.0.1 -p 5433 -U te -d te -tAc"

echo ">>> fresh schema + ordinary SQL (atomic capture)"
bash /mnt/d/claude/SQL/pg-fork/sql-surface/apply.sh
echo ">>> pending changes in outbox: $($PSQL "select count(*) from te.outbox where status='pending';")"

echo ">>> run tewriter (broadcast + cold-rebuild)"
cd /mnt/d/claude/SQL/services-go
go run ./cmd/tewriter --log /mnt/d/claude/SQL/services-go/bin/te_writer.log
RC=$?

echo ">>> te.verify(public.accounts, row '1'):"
psql -h 127.0.0.1 -p 5433 -U te -d te -c \
  "SELECT column_id, op, stream_seq, encode(txid,'hex') AS txid, anchored FROM te.verify('public.accounts','1');"
exit $RC
