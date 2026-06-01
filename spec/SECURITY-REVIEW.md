# SECURITY-REVIEW.md — Phase 7

Date: 2026-06-01. Scope: the implemented vertical slices of Packages A–D on Teranode regtest + PG18.
This review is honest about what is enforced vs. what is a documented slice. It is **not** a substitute
for an independent audit before any testnet/mainnet use.

## 1. Non-negotiable on-chain constraints (enforced)
- **No P2SH, no OP_RETURN** (`SYS-CON-002/008`): every produced locking script is built by
  `bsvscript.BuildEnvelopeIf/Drop` and passes `AssertNativeSpendable`, which rejects `IsP2SH()` and any
  `OP_RETURN` opcode and requires a native P2PKH spend tail. Static source scan: `OP_RETURN`/`P2SH`
  appear only in (a) comments and (b) the rejection checks/tests — **never in a produced script**.
  Token, EDI, and stream outputs all route through this builder. Bare `OP_CHECKMULTISIG` custody is
  native (no P2SH wrapper).
- **Sighash** (`SYS-ENC-004`): all state-transition inputs are signed `SIGHASH_ALL|FORKID`
  (`p2pkh.Unlock(priv, &sighash.AllForkID)`), binding successor outputs; accepted by the regtest
  validator across Phases 2–7.
- **Spendable carriage**: data rides in an unexecuted `OP_FALSE OP_IF <data> OP_ENDIF` branch ahead of
  P2PKH, so every data output remains a spendable UTXO node (verified by spending envelope outputs in
  the EDI/token transitions).

## 2. Keystone integrity (enforced)
- ECDH-HMAC tags (`SYS-HMAC-*`) are recomputed from `M(c)` + keys on every cold-rebuild / lineage
  verification; **any tamper to a recorded value breaks its tag** (tested), and a missing/!reordered
  entry breaks the `prev_txid` hash chain. Cross-implementation (TS↔Go) vectors are byte-identical.

## 3. Resilience (Phase 7, `SYS-PG-007`)
- **Confirmation-depth gating**: accounting-final reads gate on a policy depth (`SYS-DECIDE-004`);
  `hardene2e` shows depth<required ⇒ not final, depth≥required ⇒ final. Tip height is read from the
  tip block (`getblockchaininfo.blocks` lags on Teranode — do not use it for finality).
- **Reorg re-evaluation**: `invalidateblock` de-finalises the affected entry (divergence surfaced);
  `reconsiderblock` restores and re-confirms it.
- **Outbox idempotency**: capture is atomic with commit (`te.outbox`), the writer only processes
  `status='pending'` and records `(stream,seq)→txid` under a unique key, so a restart re-delivers
  **without double-recording or losing entries** (dedup by `M(c)`).

## 4. Confidentiality boundary (carried over, `SYS-CON-007`)
Forward access revocation is provable; **erasure of plaintext a party legitimately observed is NOT
provable** (CTO Statement B). The triple-entry chain guarantees integrity, ordering, and
non-repudiation of *recorded* entries — **not** the truthfulness of what a party chose to record
(`SYS-TE-005`). Stated to users.

## 5. Key custody
- Demo/regtest keys are fixed in code/DB for reproducibility — **NOT for production**. Production must
  hold writer master keys in threshold custody (`custody` package: Shamir loss-resistant shares
  encrypted under the ECDH common secret, ≥3 locations incl. backup; bare `OP_CHECKMULTISIG` N-of-M).
- `te.relationship.writer_priv` is stored in-DB in the slice; production must externalise to HSM /
  threshold custody.

## 6. Known limitations / documented slices (not yet production)
- C crypto binding deferred (TS↔Go parity proven; Appendix B.1 C-clause open).
- PL/pgSQL trigger capture (not in-core); single async writer; plaintext field values in the demo
  (confidential-commitment path + `te_render_pdf` not yet wired).
- DHT is a local store; SPV/BURI, CKD overlay, threshold-ECDSA-without-reconstruction, and the
  staked-compute→DFA live injection are unit-tested primitives, not yet fully wired end-to-end.
- EDI bridge covers the listed message types at envelope/mapping level, not full per-field semantics.
- **teratestnet/mainnet not attempted** — gated behind STOP-AND-ASK (live network, funds, miner-policy
  survey, Chronicle sighash validation). Move only after Appendix B passes end-to-end.

## 7. Grounding integrity (`SYS-INTEG-001`, Appendix B.12)
No requirement implemented here is grounded in a patent **abstract**. Each cited mechanism traces to
claims/description read in full (EP3860037A1, EP3748903A1, US20220253835A1, US11210372,
GB2558485A/US10579779B2, WO2022100946A1, WO2022214264A1, US11671255, EP3259724B1, EP4046048B1,
US20240364498A1) per Spec §0.2. Patent content is used for **method**, re-expressed in native BSV; no
fabricated claim values were introduced. Freedom-to-operate/licensing are out of scope (`SYS-INTEG-004`).
