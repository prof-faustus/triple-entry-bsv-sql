# sdk-ts — client SDK (TypeScript)

TypeScript client SDK. Hosts the **TS** side of the cross-implementation crypto parity
(`crypto-core`, `SYS-TEST-003`) and the developer-facing client surface consistent with the
SQL-simple principle (`SYS-CON-005`, `SYS-PG-005`) — application code uses ordinary SQL; the SDK
provides typed helpers for key registration, verification (`te_verify`), proof export
(`SYS-PROOF-005`), and PDF rendering (`te_render_pdf`).

The shared crypto/encoding vectors live in `crypto-core/vectors/` and MUST match the C and Go
implementations exactly.
