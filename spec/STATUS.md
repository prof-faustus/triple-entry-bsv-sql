# STATUS.md — build status by requirement ID

Updated: 2026-06-01. Legend: ✅ done · 🟡 partial/in-progress · ⛔ blocked · ⬜ not started.

## Current phase: **Phase 0 — Spec freeze + decisions**

### Phase 0 exit criteria (Spec §13 / kickoff)
| Phase-0 task | Status | Evidence |
|---|---|---|
| Repo skeleton from §13 (`SYS-REPO-001`) | ✅ | `git init`; all §13 dirs created with per-dir READMEs mapping to requirement IDs; spec docs in `spec/`. |
| Resolve `SYS-DECIDE-*` | 🟡 | Locked (001/005/008/010) recorded; open (002/003/004/006/007/009) **proposed with rationale, pending operator confirmation** — `DECISIONS.md`. |
| Assign every `[VERIFY]` | 🟡 | `VERIFY-LOG.md`: PG 18.4, Teranode quickstart/RPC/services/regtest-gen **RESOLVED**; Kafka topics, exact image tag, Chronicle-current **OPEN**; miner policy = POLICY (build-time survey). |
| Pin node image | 🟡 | Release **v0.15.1 (2026-05-22)** identified; exact registry tag = `${TERANODE_VERSION}` placeholder, **OPEN** (B9). |
| Pin PG version | ✅ | **PostgreSQL 18.4** (`VERIFY-LOG.md` A1). |
| **Regtest comes up & produces blocks** (`SYS-NODE-001/003`, `SYS-CON-006`) | ⛔ | **Docker not installed in this environment** (`VERIFY-LOG.md` E2). `generate`/`generatetoaddress` confirmed available once the node runs (B5). |

**Phase 0 is NOT fully exited:** the regtest bring-up cannot be confirmed without Docker, and two
blockers (missing CTO spec E1; no C toolchain E3) gate downstream phases. See "Blockers" below.

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
| `SYS-NODE-001/002/003`, `SYS-CON-006` | ⛔ | Blocked on Docker (E2); interfaces pinned (B1–B6) ready to wire. |
| everything else (`SYS-HMAC-*`, `SYS-PG-*`, `SYS-TOK-*`, `SYS-EDI-*`, `SYS-LOG-*`, `SYS-DOC-*`, `SYS-PROOF-*`, `SYS-CUST-*`, `SYS-OVL-*`, `SYS-COMP-*`, `SYS-TEST-*`) | ⬜ | Phases 1–7. |

## Blockers (must be cleared to continue the autonomous run)
1. **⛔ Docker not installed (E2)** — required for Phase 0 exit (regtest) and all regtest tests.
2. **⛔ No C toolchain (E3)** — required for the PostgreSQL-18 C fork (Phase 3) and C crypto bindings
   (Phase 1). `node`/`go` present, so TS/Go work can proceed.
3. **⛔ Missing `CTO_BSV_Build_Spec_v1.md` (E1)** — the confidential-object substrate this system builds
   on; needed to ground commitments/AEAD (Phase 1) and threshold custody (Phase 6). The ECDH-HMAC
   keystone (§5, EP3860037A1) is self-contained and can proceed without it.

## What can proceed now without clearing blockers
- **Phase 1 (TS/Go side):** the crypto core — ECDH common-secret (EP3860037A1), HKDF, HMAC-SHA256, the
  canonical length-prefixed encoder/decoder, and shared KAT vectors — in TypeScript and Go, with vectors
  authored in `crypto-core/vectors/`. (The C side and CTO-grounded commitment/AEAD await E1/E3.)
