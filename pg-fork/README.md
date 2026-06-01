# pg-fork — forked PostgreSQL 18 with native BSV

A fork of **PostgreSQL 18** (`SYS-DECIDE-008`, locked; PostgreSQL License — fork-permissive,
upstream notices preserved) in which committed accounting writes are automatically mirrored to BSV
per Spec Section 5, with on-chain mechanics invisible to the SQL user (`SYS-CON-005`, `SYS-PG-001`).

## Subpackages
- `capture/` — committed-write interception (`SYS-PG-002/003`). Mechanism decided in
  `SYS-DECIDE-002` (WAL logical decoding vs C extension vs in-core hooks). MUST capture every
  accounting-relevant change exactly once, in commit order, with table/row/column identity for `M(c)`.
  Transactional outbox + async (default) / synchronous journalling modes (`SYS-DECIDE-003`).
- `bsv-native/` — key derivation, ECDH-HMAC tagging, tx build/sign/broadcast, hash-chain maintenance
  (`SYS-HMAC-001..011`). Consumes `crypto-core`. Data rides in spendable scripts only — no OP_RETURN,
  no P2SH (`SYS-CON-008`, `SYS-CON-002`, `SYS-ENC-001`).
- `sql-surface/` — DDL extensions + catalog tables (`triple_entry` table property, per-relationship
  key registration), `te_verify(table,row)` (`SYS-PG-006`), `te_render_pdf(object_id)` (`SYS-DOC-005`).

## Key requirements
- `SYS-PG-004` — DB↔chain index (`SYS-HMAC-006`); extract recorded value from chain; **cold-rebuild**
  the whole DB from chain + master keys, asserting byte-equality (Appendix B.4).
- `SYS-PG-007` — reorg re-evaluation, confirmation-depth gating (`SYS-DECIDE-004`), outbox idempotency
  across restarts (no double-record, no lost entry).
- `SYS-HMAC-010` — the chain entry, not the DB row, is the authoritative third entry.

## Exit criteria (Phase 3 / Appendix B.2–B.6)
Ordinary SQL on a real schema yields a correct, verifiable on-chain triple-entry log; full cold-rebuild
asserts byte-equality.

## Status
**BLOCKED:** no C toolchain or PostgreSQL build environment present yet; CTO substrate spec missing.
See `spec/STATUS.md`.
