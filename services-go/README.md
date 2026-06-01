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

## Packages & Phase-2 e2e
- `node/` — minimal Teranode JSON-RPC client (`generate`/`generatetoaddress`, `getblock`,
  `getrawtransaction`, `sendrawtransaction`, `getblockchaininfo`).
- `bsvscript/` — spendable data-envelope builder + static no-P2SH/no-OP_RETURN check.
- `cmd/fundprobe/` — verifies regtest funding (coinbase → controlled key).
- `cmd/streame2e/` — the Phase-2 exit: fund → broadcast a hash-chained ECDH-HMAC stream in spendable
  envelopes (`SIGHASH_ALL|FORKID`) → discover → cold-rebuild from chain + keys. **Passes on the live
  Teranode regtest node** (Appendix B.2/B.5/B.6).

Run the e2e (the node runs in WSL; run the Go there too to avoid a Windows-exe launch stall):
```
wsl -u root -- bash /mnt/d/claude/SQL/node-docker/lib/run-e2e-wsl.sh   # logs to services-go/bin/e2e-wsl.out
```

## Note (`SYS-NODE-002`)
Teranode is a microservices cluster, **not** a single `bitcoind`. The exact RPC methods, service
endpoints, and event-topic names are taken from current Teranode docs at build time and pinned in
`spec/VERIFY-LOG.md` — they are not assumed in code.
