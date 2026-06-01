# pg-fork/bsv-native — key derivation, ECDH-HMAC, tx build/broadcast, hash chain

The native BSV engine inside the fork (C), consuming `crypto-core`. Implements the keystone
(`SYS-HMAC-001..011`): per-change `M(c) → GV → sub-keys → CS → K_hmac → tag(c)`; places `tag(c)` as
pushdata in a **spendable** locking script (`SYS-HMAC-005`, `SYS-ENC-001`); includes `prev_txid` to
form the hash chain (`SYS-HMAC-008`); signs with `SIGHASH_ALL|FORKID` binding successor outputs
(`SYS-ENC-004`). **No OP_RETURN, no P2SH** (`SYS-CON-008`, `SYS-CON-002`). Maintains the
`(table,row,column,seq) → txid` index (`SYS-HMAC-006`) and supports value extraction + cold-rebuild
(`SYS-PG-004`).
