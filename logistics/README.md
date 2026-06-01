# logistics — consignment lifecycle + DHT goods records

Grounded in **US11210372** (goods records + ownership), **EP3748903A1** (token),
**US20220253835A1** (lifecycle), **EP3860037A1** (confidentiality), and **GB2558485A / US10579779B2**
(integrity) — all re-expressed natively (no P2SH, no redeem script).

## Requirements
- `SYS-LOG-001` — consignment = goods token whose lifecycle is a DFA:
  `CREATED → BOOKED → PICKED_UP → IN_TRANSIT(leg_k) → CUSTOMS_HELD/CLEARED → INSPECTED →
  CUSTODY_TRANSFER(party_k) → DELIVERED → ACCEPTED/REJECTED → SETTLED`, plus `DISPUTED` and
  time-locked `RECOVERED` branches.
- `SYS-LOG-002/003` — goods record in a **DHT** (not on chain); on-chain metadata carries `H2`
  (DHT key) + owner/custodian key; DHT value = record data `D1` + body hash `H1` + location; record is
  header+body, header carrying the body hash.
- `SYS-LOG-004/005` — **ownership/custody**: derive second public keys + `GV`, compute `CS` (ECDH),
  verify control by matching on-chain controller key vs DHT-registered owner key (`PU2` vs `P2`); record
  encrypted under `CS`. Any authorised party verifies from chain + DHT, no intermediary.
- `SYS-LOG-006` — **bill of lading as the token**: transferring the token *is* transfer of title;
  paper copy via `doc-render/` does not itself transfer title (`SYS-DOC-004`).
- `SYS-LOG-007` — multi-party, multi-leg custody as successive `CUSTODY_TRANSFER` transitions; full
  chain of custody reconstructable from chain + DHT.
- `SYS-LOG-008` — delivery-versus-payment: `ACCEPTED` links to cash-token settlement + invoice/payment note.
- `SYS-LOG-009/010/011/012` — **integrity** (GB2558485A): `H3 == H4` over header+body; integrity and
  ownership are distinct, complementary checks, both exposed; integrity backs the PDF copy.

## Exit criteria (Phase 5 / Appendix B.9)
Consignment lifecycle, multi-party custody, B/L-as-token title transfer, ownership (key-match) and
integrity (hash-match) checks, and delivery-versus-payment all pass.
