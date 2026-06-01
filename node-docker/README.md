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

## Status
**BLOCKED (Phase 0 exit):** Docker is not installed in the current environment, so "regtest comes
up and produces blocks on demand" cannot yet be confirmed. Tracked in `spec/STATUS.md`.
