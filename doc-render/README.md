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

## Status — implemented (2026-06-01)
`services-go/docrender` renders a **deterministic, byte-stable** single-page PDF embedding `object_id`,
state, controller, cross-refs, the **BURI**, and the body integrity hash (`H4`); for a `bill_of_lading`
it stamps the **non-negotiable** marking (`SYS-DOC-002/003/004`, `SYS-LOG-012`). Tests assert
byte-stability, embedded fields, the B/L marking (and its absence on other docs), and that the body
hash binds the state. **Remaining slices:** the scannable **QR** of the BURI (`SYS-DOC-002`, needs a QR
lib + image embed) and the thin **SQL `te_render_pdf()` wrapper** (`SYS-DOC-005`) sourcing fields from
the chain/`te` schema and returning the `docrender` bytes.
