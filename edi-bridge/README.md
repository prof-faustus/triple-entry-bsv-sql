# edi-bridge — optional X12 / EDIFACT ↔ on-chain DFA translation

**Decision `SYS-DECIDE-005` (locked): the standards bridge IS provided**, as a configurable,
**optional**, per-trading-partner component (`SYS-EDI-005`). It is a **pure translation layer**
(`SYS-EDI-006`): the on-chain DFA remains the source of truth; the bridge holds no authority the
chain does not confirm; it **MUST be omittable without affecting the core DFA engine**.

## Mappings
- **ANSI X12:** 850 (PO), 855 (PO ack), 856 (ASN/despatch), 810 (invoice), 820 (payment/remittance),
  214 (transport status), 990/210 (freight).
- **UN/EDIFACT:** ORDERS, ORDRSP, DESADV, INVOIC, REMADV, IFTMIN (transport instruction),
  IFTSTA (transport status), PAYORD.

## Behaviour
- **Inbound:** parse a standard message → validate → drive the corresponding `edi-dfa` transition
  (journalled as a third entry).
- **Outbound:** serialise current on-chain document state into the standard message for a partner that
  transacts in X12/EDIFACT.
- Enabled/disabled and configured **per partner** (which standard, version, message subset).

## Exit criteria (Phase 5 / Appendix B.8)
The bridge translates the listed message types in and out and can be disabled without affecting the core.
