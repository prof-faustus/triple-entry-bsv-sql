# tests — regtest e2e, KAT, adversarial, cold-rebuild

All tests run **end-to-end against the Dockerised Teranode node on regtest before testnet/mainnet**
(`SYS-TEST-001`, `SYS-CON-006`).

## Suite (`SYS-TEST-002`)
- ECDH-HMAC **known-answer vectors** (derive `CS`, `K_hmac`, `tag` from fixed inputs).
- **Hash-chain integrity** (tamper → tag mismatch; drop/reorder → chain break).
- **Cold-rebuild** (reconstruct the entire DB from chain + master keys; assert byte-equality).
- **SQL-surface** (ordinary SQL produces correct on-chain entries with no user blockchain code).
- **Reorg** tests.
- **Tokenisation lifecycle** (mint/transfer/redeem per cash profile and goods).
- **EDI DFA lifecycle** per document.
- **PDF paper-copy rendering** (deterministic byte-stable; embedded object_id/state/BURI/Merkle proof
  match the chain; SPV-verifiable from headers alone; B/L copy marked non-negotiable).
- The **CTO substrate's own adversarial suite**.

## Cross-implementation parity (`SYS-TEST-003`)
Shared vectors between C (fork) and TS/Go for ECDH, HKDF, HMAC, commitment, and encoding — divergence
is a **release blocker**. Vectors are sourced from `crypto-core/vectors/`.

## Definition of done
See Spec Appendix B (12 acceptance items) and `spec/STATUS.md` for current pass/fail per requirement ID.
