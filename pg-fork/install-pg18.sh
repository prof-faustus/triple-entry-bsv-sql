#!/usr/bin/env bash
# Install PostgreSQL 18 (PGDG) in WSL on port 5433 (5432 may be held by Teranode's postgres),
# create the `te` role/db, and verify. Run: wsl -u root -- bash /mnt/d/.../pg-fork/install-pg18.sh
set -euo pipefail
exec > /mnt/d/claude/SQL/pg-fork/install-pg18.out 2>&1
export DEBIAN_FRONTEND=noninteractive

echo ">>> base deps"
apt-get update -qq
apt-get install -y -qq curl ca-certificates gnupg build-essential >/dev/null

if ! command -v pg_config >/dev/null 2>&1 || ! ls /usr/lib/postgresql/18 >/dev/null 2>&1; then
  echo ">>> add PGDG repo"
  install -d /usr/share/postgresql-common/pgdg
  curl -fsSL https://www.postgresql.org/media/keys/ACCC4CF8.asc -o /usr/share/postgresql-common/pgdg/apt.postgresql.org.asc
  . /etc/os-release
  echo "deb [signed-by=/usr/share/postgresql-common/pgdg/apt.postgresql.org.asc] https://apt.postgresql.org/pub/repos/apt ${VERSION_CODENAME}-pgdg main" \
    > /etc/apt/sources.list.d/pgdg.list
  apt-get update -qq
  echo ">>> install postgresql-18"
  apt-get install -y -qq postgresql-18 postgresql-client-18 postgresql-server-dev-18 >/dev/null
fi

echo ">>> configure cluster on port 5433"
CONF=/etc/postgresql/18/main/postgresql.conf
HBA=/etc/postgresql/18/main/pg_hba.conf
sed -i "s/^#\?port = .*/port = 5433/" "$CONF"
grep -q "127.0.0.1/32 scram-sha-256" "$HBA" || echo "host all all 127.0.0.1/32 scram-sha-256" >> "$HBA"
pg_ctlcluster 18 main restart || pg_ctlcluster 18 main start || true
sleep 2

echo ">>> psql version"
su postgres -c "psql -p 5433 -tAc 'select version();'"

echo ">>> create role+db te"
su postgres -c "psql -p 5433 -tAc \"SELECT 1 FROM pg_roles WHERE rolname='te'\"" | grep -q 1 \
  || su postgres -c "psql -p 5433 -c \"CREATE ROLE te LOGIN PASSWORD 'te' SUPERUSER;\""
su postgres -c "psql -p 5433 -tAc \"SELECT 1 FROM pg_database WHERE datname='te'\"" | grep -q 1 \
  || su postgres -c "psql -p 5433 -c \"CREATE DATABASE te OWNER te;\""

echo ">>> connect as te"
PGPASSWORD=te psql -h 127.0.0.1 -p 5433 -U te -d te -tAc "select 'te-connect-ok', version();"
echo "PG18 READY"
