# STATUS.md — build status by requirement ID

Updated: 2026-06-01. Legend: ✅ done · 🟡 partial/in-progress · ⛔ blocked · ⬜ not started.

## Current phase: **Phase 2 — Node + hash-chain log** (2a ✅ node up, 2b ✅ envelope, 2c in progress)

> Progress: Phase 0 ✅ (regtest confirmed), Phase 1 ✅ (TS/Go crypto core, parity green),
> Phase 2a ✅ (Teranode regtest up in WSL Docker), Phase 2b ✅ (spendable data-envelope builder).
> **Phase 2c (broadcast + discovery + cold-rebuild on regtest) is the remaining Phase-2 work.**

### Phase 2c plan (de-risked 2026-06-01; not yet built)
Funding mechanism **verified viable** on Teranode regtest:
- `generatetoaddress <addr>` mines a coinbase with a P2PKH output to a key we control; for a
  single-tx block **`merkleroot == coinbase txid`**, and `getrawtransaction <txid>` returns the raw
  coinbase (3 standard P2PKH outputs observed). Mine +100 blocks for coinbase maturity, then spend.
- go-sdk confirmed for tx build/sign: `transaction.NewTransaction`, `AddInputFrom`,
  `p2pkh.Unlock(priv, &sighash.AllForkID)`, `tx.Sign()`, `tx.Hex()`/`TxID()`; broadcast via
  `sendrawtransaction`; mine via `generate`.

Remaining build (Go, `services-go`): fund → build a stream of N txs, each carrying a hash-chained
ECDH-HMAC record in a spendable envelope (`bsvscript`), spending the prior tx's P2PKH output so the
UTXO lineage **is** the stream; broadcast each; tag-discovery index `(table,row,col,seq)→txid`;
cold-rebuild the toy stream from genesis + master keys, asserting equality. Exit: Appendix B.2/B.5/B.6.

**Design refinement needed for self-contained cold-rebuild (`SYS-PG-004`, `SYS-HMAC-010`):** the
on-chain record must carry the canonical `M(c)` (table/row/column/op + seq + prev_txid) so the row is
reconstructable from chain alone. Plan: redefine REC = `MAGIC_R, version, stream_id, encode(M(c)),
image_kind, change_image, tag` (ALGORITHMS.md §1.2). This changes only the `encodedRecord` KAT vector
(GV/CS/tag are unaffected); regenerate vectors and re-run TS+Go parity.

---

## Phase 1 — Crypto core + KAT (TS/Go; C side deferred)

> Phase 0 governance is frozen (skeleton + DECISIONS + VERIFY-LOG committed). Operator decisions
> (2026-06-01): build the Phase 1 crypto core in **TS/Go now**, **defer the C binding** until a
> toolchain exists (so Appendix B.1's C-parity clause stays open); **re-derive the CTO primitives**
> with documented assumptions (`spec/CTO-PRIMITIVES.md`). The regtest bring-up (Docker, E2) remains a
> Phase-0 item to close before Phase 2.

### Phase 0 exit criteria (Spec §13 / kickoff)
| Phase-0 task | Status | Evidence |
|---|---|---|
| Repo skeleton from §13 (`SYS-REPO-001`) | ✅ | `git init`; all §13 dirs created with per-dir READMEs mapping to requirement IDs; spec docs in `spec/`. |
| Resolve `SYS-DECIDE-*` | 🟡 | Locked (001/005/008/010) recorded; open (002/003/004/006/007/009) **proposed with rationale, pending operator confirmation** — `DECISIONS.md`. |
| Assign every `[VERIFY]` | 🟡 | `VERIFY-LOG.md`: PG 18.4, Teranode quickstart/RPC/services/regtest-gen **RESOLVED**; Kafka topics, exact image tag, Chronicle-current **OPEN**; miner policy = POLICY (build-time survey). |
| Pin node image | 🟡 | Release **v0.15.1 (2026-05-22)** identified; exact registry tag = `${TERANODE_VERSION}` placeholder, **OPEN** (B9). |
| Pin PG version | ✅ | **PostgreSQL 18.4** (`VERIFY-LOG.md` A1). |
| **Regtest comes up & produces blocks** (`SYS-NODE-001/003`, `SYS-CON-006`) | ✅ | Docker Engine in WSL2 (E2 resolved); Teranode `v0.15.1` regtest **FSM RUNNING**, all core services healthy; `generate` mines blocks on demand; `getblock` retrieves them; `chain="regtest"`. Reproducible via `node-docker/regtest-up.sh`. |

**Phase 0 is now fully exited** (regtest confirmed). Remaining caveats: C toolchain deferred (E3, by
operator) and the CTO spec re-derived rather than supplied (E1). See "Blockers / deferrals".

### Phase 1 results (TS/Go; C deferred)
| Deliverable (Appendix B.1) | Status | Evidence |
|---|---|---|
| ECDH common-secret (EP3860037A1) | ✅ | `commonSecretAsWriter`/`AsCounterparty`; symmetric (writer-side == counterparty-side) in both impls |
| HKDF-SHA256 | ✅ | `deriveHmacKey`; RFC-5869 case-1 KAT green (TS+Go) |
| HMAC-SHA256 | ✅ | `tag`; RFC-4231 case-2 KAT green (TS+Go) |
| SHA-256 blinded commitment | ✅ | `commit`; binding/hiding/determinism tests |
| Canonical length-prefixed encoder/decoder | ✅ | `Writer`/`Reader`, message + record; round-trip + rejection tests |
| AEAD (AES-256-GCM, re-derived) | ✅ | encrypt/decrypt + tamper-fail; vector match (TS+Go) |
| **Cross-impl vectors identical** (`SYS-TEST-003`) | ✅ | Go `TestCoreVectors` recomputes the TS-generated `crypto-core/vectors/core_vectors.json` byte-for-byte; 14 TS + 7 Go tests green |
| **C ↔ TS/Go parity** (Appendix B.1 C clause) | ⏸ OPEN | C binding deferred (E3); to be added when a toolchain exists |

Run: `cd crypto-core/ts && NODE_OPTIONS=--use-system-ca npm test` · `cd crypto-core/go && go test ./...`
Regenerate vectors (TS is the source of truth): `cd crypto-core/ts && npm run gen-vectors`.

## Requirement coverage snapshot (Appendix A)
All 110 requirements are ⬜ **not started** except the Phase-0 governance items below. This is a
freeze/scaffold state — no functional code yet.

| ID(s) | Status | Note |
|---|---|---|
| `SYS-REPO-001` | ✅ | Skeleton matches §13. |
| `SYS-INTEG-002` | 🟡 | `[VERIFY]` register established; OPEN gates listed. |
| `SYS-INTEG-003` | 🟡 | Decisions recorded; open ones proposed, not silently assumed. |
| `SYS-DECIDE-001/005/008/010` | ✅ | Locked, recorded. |
| `SYS-DECIDE-002/003/004/006/007/009` | 🟡 | Proposed, pending confirmation. |
| `SYS-NODE-001/003`, `SYS-CON-006` | ✅ | Teranode regtest up in WSL Docker; blocks on demand; reproducible (`node-docker/`). |
| `SYS-NODE-002` (RPC + events) | 🟡 | RPC verified (getblockchaininfo/generate/getblock); Kafka topics identified (B7); `services-go` wiring is Phase 2c. |
| `SYS-ENC-005` (canonical encoding) | 🟡 | Encoder/decoder done in TS+Go (`crypto-core`); C side + on-chain script wiring later. |
| `SYS-ENC-001/002`, `SYS-CON-002/008` (spendable data envelope) | 🟡 | `services-go/bsvscript`: OP_FALSE OP_IF / OP_DROP carriers + native P2PKH tail; data round-trip; **static check rejects P2SH/OP_RETURN/data-only** (Appendix B.5 logic). Go tests green. On-chain broadcast = Phase 2c. |
| `SYS-HMAC-002/003/004` (GV/subkeys/CS, K_hmac, tag) | 🟡 | Algorithms implemented + KAT-green in TS+Go; on-chain placement (`SYS-HMAC-005`), hash chain (`008`), discoverability (`006`) are Phase 2. |
| `SYS-HMAC-009` (blinded commitment) | 🟡 | `commit` done TS+Go. |
| `SYS-TEST-003` (cross-impl vectors) | 🟡 | TS↔Go green; C clause open. |
| `SYS-SUB-001` (CTO primitives) | 🟡 | Re-derived + documented (`CTO-PRIMITIVES.md`); ECDH/HKDF/HMAC/commitment/AEAD implemented; UTXO-lineage/time-locked recovery/Tier F-S-T later. |
| everything else (`SYS-PG-*`, `SYS-TOK-*`, `SYS-EDI-*`, `SYS-LOG-*`, `SYS-DOC-*`, `SYS-PROOF-*`, `SYS-CUST-*`, `SYS-OVL-*`, `SYS-COMP-*`) | ⬜ | Phases 2–7. |

## Blockers / deferrals
1. **⛔ Docker not installed (E2)** — required to close the Phase 0 regtest item and for Phase 2 + all
   regtest tests. Still open; revisit before Phase 2.
2. **⏸ No C toolchain (E3) — DEFERRED by operator** — C crypto binding (Phase 1) and the PostgreSQL-18
   fork (Phase 3) wait for a toolchain. TS/Go proceed now; **Appendix B.1 C-parity clause stays open**.
3. **✍ Missing CTO spec (E1) — RESOLVED-BY-DERIVATION** — primitives re-derived & documented in
   `spec/CTO-PRIMITIVES.md`, flagged for review; reconcile if the authoritative CTO spec is supplied.
