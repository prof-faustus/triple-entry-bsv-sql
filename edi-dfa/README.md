# edi-dfa — commercial-document DFA engine

Grounded in **US20220253835A1** (read in full), native BSV. Each commercial-document lifecycle is a
**deterministic finite automaton whose states are UTXOs**; a transition spends the current state UTXO
and creates the next (`SYS-EDI-001`). Every transition is journalled as a third entry (`SYS-EDI-003`).

## Mandatory document set (`SYS-EDI-002`)
- **Trade/payment:** RFQ & quotation; PO & order acknowledgement/change; commercial invoice;
  payment note (links to cash, Section 8.1); credit/debit note.
- **Logistics/transport:** despatch advice/ASN (DESADV); **bill of lading** (title document — the
  consignment token *is* the negotiable B/L, see `logistics/`); sea/air waybill; CMR; rail consignment
  note; packing list; booking confirmation/transport instruction (IFTMIN-class); arrival notice; POD.
- **Regulatory/assurance:** certificate of origin; customs/import-export declaration;
  inspection/quality/condition certificate; insurance certificate.

Each DFA needs an explicit state set, event alphabet, and transition table (per the worked method:
state set, input alphabet, transition matrix, origination + completion transactions, bounded cost).

## Requirements
- `SYS-EDI-003` — discoverability: derive a search key from state metadata, scan for matching UTXO,
  extract, determine current state; the Postgres index (`SYS-HMAC-006`) is the materialised search.
- `SYS-EDI-004` — cross-reference by `object_id` (invoice→PO+consignment; payment note→invoice;
  POD→B/L); reference graph reconstructable from chain alone.

## Exit criteria (Phase 5 / Appendix B.8)
Every document type runs its DFA lifecycle.
