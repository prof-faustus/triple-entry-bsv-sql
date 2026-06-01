# node-docker — Dockerised Teranode (BSV node)

Dockerised full BSV node for development and testing. **Decision `SYS-DECIDE-010` (locked):**
the node is **Teranode** (Go microservices, horizontally scalable), shipped via
`teranode-quickstart`, run on **regtest** for dev/test (`SYS-NODE-001`).

## Requirements served
- `SYS-NODE-001` — Docker composition: Teranode + forked PostgreSQL + service layer.
- `SYS-NODE-002` — services talk to Teranode over its RPC + event/notification interfaces.
  Teranode is a **microservices cluster, not a single `bitcoind`** — RPC method names, service
  endpoints, and event-topic names **MUST** be taken from current Teranode docs at build time,
  not assumed (see `spec/VERIFY-LOG.md`).
- `SYS-NODE-003` — regtest: instant block generation, funded coinbase for fees, full teardown/rebuild.
- `SYS-CON-006` — end-to-end on regtest before any testnet/mainnet use.

## Constraints
- Chronicle upgrade is mandatory (mainnet 7 April 2026); the Teranode build **MUST** be Chronicle-current.
- Image tags/minor versions are re-confirmed at build time and pinned in `spec/VERIFY-LOG.md`.
- SV Node (`bitcoinsv/bitcoin-sv`, C++ monolith) is the non-chosen alternative — reference only.

## Pinned setup (resolved 2026-06-01 — see `spec/VERIFY-LOG.md` B1–B10)
- **Images:** `ghcr.io/bsv-blockchain/teranode:v0.15.1` (image id `044a2c5e8b6a`), `postgres:17`,
  `redpandadata/redpanda:v25.2.1`, `ghcr.io/bsv-blockchain/aerospike-server:8.1.2.0-1`.
- **RPC:** JSON-RPC, HTTP Basic Auth, `http://localhost:9292`. Methods we use: `getblockchaininfo`,
  `getblockbyheight`, `getblock`, `getbestblockhash`, `getrawtransaction`, `sendrawtransaction`,
  `generate`/`generatetoaddress` (regtest mining), `invalidateblock`/`reconsiderblock` (reorg tests).
  Note: `getblockcount` is **not** implemented — use `getblockchaininfo.blocks`.
- **Events:** Kafka/Redpanda `:9092`; topics `blocks-final` (new-block), `txmeta` (new-tx),
  `invalid-blocks`, `invalid-subtrees`, `rejectedtx`.
- **Chronicle:** v0.15.1 is post-Chronicle (mainnet activation 7 Apr 2026) → Chronicle-current.

## How it runs in this environment (no Windows admin)
Docker Desktop isn't installed and the host has no admin/UAC, so the node runs on **Docker Engine
inside WSL2 Ubuntu** (`VERIFY-LOG.md` E2). All commands run as `wsl -u root -- bash <script>`; the
Windows repo path `D:\claude\SQL` is `/mnt/d/claude/SQL` in WSL.

```
wsl -u root -- bash /mnt/d/claude/SQL/node-docker/regtest-up.sh        # clone+config+up+RUNNING
wsl -u root -- bash /mnt/d/claude/SQL/node-docker/rpc.sh generate 1    # mine a block
wsl -u root -- bash /mnt/d/claude/SQL/node-docker/lib/smoke-regtest.sh # SYS-NODE-003 smoke test
```
The quickstart repo itself is cloned to `/root/teranode-quickstart` (outside the git repo); `node-docker/`
holds the pinned config + thin helper scripts that drive it. `lib/set-regtest-creds.sh` and
`lib/inspect.sh` are one-shot maintenance/verification helpers.

## Status
✅ **Regtest confirmed up and producing blocks on demand** (FSM RUNNING; all core services healthy;
`generate` mines blocks; `getblock` retrieves them; `chain="regtest"`). Satisfies `SYS-NODE-001/003`
and the Phase-0 regtest exit (`SYS-CON-006`). Lean profile = core services only (no monitoring/p2p/legacy).
