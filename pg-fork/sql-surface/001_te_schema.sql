-- Triple-entry SQL surface (Phase 3) — catalog + atomic capture + verification.
-- SYS-DECIDE-002 (trigger + transactional outbox), SYS-PG-002/003/005/006, SYS-HMAC-006, SYS-DECIDE-006/007.
-- Runs on stock PostgreSQL 18; the "fork" is PG18 + this schema + the Go writer.

CREATE SCHEMA IF NOT EXISTS te;

-- Per-ledger relationship + keys (SYS-DECIDE-006/007). Demo stores the writer master key in-DB;
-- a production deployment holds it in threshold custody (Phase 6, SYS-CUST-*).
CREATE TABLE IF NOT EXISTS te.relationship (
  stream_id        text PRIMARY KEY,
  writer_priv      bytea NOT NULL,            -- 32-byte secp256k1 master private scalar
  counterparty_pub bytea NOT NULL,            -- 33-byte compressed auditor/counterparty pubkey
  created_at       timestamptz NOT NULL DEFAULT now()
);

-- Which tables are journalled, to which stream, and their primary-key columns (row identity).
CREATE TABLE IF NOT EXISTS te.journalled (
  table_name   text PRIMARY KEY,             -- schema-qualified, e.g. 'public.accounts'
  stream_id    text NOT NULL REFERENCES te.relationship(stream_id),
  pk_columns   text[] NOT NULL,
  confidential boolean NOT NULL DEFAULT false
);

-- Transactional outbox: capture is atomic with the user's COMMIT (SYS-PG-003).
-- outbox.seq is the global commit order; per-stream M(c).seq is assigned by the writer.
CREATE TABLE IF NOT EXISTS te.outbox (
  seq         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  stream_id   text NOT NULL,
  table_name  text NOT NULL,
  row_id      bytea NOT NULL,
  column_id   text NOT NULL,
  op          smallint NOT NULL,             -- 1=INSERT 2=UPDATE 3=DELETE
  value       text,                          -- plaintext (NULL for delete)
  captured_at timestamptz NOT NULL DEFAULT now(),
  status      text NOT NULL DEFAULT 'pending',
  stream_seq  bigint,                         -- M(c).seq, filled by writer
  txid        bytea                           -- recording txid, filled by writer
);

-- Chain index (SYS-HMAC-006): (stream, per-stream seq) -> txid, rebuildable from chain alone.
CREATE TABLE IF NOT EXISTS te.chain_index (
  stream_id text   NOT NULL,
  seq       bigint NOT NULL,
  txid      bytea  NOT NULL,
  PRIMARY KEY (stream_id, seq)
);

-- Blinding factors for confidential fields (SYS-HMAC-009): the plaintext stays in the DB, the on-chain
-- change_image is commit(value, r); r is held here so the commitment can be opened/verified by the
-- entitled parties. The on-chain record never carries the plaintext.
CREATE TABLE IF NOT EXISTS te.blinding (
  stream_id text   NOT NULL,
  seq       bigint NOT NULL,
  r         bytea  NOT NULL,
  PRIMARY KEY (stream_id, seq)
);

-- Generic capture trigger: writes one outbox row per changed column, atomically (SYS-PG-002).
CREATE OR REPLACE FUNCTION te.capture() RETURNS trigger LANGUAGE plpgsql AS $$
DECLARE
  j           te.journalled%ROWTYPE;
  tname       text := TG_TABLE_SCHEMA || '.' || TG_TABLE_NAME;
  newj        jsonb;
  oldj        jsonb;
  pk          text;
  rid         bytea;
  k           text;
  v_new       text;
  v_old       text;
BEGIN
  SELECT * INTO j FROM te.journalled WHERE table_name = tname;
  IF NOT FOUND THEN
    RETURN NULL;
  END IF;

  IF TG_OP = 'DELETE' THEN
    oldj := to_jsonb(OLD);
  ELSE
    newj := to_jsonb(NEW);
    IF TG_OP = 'UPDATE' THEN oldj := to_jsonb(OLD); END IF;
  END IF;

  -- row identity = pipe-joined primary-key values
  SELECT string_agg(coalesce((COALESCE(newj, oldj) ->> col), ''), '|' ORDER BY ord)
    INTO pk
    FROM unnest(j.pk_columns) WITH ORDINALITY AS t(col, ord);
  rid := convert_to(pk, 'UTF8');

  IF TG_OP = 'DELETE' THEN
    FOR k IN SELECT jsonb_object_keys(oldj) LOOP
      IF k <> ALL (j.pk_columns) THEN
        INSERT INTO te.outbox(stream_id, table_name, row_id, column_id, op, value)
        VALUES (j.stream_id, tname, rid, k, 3, NULL);
      END IF;
    END LOOP;
    RETURN OLD;
  END IF;

  FOR k IN SELECT jsonb_object_keys(newj) LOOP
    IF k = ANY (j.pk_columns) THEN CONTINUE; END IF;
    v_new := newj ->> k;
    IF TG_OP = 'UPDATE' THEN
      v_old := oldj ->> k;
      CONTINUE WHEN v_new IS NOT DISTINCT FROM v_old;
      INSERT INTO te.outbox(stream_id, table_name, row_id, column_id, op, value)
      VALUES (j.stream_id, tname, rid, k, 2, v_new);
    ELSE
      INSERT INTO te.outbox(stream_id, table_name, row_id, column_id, op, value)
      VALUES (j.stream_id, tname, rid, k, 1, v_new);
    END IF;
  END LOOP;
  RETURN NEW;
END;
$$;

-- Register a table for journalling and attach the capture trigger (SYS-PG-005 — plain SQL/DDL).
CREATE OR REPLACE FUNCTION te.journal_table(p_table regclass, p_stream text, p_pk text[], p_confidential boolean DEFAULT false)
RETURNS void LANGUAGE plpgsql AS $$
DECLARE tn text;
BEGIN
  -- fully-qualified schema.table (matches TG_TABLE_SCHEMA||'.'||TG_TABLE_NAME in the trigger)
  SELECT n.nspname || '.' || c.relname INTO tn
    FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
   WHERE c.oid = p_table;
  INSERT INTO te.journalled(table_name, stream_id, pk_columns, confidential)
  VALUES (tn, p_stream, p_pk, p_confidential)
  ON CONFLICT (table_name) DO UPDATE SET stream_id = EXCLUDED.stream_id, pk_columns = EXCLUDED.pk_columns, confidential = EXCLUDED.confidential;
  EXECUTE format('DROP TRIGGER IF EXISTS te_capture ON %s', tn);
  EXECUTE format('CREATE TRIGGER te_capture AFTER INSERT OR UPDATE OR DELETE ON %s
                  FOR EACH ROW EXECUTE FUNCTION te.capture()', tn);
END;
$$;

-- SQL-callable verification surface (SYS-PG-006): for a row, show each column's latest on-chain anchor.
-- Cryptographic tag re-verification (ECDH-HMAC) is performed by the Go verifier / cold-rebuild, since
-- secp256k1 is unavailable in PL/pgSQL; this function reports anchoring + chain position.
CREATE OR REPLACE FUNCTION te.verify(p_table text, p_row text)
RETURNS TABLE(column_id text, op smallint, stream_seq bigint, txid bytea, anchored boolean)
LANGUAGE sql AS $$
  SELECT DISTINCT ON (o.column_id)
         o.column_id, o.op, o.stream_seq, o.txid, (o.txid IS NOT NULL) AS anchored
  FROM te.outbox o
  WHERE o.table_name = p_table AND o.row_id = convert_to(p_row, 'UTF8')
  ORDER BY o.column_id, o.seq DESC;
$$;

-- SQL-callable PDF render surface (SYS-DOC-005): returns the deterministic field-set + on-chain anchors
-- (object_id, per-column txid for the BURI) for a row; services-go/docrender turns this into the
-- byte-stable PDF paper copy (with embedded BURI + scannable QR).
CREATE OR REPLACE FUNCTION te.render_pdf(p_table text, p_row text)
RETURNS jsonb LANGUAGE sql AS $$
  SELECT jsonb_build_object(
    'object_id', p_table || '/' || p_row,
    'anchors', coalesce((
      SELECT jsonb_agg(jsonb_build_object(
        'column', column_id, 'op', op, 'stream_seq', stream_seq,
        'txid', encode(txid, 'hex'), 'anchored', anchored))
      FROM te.verify(p_table, p_row)), '[]'::jsonb));
$$;
