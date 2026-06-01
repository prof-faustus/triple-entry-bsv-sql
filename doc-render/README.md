# doc-render — deterministic PDF paper copies (+ BURI/QR)

Renders any commercial/logistics document — and the **bill of lading** in particular — to a complete,
human-readable **PDF paper copy**, assembled deterministically from (i) on-chain DFA state + lineage,
(ii) the off-chain detail record (forked Postgres and/or DHT), and (iii) the SPV inclusion proof
(BURI + Merkle proof) (`SYS-DOC-001`).

## Requirements
- `SYS-DOC-002` — embed verifiability: `object_id`, current-state tx as a **BURI**, and the Merkle/SPV
  proof, as text **and** a scannable QR of the BURI, so a holder can verify against block headers
  without trusting the issuer or DB operator.
- `SYS-DOC-003` — **deterministic & byte-stable**: fixed templates, fixed field order, no nondeterministic
  timestamps in the body; tests assert embedded `object_id`/state/proof match the chain.
- `SYS-DOC-004` — **negotiability honesty**: for a negotiable B/L the on-chain token is the single source
  of title; a PDF is a *verifiable representation, not a second negotiable original*, and MUST be marked
  "COPY — title held on-chain…". A controlled single-paper-original workflow, if ever offered
  (`SYS-DECIDE-009`, default: not offered), is an explicit `PAPER_ISSUED` token-locking transition.
- `SYS-DOC-005` — invocable from SQL: `te_render_pdf(object_id)`.
- `SYS-LOG-012` — the integrity check (GB2558485A hash-match) backs the paper copy: a verifier recomputes
  `H3` over the rendered body and confirms against the on-chain-anchored `H4`.

Uses the `pdf` skill conventions. Confidential fields are included only when the renderer holds the
entitlement keys (CTO confidentiality rules).

## Exit criteria (Phase 5 / Appendix B.3)
`te_render_pdf()` emits a deterministic, SPV-verifiable PDF; B/L copy marked non-negotiable.
