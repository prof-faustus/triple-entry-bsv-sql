# STATUS.md — build status by requirement ID

Updated: 2026-06-01. Legend: ✅ done · 🟡 partial/in-progress · ⛔ blocked · ⬜ not started.

## Current phase: **Phase 4 — Definable token ✅ COMPLETE** (operator-confirmed past checkpoint)

> Progress: Phases 0–3 ✅, **Phase 4 ✅** (definable token: cash profiles, goods, atomic swap,
> external-linkage adapter contract). Next: Phase 5 (EDI DFA + bridge + logistics).

### Phase 4 result (2026-06-01) — PASS (Appendix B.7, on regtest)
- `services-go/token` + `cmd/tokene2e`; defs in `tokenisation/token-defs.json`.
- **`SYS-TOK-005`**: 4 token types defined from data, **no code change** (issuer-backed / satoshi-tagged
  / pegged-CBDC / goods).
- **`SYS-TOK-001/004`, `SYS-CASH-001/002`**: each token is a UTXO lineage carrying metadata + a
  third-entry record in a spendable envelope; mint/transfer/redeem all journalled; lineages tag-verified.
- **`SYS-TOK-003`**: satoshi binding `f(value, pegging_rate)` + min threshold.
- **`SYS-TOK-006`**: external linkage via the **adapter contract only** (`RailAdapter` interface +
  `MockAdapter`); real CBDC/stablecoin rail integration remains gated behind STOP-AND-ASK.
- **`SYS-TOK-007`**: atomic two-token swap in ONE tx (deliver-vs-deliver), both legs verified.
- All token scripts native (no P2SH/OP_RETURN). Reproduce: `tokenisation/run-token-e2e-wsl.sh`.

### Phase 3 — PostgreSQL fork ✅ COMPLETE (vertical slice)

### Phase 3 result (2026-06-01) — PASS (PostgreSQL 18.4 + regtest, in WSL)
- **PG 18.4** installed (PGDG) in WSL on :5433; `pg-fork` = PG18 + `te` schema + Go writer.
- **Capture** (`SYS-PG-002/003`, `SYS-DECIDE-002`): `te.capture()` AFTER trigger writes per-changed-column
  rows to `te.outbox` **atomically with the user's commit**; verified 4 changes from plain INSERT/UPDATE.
- **Writer** (`services-go/cmd/tewriter`): drains outbox in commit order → `M(c)→tag` → broadcasts a
  hash-chained third-entry stream on regtest (spendable envelope, `SIGHASH_ALL|FORKID`) → `te.chain_index`.
- **Cold-rebuild** (`SYS-PG-004`, Appendix B.4): from chain head txid + master keys alone, tag-verified
  every entry and reconstructed the ledger == **live DB** ✓.
- **`te.verify()`** (`SYS-PG-006`): SQL-callable, returns each column's on-chain anchor (seq + txid).
- Reproduce: `node-docker/regtest-up.sh` (node), `pg-fork/install-pg18.sh` (db), `pg-fork/run-pg-e2e-wsl.sh` (e2e).

**Scope honesty:** vertical slice — single demo table, plaintext fields, PL/pgSQL trigger capture (not
in-core), keys in-DB (not yet threshold custody), async single-writer. `te_render_pdf`, confidential
commitments path, reorg/idempotency hardening, and the full multi-table/multi-stream story remain.

### Phase 2c result (2026-06-01) — PASS on live Teranode regtest
`services-go/cmd/streame2e` ran end-to-end against the regtest node (executed inside WSL to avoid a
Windows-exe launch stall under Defender; node is local to WSL):
- **Funded** from a matured coinbase (50 BSV) paid to a controlled key via `generatetoaddress`;
  located via `merkleroot == coinbase txid` + `getrawtransaction`, matured with +100 blocks.
- **Broadcast** a 4-entry hash-chained ECDH-HMAC stream; each tx carries its record in a spendable
  `OP_FALSE OP_IF <REC> OP_ENDIF + P2PKH` envelope (no OP_RETURN/P2SH), signed `SIGHASH_ALL|FORKID`,
  spending the prior tx's output so the UTXO lineage **is** the stream (`prev_txid` links `M(c)`).
- **Discovery**: each tag recomputed from `M(c)`+keys matched the on-chain record.
- **Cold-rebuild**: from the head txid + master keys alone, walked the chain via `prev_txid`, verified
  every tag, and reconstructed the ledger (2 cells across 4 chained changes) == source.
- Record format refined so it is self-describing: REC now embeds `encode(M(c))` (ALGORITHMS.md §1.2);
  TS+Go vectors regenerated, parity green. Reproduce: `node-docker/lib/run-e2e-wsl.sh`.

**Phase 2 exit (Appendix B.2/B.5/B.6) met.**

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
| `SYS-ENC-001/002`, `SYS-CON-002/008` (spendable data envelope) | ✅ | `services-go/bsvscript` carriers + static check; **broadcast & accepted on regtest** in the e2e (B.5). |
| `SYS-HMAC-005/006/008` (tag on-chain, discovery, hash chain) | ✅ | e2e: tag in spendable script; recomputed-tag discovery; `prev_txid` chain walked in cold-rebuild (B.2). |
| `SYS-ENC-004` (sighash binds successors) | ✅ | all stream txs signed `SIGHASH_ALL\|FORKID` and accepted (B.6). |
| `SYS-PG-004` (cold-rebuild from chain) | ✅ | `tewriter` cold-rebuilds `public.accounts` from chain+keys == live DB (Appendix B.4, vertical slice). |
| `SYS-PG-001/002/003/005/006` | 🟡 | PG18 fork + atomic trigger capture + outbox + `te.journal_table`/`te.verify` SQL surface (vertical slice; hardening/te_render_pdf later). |
| `SYS-NODE-002` (RPC + events) | 🟡 | RPC client (`services-go/node`) broadcasts/queries live; Kafka event relay still later wiring. |
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
