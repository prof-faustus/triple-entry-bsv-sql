# DECISIONS.md — `SYS-DECIDE-*` register

Per `SYS-INTEG-003`, **no deployment decision is made silently**. This file records every
`SYS-DECIDE-*`. Decisions marked **LOCKED** are fixed by the Spec (Section 14). Decisions marked
**PROPOSED** are recommendations with rationale, recorded here per the kickoff brief but **pending
operator confirmation at the Phase-3 checkpoint** before any code irreversibly depends on them; none
is treated as silently assumed. The "bites in" column is the earliest phase whose work depends on it.

Last updated: 2026-06-01.

## Locked (Spec Section 14)

| ID | Decision | Value |
|---|---|---|
| `SYS-DECIDE-001` | Cash / token model | **Definable token** (`SYS-TOK-005`) for any item/money unit; all three cash profiles provided & selectable; external linkage to token-coins/stablecoins/**CBDC** (`SYS-TOK-006`). |
| `SYS-DECIDE-005` | EDI standards bridge | **Provided**, configurable, optional, per-partner (X12 850/855/856/810/820/214/990/210; EDIFACT ORDERS/ORDRSP/DESADV/INVOIC/REMADV/IFTMIN/IFTSTA/PAYORD). Omittable without affecting core DFA. |
| `SYS-DECIDE-008` | PostgreSQL version | **PostgreSQL 18** (latest major; PostgreSQL License, fork-permissive). Exact 18.x minor pinned in `VERIFY-LOG.md`. |
| `SYS-DECIDE-010` | BSV node | **Teranode** (Go microservices) via `teranode-quickstart` Docker; regtest for dev/test; Chronicle-current. |

## Open — PROPOSED (pending operator confirmation)

### `SYS-DECIDE-002` — Write-interception mechanism  *(bites: Phase 3)*
**Options:** (a) logical decoding / WAL output plugin; (b) native C extension with commit hooks;
(c) deeper in-core commit-path hooks.
**Proposed:** **(a) logical-decoding output plugin as the authoritative commit-ordered capture**, with
the transactional outbox (`SYS-PG-003`) populated atomically with the commit, augmented by a thin
in-core hook only where logical decoding cannot supply the table/row/**column** identity `M(c)` needs
(use `REPLICA IDENTITY FULL` for old/new column values).
**Rationale:** logical decoding gives committed changes **in commit order, exactly once**, via a stable
documented API, and is the **least invasive to maintain against PostgreSQL 18 upstream** — which matters
because this is a long-lived fork. Pure in-core hooks (c) maximise control but maximise merge-maintenance
cost and risk; trigger-only capture (b) gives weaker commit-ordering/column-identity guarantees.
**Open question for operator:** is minimal-divergence-from-upstream (favours a) or deepest-control
(favours c) the priority for this fork?

### `SYS-DECIDE-003` — Journalling mode  *(bites: Phase 3)*
**Proposed (per Spec default):** **async outbox = default** (throughput; outbox guarantees no lost entry
with at-least-once delivery + idempotent dedup by `M(c)`); **synchronous commit-on-chain** offered and
selectable per deployment for use cases needing on-acceptance finality. No open conflict — confirm only.

### `SYS-DECIDE-004` — Confirmation depth for accounting finality  *(bites: Phase 3/7)*
**Proposed:** configurable **per use case**; defaults — regtest/dev = **1**; testnet/mainnet general =
**6 confirmations**; high-value settlement (cash redemption, B/L title transfer) = deeper, operator-set.
**Rationale:** depth trades latency against reorg risk; the right number is a risk/policy choice, so it
is parameterised, not hard-coded. **Operator to set production values** (a policy `[VERIFY]`, not a fixed
fact — see `VERIFY-LOG.md`).

### `SYS-DECIDE-006` — Stream (hash-chain) granularity  *(bites: Phase 2/3)*
**Options:** per-ledger vs per-account vs per-table.
**Proposed:** **per-ledger (per-relationship) stream as default** — one genesis-rooted hash chain per
ledger relationship, so both parties reconcile **one shared stream** (`SYS-TE-002`); optional per-table
sub-streams for high-throughput sharding, with the **union of streams reconstructing the whole ledger**
(`SYS-HMAC-008`). **Rationale:** matches the bilateral-reconciliation model directly; per-account is too
fine for cross-account transactions, per-table fragments a relationship.

### `SYS-DECIDE-007` — Counterparty/auditor key for single-party books  *(bites: Phase 3)*
**Proposed:** a per-relationship **counterparty registry**; when there is a trading partner, the
counterparty is the partner's key; for single-party books the ECDH counterparty defaults to a
deployment-level **designated auditor key** (as `SYS-HMAC-002` anticipates), overridable per relationship.
**Rationale:** the ECDH construction needs a second party; the auditor is the natural reconciling party
when no trading partner exists, and keeps the "bilateral verifiability" property (`SYS-HMAC-011`).

### `SYS-DECIDE-009` — Controlled single-paper-original B/L workflow  *(bites: Phase 5)*
**Proposed (per Spec default): NOT offered** — the on-chain token is the sole title; a rendered PDF is
copy-only and marked non-negotiable (`SYS-DOC-004`). The `PAPER_ISSUED` token-locking state remains
available behind an explicit, audited config flag for a counterparty that genuinely cannot transact on
chain. **Rationale:** avoids the double-title hazard by default; the locking transition is the only safe
way to permit a paper original, and only when truly required.

## Notes
- `SYS-DECIDE-002/004/006/007` materially shape the Phase-3 architecture and Phase-7 hardening; they will
  be surfaced for explicit confirmation at the **Phase-3 checkpoint** (or earlier if their dependent work
  starts sooner).
- `SYS-DECIDE-003/009` follow the Spec's stated defaults and need confirmation only.
