# CTO-PRIMITIVES.md — CTO substrate primitives (RECONCILED 2026-06-01)

> **Status: reconciled against the authoritative spec.** The CTO spec was initially believed missing
> and these primitives were re-derived; it was then located at `D:\claude\cto\spec\source\
> CTO_BSV_Build_Spec_v1.md` and this document has been **reconciled against it**. Outcome:
> - **Commitment — ALIGNED (was divergent):** CTO §6/Step T4 specifies `SHA-256("CTO/commit/v1" ‖ r ‖
>   value)` (raw concat, 32-byte `r`, on-chain-openable via `OP_CAT`). The implementations were changed
>   from a length-prefixed `TE/commit/v1` form to this exact CTO format; C/TS/Go vectors regenerated, all
>   parity tests green.
> - **HKDF / AEAD / ECDH-octet-form — consistent:** the CTO spec mandates HKDF-SHA256 (RFC 5869),
>   AES-256-GCM **or** ChaCha20-Poly1305 (we use AES-256-GCM, an allowed default), and leaves exact HKDF
>   salt/info placement and the ECDH-secret octet form as its own `[VERIFY]` items — our choices fall
>   within them.
> - **Layering:** the TE per-change keystone uses **EP3860037A1** (additive sub-keys), which is a
>   *different* mechanism from the CTO per-transfer forward secret (`OTS = ECDH(ephemeral, recipient)`);
>   both coexist. TE is **stricter** than CTO on carriage: CTO permits `OP_RETURN`, TE forbids it
>   (`SYS-CON-008`) — TE always uses spendable envelopes.
> - **Tiers:** Tier S (threshold custody) ↔ `services-go/custody` (Shamir + bare multisig); Tier T (TEE)
>   not implemented.

The byte-exact definitions live in `ALGORITHMS.md`; this file gives the construction choices and why.

## CTO surface the Spec depends on (`SYS-SUB-001`)
> "secp256k1 ECDH, HKDF, AEAD, SHA-256 commitments, UTXO-lineage objects, time-locked recovery" plus
> the confidentiality honesty boundary (`SYS-CON-007`) and Tier F/S/T deletion/custody profiles.

## A. secp256k1 ECDH common secret  *(grounded, not assumed: EP3860037A1)*
The common-secret derivation is taken **directly from EP3860037A1** (claims 1–5), so it is not a
re-derivation: `GV = SHA-256(M)`, sub-keys `v2 = v1 + GV mod n` / `P2 = P1 + GV·G`, and
`CS = v2_W·P2_C = v2_C·P2_W`. See `ALGORITHMS.md` §2.
- **Assumption A1 (octet form of CS):** the common secret is an EC *point*; we serialise it as the SEC1
  **compressed** encoding (33 bytes) and use that as the HKDF IKM. This is standard practice for turning
  an ECDH point into key material and keeps the secret fixed-length. *Reviewer check: confirm the CTO
  spec doesn't instead specify `SHA-256(compressed(point))` or the raw X-coordinate as the secret.*

## B. HKDF (RFC 5869, SHA-256)  *(standard construction)*
Used to derive `K_hmac` (and `K_aead`) from `CS`, never using `CS` as a key directly (`SYS-HMAC-003`).
- **Assumption B1 (domain parameter):** `SYS-HMAC-003` lists `domain`, `ikm`, `salt`, `info` but HKDF
  has only `salt`, `ikm`, `info`, `L`. We realise `domain` as a **prefix of `info`** (`utf8("TE/hmac/v1")
  ‖ u64(seq)`), giving the required domain separation. *Reviewer check: acceptable vs folding `domain`
  into `salt`.*

## C. CTO blinded commitment  *(RECONCILED to CTO §6/T4)*
`commit(value, r) = SHA-256("CTO/commit/v1" ‖ r ‖ value)` — **raw** concat, `r` = 32-byte random
blinding (`ALGORITHMS.md` §4). Matches `CTO_BSV_Build_Spec_v1` §6 / Step T4 exactly, so a TE field
commitment is byte-identical to a CTO commitment and openable on-chain via `OP_CAT` (CTO-SCRIPT-004).
- **C1 (resolved):** the CTO spec confirms a **hash commitment** with mandatory 32-byte blinding (not
  Pedersen/EC). The earlier length-prefixed `TE/commit/v1` form was changed to this. Not additively
  homomorphic; the EC commitment of `SYS-PROOF-004` is a separate optional primitive.

## D. AEAD = AES-256-GCM  *(re-derived; `SYS-SUB-001` "AEAD")*
Confidential payloads (private ledger fields, document/goods bodies) are encrypted under a key derived
from `CS` (`ALGORITHMS.md` §5); for goods records this realises `SYS-LOG-004` ("encrypted under the
common secret"). 96-bit nonce, 128-bit tag, AAD = record header (binds ciphertext to its anchor).
- **Assumption D1 (cipher choice):** AES-256-GCM, available natively in Node and Go (no third-party dep)
  and in OpenSSL for the C fork. *Reviewer check: the CTO spec may mandate ChaCha20-Poly1305 or XChaCha;
  swapping is localised to §5.*
- **Assumption D2 (nonce):** deterministic per-record nonce derived from the record header to avoid reuse;
  exact derivation pinned when the encrypt-then-anchor path is implemented (Phase 3/logistics). *Reviewer
  check.*

## E. UTXO-lineage objects  *(structural; implemented Phase 2+)*
An object/token/document/stream is a single-threaded UTXO lineage: spend current state → create successor
bound to the new controller; no residual prior-controller path except a time-locked recovery branch
(`SYS-TOK-004`, `SYS-EDI-001`). Native BSV only — no P2SH, no OP_RETURN. Not a Phase-1 crypto-core item;
recorded here for completeness.

## F. Time-locked recovery  *(script-level; implemented Phase 2+)*
A recovery branch using `OP_CHECKLOCKTIMEVERIFY`/`OP_CHECKSEQUENCEVERIFY` (`SYS-ENC-003`) lets a
designated key reclaim an object after a timeout. Script-level, not crypto-core; pinned when scripts are
built (Phase 2).

## G. Confidentiality honesty boundary  *(carried over verbatim, `SYS-CON-007`)*
Forward access revocation is provable; **erasure of plaintext a party legitimately observed is NOT
provable** (CTO Statement B). No package may claim otherwise (`SYS-HMAC-011`, `SYS-TE-005`).

## H. Tier F/S/T profiles  *(deferred to Phase 6 custody)*
F/S/T are the CTO deletion/custody profiles. Tier-S (threshold custody) maps to `SYS-CUST-001..003`
(US11671255 / EP3259724B1). Re-derivation of F/T deletion semantics is deferred until needed and will be
appended here. *Reviewer check: F/S/T exact semantics are the most likely place the missing CTO spec
matters; treat as OPEN until reconciled.*

---

### Review checklist (resolve if/when `CTO_BSV_Build_Spec_v1.md` is supplied)
- [ ] A1 octet form of the common secret
- [ ] B1 HKDF domain placement
- [ ] C1 commitment is a hash commitment (not EC/Pedersen) for field values
- [ ] D1/D2 AEAD cipher + nonce derivation
- [ ] H Tier F/S/T semantics
