# pg-fork/sql-surface — DDL extensions, catalog, te_verify(), te_render_pdf()

The SQL-simple surface (`SYS-CON-005`, `SYS-PG-005`): configuration as SQL DDL extensions / catalog
tables (a `triple_entry` table property; per-relationship counterparty/auditor key registration; stream
layout; cash/token bindings). Verification is SQL-callable: `te_verify(table,row)` returns whether the
on-chain tag matches the current value and the entry's hash-chain position (`SYS-PG-006`), and exports a
self-contained SPV proof (BURI + Merkle proof) verifiable from block headers alone (`SYS-PROOF-005`).
`te_render_pdf(object_id)` emits the deterministic PDF paper copy (`SYS-DOC-005`).
