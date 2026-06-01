# services-go — indexer, relay, broadcaster (Go)

Go service layer that glues the forked database to **Teranode** (`SYS-NODE-002`): transaction
submission, chain/UTXO/asset queries, and the new-block/new-tx event/notification stream. Hosts the
**Go** side of cross-implementation crypto parity (`crypto-core`, `SYS-TEST-003`).

## Responsibilities
- **Broadcaster** — builds (with `pg-fork/bsv-native`), signs (`SIGHASH_ALL|FORKID`, `SYS-ENC-004`),
  and broadcasts hash-chained ECDH-HMAC third-entry transactions; at-least-once delivery with
  idempotent dedup by `M(c)` (`SYS-PG-003`).
- **Indexer** — maintains the `(table_id,row_id,column_id,seq) → txid` index (`SYS-HMAC-006`),
  rebuildable purely from the chain; may back the Merkle-proof service entity (`SYS-PROOF-002`).
- **Relay** — consumes Teranode events; drives reorg re-evaluation and confirmation-depth gating
  (`SYS-PG-007`).

## Note (`SYS-NODE-002`)
Teranode is a microservices cluster, **not** a single `bitcoind`. The exact RPC methods, service
endpoints, and event-topic names are taken from current Teranode docs at build time and pinned in
`spec/VERIFY-LOG.md` — they are not assumed in code.
