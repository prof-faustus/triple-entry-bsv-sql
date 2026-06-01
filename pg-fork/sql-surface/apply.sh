#!/usr/bin/env bash
# Apply the te schema + demo and show the captured outbox. Run in WSL:
#   wsl -u root -- bash /mnt/d/claude/SQL/pg-fork/sql-surface/apply.sh
set -euo pipefail
exec > /mnt/d/claude/SQL/pg-fork/sql-surface/apply.out 2>&1
export PGPASSWORD=te
PSQL="psql -h 127.0.0.1 -p 5433 -U te -d te -v ON_ERROR_STOP=1"
D=/mnt/d/claude/SQL/pg-fork/sql-surface

echo ">>> reset demo objects"
$PSQL -c "DROP TABLE IF EXISTS public.accounts CASCADE;" -c "DROP SCHEMA IF EXISTS te CASCADE;"

echo ">>> 001_te_schema.sql"; $PSQL -f "$D/001_te_schema.sql"
echo ">>> 002_demo.sql";      $PSQL -f "$D/002_demo.sql"

echo ">>> live table:"
$PSQL -c "SELECT * FROM public.accounts ORDER BY id;"
echo ">>> captured outbox (atomic third-entry capture):"
$PSQL -c "SELECT seq, table_name, convert_from(row_id,'UTF8') AS row, column_id, op, value, status FROM te.outbox ORDER BY seq;"
echo "APPLY OK"
