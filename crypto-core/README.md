# crypto-core — shared cryptographic primitives + KAT vectors

The shared cryptographic core, implemented once and bound into **C** (PostgreSQL fork) and
**TypeScript/Go** (services/SDK), with **cross-implementation known-answer vectors that MUST
match exactly** — divergence is a release blocker (`SYS-TEST-003`).

## Primitives (Phase 1)
- **ECDH common-secret** by the EP3860037A1 method (`SYS-HMAC-002`): from a public message `M`,
  `GV = SHA-256(M)`; derive sub-keys `V2 = V1 + GV mod n`, `P2 = P1 + GV·G`; compute
  `CS = V2_writer · P2_counterparty = V2_counterparty · P2_writer` (secp256k1).
- **HKDF** (`SYS-HMAC-003`): `K_hmac = HKDF(domain="TE/hmac/v1", ikm=CS, salt=table||row||column, info=seq)`.
  `CS` is never used directly as the HMAC key (domain separation).
- **HMAC-SHA256** (`SYS-HMAC-004`): `tag(c) = HMAC-SHA256(K_hmac, canonical(change_image))`.
- **Commitments** (`SYS-HMAC-009`): blinded CTO commitment to a confidential field value.
- **Canonical length-prefixed encoder/decoder** (`SYS-ENC-005`): magic, version, object/stream id,
  sequence, prev-hash, ECDH-HMAC tag, payload-or-hash. Round-trip + rejection tests.

## Exit criteria (Appendix B.1)
KAT vectors green and **byte-identical** across the C fork and the TS/Go components.

## Layout
- `vectors/` — shared known-answer test vectors (the single source of truth for cross-impl parity).

## Implementations & how to run
- `ts/` — TypeScript (Node built-in `crypto` for SHA-256/HMAC/HKDF/AES-GCM; `@noble/curves` for secp256k1).
  `cd ts && NODE_OPTIONS=--use-system-ca npm install && npm test` (the env var trusts the Windows cert
  store — see `VERIFY-LOG.md` E5). `npm run gen-vectors` regenerates the shared vectors (TS is the source
  of truth).
- `go/` — Go (`crypto/*` std lib; `decred/dcrd/dcrec/secp256k1/v4`; `x/crypto/hkdf`).
  `cd go && go test ./...` — consumes the same `vectors/*.json` and must match byte-for-byte.
- `vectors/` — `core_vectors.json` (composite KAT, TS-generated) + `rfc_vectors.json` (RFC-4231 / RFC-5869
  / NIST standard KATs validating the primitive wiring independently in every impl).

**Status (Phase 1):** TS and Go pass and agree byte-for-byte (`SYS-TEST-003`). The **C** binding is
deferred (no toolchain yet, `VERIFY-LOG.md` E3), so Appendix B.1's C-parity clause stays open.

## Dependency / status
`SYS-SUB-001` grounds the confidential primitives on the **CTO substrate**
(`CTO_BSV_Build_Spec_v1.md`) — secp256k1 ECDH, HKDF, AEAD, SHA-256 commitments. That spec is
**not present in this repo** and must be supplied or its relevant primitives re-derived; see
`spec/STATUS.md`. The C bindings additionally need a C toolchain (not yet present).
