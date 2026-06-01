# VERIFY-LOG.md — `[VERIFY]` gate register

Per `SYS-INTEG-002`, every value/opcode/API/RPC/ecosystem fact that must be confirmed against a
current authoritative source is a gate resolved here **before code relies on it**. Status legend:
**RESOLVED** (confirmed against a cited source on the date shown) · **OPEN** (not yet confirmed; code
must not depend on it) · **POLICY** (an operator/deployment choice, not a fixed fact).

Research date: **2026-06-01**. Re-confirm at build time — the ecosystem moves (Chronicle activated
Apr 2026; node minor versions and miner policies change), per `SYS-VERIFY-LIST`.

---

## A. PostgreSQL (`SYS-PG-001`, `SYS-DECIDE-008`)

| # | Item | Status | Finding / source |
|---|---|---|---|
| A1 | Latest PostgreSQL 18 minor | **RESOLVED** | **18.4**, released **2026-05-14** (latest 18.x). Fork target = 18.4. Source: postgresql.org news. |
| A2 | Licence permits fork | **RESOLVED** | PostgreSQL License (permissive, OSI-approved); fork/modify/redistribute allowed; **upstream copyright/licence notice MUST be preserved**. Source: postgresql.org/about/licence. |

## B. Teranode node (`SYS-NODE-001..003`, `SYS-DECIDE-010`)

| # | Item | Status | Finding / source |
|---|---|---|---|
| B1 | Quickstart bring-up | **RESOLVED** | `git clone bsv-blockchain/teranode-quickstart` → `./setup.sh` (interactive: pick **regtest** + mode, generates `.env`) → `./start.sh` (`docker compose up` with the chosen profiles). Source: teranode-quickstart README. |
| B2 | Compose files | **RESOLVED** | `docker-compose.yml` (root), `compose/docker-teranode.yml`, `compose/docker-services.yml`; network selected via `network=regtest` in `.env`. Source: teranode-quickstart. |
| B3 | RPC interface | **RESOLVED** | JSON-RPC over **HTTP Basic Auth**; admin creds `rpc_user`/`rpc_pass`, limited creds `rpc_limit_user`/`rpc_limit_pass`. Bind **`127.0.0.1:9292`** (never exposed externally). Source: teranode docs RPC reference + quickstart. |
| B4 | RPC methods (for our use) | **RESOLVED** | Documented set includes: `createrawtransaction`, `sendrawtransaction`, `getrawtransaction`, `getrawmempool`, `getblock`, `getblockbyheight`, `getblockhash`, `getblockheader`, `getbestblockhash`, `getblockchaininfo`, `getchaintips`, `getinfo`, `getmininginfo`, `getminingcandidate`, `submitminingsolution`, `generate`, `generatetoaddress`, `invalidateblock`, `reconsiderblock`, `getdifficulty`, `getpeerinfo`, `freeze`/`unfreeze`/`reassign`, ban mgmt, `stop`, `version`, `help`. Source: teranode docs RPC reference. |
| B5 | Regtest block generation | **RESOLVED** | `generate` (mine N blocks, "testing only") and `generatetoaddress` (same, reward to address) are present → satisfies `SYS-NODE-003` instant block generation. Reorg testing: `invalidateblock`/`reconsiderblock`. Source: teranode docs RPC reference. |
| B6 | Microservices (named) | **RESOLVED** | Propagation, Validator, Block Assembly, Block Validation, Blockchain, **Asset Server**, Subtree Validation, Alert, **UTXO Persister**, Block Persister, P2P, Pruner, Legacy. Source: teranode docs architecture. |
| B7 | Event/notification backbone | **RESOLVED (mechanism)** / **OPEN (topics)** | Eventing is **Kafka** (Redpanda on `:9092`; Kafka Console UI `:8080`; config `config/kafka-console-config.yml`, protobuf). **Exact topic names (new-block / new-tx) are NOT enumerated in the consulted pages → OPEN; confirm from the "Kafka Overview" docs page before the relay (`services-go`) depends on a topic name.** Source: teranode docs (Kafka Overview) + quickstart. |
| B8 | Latest release tag | **RESOLVED** | **v0.15.1** (2026-05-22), 32 releases total. Source: github.com/bsv-blockchain/teranode releases. |
| B9 | Exact Docker image ref/tag | **OPEN** | Quickstart uses a `${TERANODE_VERSION}` placeholder in `.env` (bumped by `./update.sh`); the concrete registry URL/tag (e.g. `ghcr.io/...:vX.Y.Z`) was **not stated** in the consulted pages. Pin the exact image ref pulled by `docker compose` at build. |
| B10 | Chronicle-current confirmation | **OPEN** | Spec requires the Teranode build be **Chronicle-current** (mainnet activation 7 Apr 2026). Neither the repo README nor the quickstart consulted pages mention Chronicle → **confirm that the pinned tag (v0.15.1 or later) is Chronicle-current** from release notes before mainnet/testnet use. |

## C. Script / consensus / sighash (`SYS-CON-002`, `SYS-ENC-001/002/004`) — from Spec §14.1

| # | Item | Status | Finding |
|---|---|---|---|
| C1 | P2SH prohibited | **RESOLVED (per Spec §14.1)** | Consensus rule post-Genesis (confirms `SYS-CON-002`). Re-confirm against bitcoin-sv consensus docs at build. |
| C2 | Pushdata/stack limits | **RESOLVED (per Spec §14.1)** | Pre-Genesis 520-byte/1000-elem caps replaced by configurable stack-memory limit; large pushes in spendable scripts valid (supports `SYS-ENC-001`). |
| C3 | Multisig semantics | **RESOLVED (per Spec §14.1)** | Bare `OP_CHECKMULTISIG` native N-of-M; leading dummy item required pre-Chronicle; sigs in pubkey order; Chronicle relaxes NULLDUMMY/NULLFAIL for tx version > 1. |
| C4 | `OP_FALSE OP_IF … OP_ENDIF` data pattern | **RESOLVED (per Spec §14.1)** | Valid; Chronicle relaxes MINIMALIF. |
| C5 | Sighash | **RESOLVED (per Spec §14.1)** | `SIGHASH_FORKID` always set; BIP143 digest baseline; bind successors with `SIGHASH_ALL\|FORKID`; OTDA opt-in via Chronicle `0x20` for tx version > 1. |

> C1–C5 are taken from the Spec's resolved §14.1 and **re-confirmed against current bitcoin-sv
> consensus/Chronicle release notes at build time** (Spec marks these current-as-of 1 Jun 2026).

## D. Miner relay policy (`SYS-VERIFY-LIST`) — POLICY, surveyed at build

| # | Item | Status | Note |
|---|---|---|---|
| D1 | `maxscriptsizepolicy` | **POLICY** | Default 500 KB/script post-Genesis, **miner-configurable**; survey target miners (MinerID coinbase docs). |
| D2 | Fees / dust thresholds | **POLICY** | Miner-configurable; survey at build. |
| D3 | Max tx size policy | **POLICY** | Default 10 MB (consensus 1 GB), miner-configurable. |
| D4 | Confirmation depth | **POLICY** | `SYS-DECIDE-004` — operator-set per use case; see `DECISIONS.md`. |

## E. Dependencies & environment prerequisites (this repo / host)

| # | Item | Status | Note |
|---|---|---|---|
| E1 | **CTO substrate spec** (`CTO_BSV_Build_Spec_v1.md`) | **RESOLVED-BY-DERIVATION (2026-06-01)** | Spec line 15 / `SYS-SUB-001` / `SYS-CON-007` build *on* the CTO confidential-object primitive (secp256k1 ECDH, HKDF, AEAD, SHA-256 commitments, UTXO-lineage objects, time-locked recovery, Tier F/S/T profiles). **The file is not present in this repo or on disk.** **Operator decision (2026-06-01): re-derive the needed CTO primitives from standard constructions + EP3860037A1, document the assumptions explicitly, and flag for review** — see `spec/CTO-PRIMITIVES.md`. If the authoritative CTO spec is later supplied, reconcile against it (any divergence is a `[VERIFY]` to resolve). |
| E2 | **Docker** on host | **OPEN — BLOCKER** | Not installed in the current environment → Phase 0 "regtest comes up and produces blocks" (`SYS-NODE-001/003`, `SYS-CON-006`) cannot be confirmed; all regtest e2e/adversarial tests (`SYS-TEST-001`) blocked. |
| E3 | **C toolchain** (gcc/clang, make) | **DEFERRED (operator decision 2026-06-01)** | Not present → the PostgreSQL-18 C fork (`pg-fork`, Phase 3) and the C crypto bindings cannot be built here. **Operator decision: build the Phase 1 crypto core in TS/Go now and defer the C side until a toolchain exists.** Appendix B.1 (C + TS/Go parity) therefore stays **open** until the C binding lands; TS↔Go parity is provable now. `node` (v24) and `go` (1.26) present. |
| E5 | **npm TLS / CA** | **RESOLVED (2026-06-01)** | npm to registry.npmjs.org failed with `UNABLE_TO_VERIFY_LEAF_SIGNATURE` (corporate root CA). Fix: run npm/node with **`NODE_OPTIONS=--use-system-ca`** (Node 24 trusts the Windows cert store) — `npm ping` → PONG. Use this env var for all npm operations in this environment. Go module proxy (`proxy.golang.org`) works without changes. |
| E4 | PostgreSQL build deps (bison, flex, readline, zlib, ICU, etc.) | **OPEN** | Needed to build the 18.4 fork; confirm once a C toolchain is available. |

---

## Resolution gates before each phase
- **Before Phase 1:** E1 (CTO primitives) for commitments/AEAD; E3 if C bindings are in-scope this phase.
- **Before Phase 2:** B7 (Kafka topics), B9 (image tag), B10 (Chronicle-current), E2 (Docker).
- **Before testnet/mainnet (Phase 7):** B10, D1–D4 surveyed against in-force miner policy.
