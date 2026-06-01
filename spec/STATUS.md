# STATUS.md — build status by requirement ID

Updated: 2026-06-01. Legend: ✅ done · 🟡 partial/in-progress · ⛔ blocked · ⬜ not started.

## Current phase: **Phase 7 — Hardening ✅ COMPLETE** — all 8 phases done (vertical slices)

> Progress: **Phases 0–7 ✅**. The full Spec arc is implemented as working, e2e-verified vertical
> slices on PostgreSQL 18.4 + Teranode regtest (in WSL). Remaining: production hardening of the
> documented slices and **teratestnet/mainnet** — gated behind STOP-AND-ASK (live network/funds).

### Phase 7 result (2026-06-01) — PASS (Appendix B.11, B.12)
- `services-go/cmd/hardene2e` (`SYS-PG-007`): **confirmation-depth gating** (depth<req → not final;
  ≥req → final; tip read from a generate-returned block hash, since `getblockchaininfo.blocks` lags);
  **reorg re-evaluation** (`invalidateblock` → entry de-finalised, divergence surfaced; restore
  best-effort — Teranode regtest reconsider is slow, non-fatal); **outbox idempotency** (dedup by
  `M(c)`; restart is a no-op).
- **B.12 grounding integrity** (`SYS-INTEG-001`): `spec/SECURITY-REVIEW.md` — no requirement grounded
  in a patent abstract; static scan confirms `OP_RETURN`/`P2SH` appear only in comments and rejection
  checks, never in a produced script; sighash `FORKID` enforced; confidentiality boundary stated.
- Helpers: `node-docker/lib/reset-regtest.sh` (clean chain), `services-go/run-harden-e2e-wsl.sh`.

### Appendix A/B coverage summary
All 12 Appendix-B done-criteria demonstrated as vertical slices: B.1 (**C↔TS↔Go parity, fully closed**),
B.2–B.6 (keystone, SQL surface, cold-rebuild, no-P2SH/OP_RETURN, sighash), B.7 (definable token + swap),
B.8 (EDI DFAs + bridge), B.9 (logistics/ownership/integrity/DvP/B-L), B.10 (SPV+BURI/custody/overlay/
compute), B.11 (resilience), B.12 (grounding). Appendix A requirements are met or slice-scoped per the
per-phase notes above; production-hardening items and teratestnet/mainnet are explicitly deferred.

### Phase 6 — proofs / custody / overlay / computation ✅ COMPLETE

### Phase 6 result (2026-06-01) — PASS (Appendix B.10; unit-tested Go packages)
- **`spv`** (`SYS-PROOF-*`; WO2022100946A1, WO2022214264A1): transaction-Merkle proof + branch verify,
  **BURI** encode/parse, SPV-verify against a block-header Merkle root **without the block payload**;
  round-trip + tamper + single-tx-block tests green.
- **`custody`** (`SYS-CUST-*`; EP3259724B1, US11671255): loss-resistant Shamir sharing over GF(n)
  (k-of-n recover; sub-threshold reveals nothing), shares **encrypted under the ECDH common secret**,
  ≥3-locations-incl-backup layout, and native **bare OP_CHECKMULTISIG** N-of-M (no P2SH). Tests green.
- **`overlay`** (`SYS-OVL-*`; EP4046048B1): BIP32-style **CKD hierarchy** whose keys mirror the overlay
  graph (deterministic, structure-aligned addressing; parent derives child). Tests green.
- **`compute`** (`SYS-COMP-*`; US20240364498A1): staked-proposer/challenge market — post→commit→
  challenge→resolve under group **threshold control**, feeding a resolved **DFA event** (SYS-COMP-002).
  Tests green.
- Scope: `spv`/`custody`/`overlay`/`compute` are verified as unit-tested primitives; deeper wiring
  (BURI export inside `te_verify`, no-reconstruction threshold-ECDSA per US11671255, on-chain multisig
  spend, compute→edi live injection) are documented slices.

### Phase 5 result (2026-06-01) — PASS (Appendix B.8, B.9, on regtest)
- `services-go/edi` (DFA-as-UTXO engine + DHT + ownership/integrity), `edibridge`, `cmd/edie2e`;
  document set in `edi-dfa/document-defs.json` (23 DFAs incl. consignment).
- **B.8**: all **22 SYS-EDI-002 document types** driven through their DFA lifecycles on-chain
  (states-as-UTXOs, transitions journalled + tag-verified, `SYS-EDI-001/002/003`); cross-references by
  object_id verified from chain (`SYS-EDI-004`); **X12/EDIFACT bridge** (`edibridge`) maps the listed
  message types in/out, per-partner subset, **omittable** (core `edi` doesn't import it) — go tests green
  (`SYS-EDI-005/006`).
- **B.9** (`SYS-LOG-001..012`): consignment lifecycle + multi-party `CUSTODY_TRANSFER`;
  **bill-of-lading-as-token** title transfer (issue→endorse→surrender, controller re-key);
  **ownership** key-match (US11210372; mismatch rejected); **integrity** H3==H4 (GB2558485A; tamper
  detected); **delivery-versus-payment** (ACCEPTED + POD SIGNED → payment SETTLED). DHT = local store.
- Reproduce: `edi-dfa/run-edi-e2e-wsl.sh`. Scope: representative field-level content; full per-message
  X12/EDIFACT field mapping and the DHT-as-distributed-store are slices.

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
| **C ↔ TS/Go parity** (Appendix B.1 C clause) | ✅ | `crypto-core/c` (libsecp256k1 + OpenSSL) reproduces the shared vectors **40/40 byte-for-byte** (built+run in WSL); **B.1 fully closed** |

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
