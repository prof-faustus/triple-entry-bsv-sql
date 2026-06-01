# Tokenised Commercial System on Bitcoin SV — Triple-Entry SQL

Monorepo for the system specified in [`spec/TripleEntry_BSV_SQL_Build_Spec_v8.md`](spec/TripleEntry_BSV_SQL_Build_Spec_v8.md)
(the **Spec** — the build contract). The kickoff brief that drives the phased build
is [`spec/TripleEntry_BSV_SQL_ClaudeCode_Kickoff.md`](spec/TripleEntry_BSV_SQL_ClaudeCode_Kickoff.md).

A fork of PostgreSQL 18 with native BSV embedded: ordinary SQL writes are mirrored,
per-field and per-change, onto Bitcoin SV as an immutable, ECDH-HMAC-bound, hash-chained
**third entry** (triple-entry accounting). On top sit four packages sharing one substrate:
a triple-entry ledger (A), tokenised EDI documents (B), multi-currency tokenised cash (C),
and tokenised logistics/goods (D).

## Non-negotiable constraints (build failures if violated)

- **BSV only.** No BTC constructs: no SegWit, Taproot, witness, Lightning, RBF (`SYS-CON-001`).
- **No P2SH** — consensus-prohibited on BSV. Native locking scripts only: P2PKH, bare
  `OP_CHECKSIG`, bare `OP_CHECKMULTISIG` (`SYS-CON-002`).
- **No `OP_RETURN` — at all.** All on-chain data rides as pushdata in **spendable** locking
  scripts (`OP_FALSE OP_IF <data> OP_ENDIF`, or `<data> OP_DROP` before the auth opcodes)
  (`SYS-CON-008`, `SYS-ENC-001`).
- **ECDH-HMAC keystone** (`SYS-CON-003`, Section 5): every accounting change is bound to an
  ECDH-keyed HMAC tag carried on-chain in a spendable script; entries form a hash chain
  (`SYS-CON-004`); the DB is rebuildable from the chain alone.
- **SQL-simple** (`SYS-CON-005`): the user runs ordinary SQL; on-chain mechanics are invisible.
- **Node is Teranode** (`SYS-DECIDE-010`), via `teranode-quickstart` Docker; dev/test on regtest.

## Repository layout (Spec Section 13, `SYS-REPO-001`)

| Path | Purpose | Primary requirement IDs |
|---|---|---|
| `node-docker/` | Dockerised Teranode (regtest) + compose | `SYS-NODE-001..003`, `SYS-CON-006` |
| `pg-fork/` | Forked PostgreSQL 18 + native BSV (C) | `SYS-PG-001..007` |
| `pg-fork/capture/` | Committed-write interception (WAL/extension/in-core) | `SYS-PG-002/003`, `SYS-DECIDE-002` |
| `pg-fork/bsv-native/` | Key derivation, ECDH-HMAC, tx build/broadcast, hash chain | `SYS-HMAC-001..011` |
| `pg-fork/sql-surface/` | DDL extensions, catalog, `te_verify()`, `te_render_pdf()` | `SYS-PG-005/006`, `SYS-DOC-005` |
| `crypto-core/` | Shared ECDH/HKDF/HMAC/commitment (C + TS/Go), KAT vectors | `SYS-HMAC-*`, `SYS-TEST-003` |
| `tokenisation/` | Definable token primitive + cash/CBDC linkage + goods | `SYS-TOK-001..007`, `SYS-CASH-*`, `SYS-GOODS-001` |
| `edi-dfa/` | Commercial-document DFAs (PO/invoice/payment/shipping) | `SYS-EDI-001..004` |
| `edi-bridge/` | Optional X12/EDIFACT ↔ on-chain DFA translation (per partner) | `SYS-EDI-005/006`, `SYS-DECIDE-005` |
| `doc-render/` | Deterministic PDF paper copies + BURI/QR embed | `SYS-DOC-001..005` |
| `logistics/` | Consignment lifecycle + DHT goods records | `SYS-LOG-001..012` |
| `sdk-ts/` | Client SDK (TypeScript) | `SYS-PG-005` |
| `services-go/` | Indexer, relay, broadcaster glue to node | `SYS-NODE-002`, `SYS-HMAC-006` |
| `spec/` | Spec + requirements + `VERIFY-LOG.md` + `DECISIONS.md` | `SYS-INTEG-*` |
| `tests/` | regtest e2e, KAT, adversarial, cold-rebuild | `SYS-TEST-001..003` |

## Phased delivery (Spec Section 13, `SYS-PHASE-001`)

No phase begins before the prior phase's exit criteria pass (`SYS-PHASE-002`).

0. **Spec freeze + decisions** — skeleton, pin node/PG, resolve decisions & `[VERIFY]`.
1. **Crypto core + KAT** — ECDH common-secret, HKDF, HMAC, commitment, encoder; cross-impl vectors green.
2. **Node + hash-chain log** — Teranode regtest; hash-chained ECDH-HMAC TX stream; cold-rebuild toy stream.
3. **PostgreSQL fork** — write interception + outbox + async/sync + `te_verify()` + cold-rebuild real schema.
4. **Definable token** — cash (3 profiles) + goods + atomic swap + external-linkage adapter contract.
5. **EDI DFA + bridge + logistics** — document DFAs, consignment lifecycle, X12/EDIFACT bridge.
6. **Proofs / custody / overlay / computation** — SPV+BURI, threshold custody, overlay CKD, staked computation.
7. **Hardening** — reorg, idempotency, confirmation gating, security review, testnet.

See [`spec/VERIFY-LOG.md`](spec/VERIFY-LOG.md) and [`spec/DECISIONS.md`](spec/DECISIONS.md) for the
resolved/open verification gates and deployment decisions, and [`spec/STATUS.md`](spec/STATUS.md)
for current build status against each requirement ID.

## Status

Phase 0 in progress. See [`spec/STATUS.md`](spec/STATUS.md). **Prerequisites not yet present in this
environment** (Docker, a C toolchain, a PostgreSQL build environment) and a **missing dependency**
(`CTO_BSV_Build_Spec_v1.md`, the substrate this system builds on) are tracked there — these gate the
node bring-up and the C-side builds.
