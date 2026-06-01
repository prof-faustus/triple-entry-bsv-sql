# pg-fork/capture — committed-write interception

Captures every accounting-relevant committed change **exactly once, in commit order**, with the
table/row/column identity needed to form `M(c)` (`SYS-PG-002`). Writes to a **transactional outbox**
within the same DB transaction; the BSV tx is built/broadcast asynchronously (default) or the commit
blocks on chain acceptance (synchronous), selectable per deployment (`SYS-PG-003`, `SYS-DECIDE-003`).

Interception mechanism — **`SYS-DECIDE-002`** (open): (a) logical decoding / WAL output plugin,
(b) native C extension with commit hooks, or (c) deeper in-core commit-path hooks. See
`spec/DECISIONS.md` for the proposed choice and rationale.
