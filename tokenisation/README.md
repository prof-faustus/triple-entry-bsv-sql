# tokenisation — definable token primitive (any item or money unit)

Grounded in **EP3748903A1** (Universal Tokenisation System, read in full), re-expressed in native
BSV (no P2SH, never OP_RETURN). One general primitive represents **any item or any unit of value**;
cash, CBDC/stablecoin-linked tokens, and goods are all instances of it.

## Requirements
- `SYS-TOK-001` — token = metadata + ≥1 public key in a **spendable** locking script (pushdata via
  `SYS-ENC-001`), not a P2SH redeem script, never OP_RETURN.
- `SYS-TOK-002` — entity storable on/off chain (forked Postgres and/or DHT).
- `SYS-TOK-003` — value binding `satoshi_qty = f(token_value, pegging_rate)` with minimum threshold;
  N-of-M auth via bare `OP_CHECKMULTISIG` (native).
- `SYS-TOK-004` — transfer = UTXO-lineage transfer; no residual prior-controller path except time-locked recovery.
- `SYS-TOK-005` — **definable token schema** (type id, label, unit, divisibility, supply policy,
  backing/pegging, controller model, confidentiality) — new token types with **no code change**.
- `SYS-TOK-006` — **external linkage**: peg/back to an external unit (token-coin, stablecoin, **CBDC**)
  via pegging-rate + named oracle + named custodian; or **bridge** via an adapter contract. The
  external rail's own interface is **out of scope here beyond the adapter contract**; integrating a
  real rail is gated behind STOP-AND-ASK.
- `SYS-TOK-007` — atomic two-token swap (deliver-versus-deliver), each leg journalled as a third entry.
- `SYS-CASH-001/002` — cash profiles: issuer-backed redeemable / satoshi-tagged / pegged (`SYS-DECIDE-001`),
  all mint/transfer/redeem journalled as third entries.
- `SYS-GOODS-001` — goods token: on-chain control/transfer, off-chain detailed record (US11210372).

## Exit criteria (Phase 4 / Appendix B.7)
A new token type defined via schema (no code change); cash (3 profiles), a CBDC/stablecoin-linked token
(adapter contract only), and a goods token all mint/transfer/redeem and journal; any two instances swap atomically.
