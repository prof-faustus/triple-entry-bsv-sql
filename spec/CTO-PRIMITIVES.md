# CTO-PRIMITIVES.md — re-derived CTO substrate primitives (FOR REVIEW)

> **Status: re-derivation, pending review.** The Spec (line 15, `SYS-SUB-001`, `SYS-CON-007`) builds on
> the CTO confidential-object primitive defined in `CTO_BSV_Build_Spec_v1.md`, which is **not present in
> this repository**. Per the operator decision of 2026-06-01, this document **re-derives the CTO
> primitives the build needs from standard constructions and EP3860037A1**, states every assumption
> explicitly, and flags them for review. **If the authoritative `CTO_BSV_Build_Spec_v1.md` is later
> supplied, every item here MUST be reconciled against it; any divergence is a `[VERIFY]` gate**
> (`VERIFY-LOG.md` E1). Nothing here is asserted as the CTO spec's actual content — these are our
> documented, reviewable stand-ins.

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

## C. SHA-256 blinded commitment  *(re-derived; `SYS-SUB-001` "SHA-256 commitments")*
`commit(value, r) = SHA-256("TE/commit/v1" ‖ len-prefixed r ‖ len-prefixed value)` with 32-byte random
`r` (`ALGORITHMS.md` §4).
- **Assumption C1:** a **hash commitment** (binding via collision-resistance, hiding via random `r`) is
  what `SYS-SUB-001`'s "SHA-256 commitments" means. It is **not additively homomorphic**; the EC
  homomorphic commitment of `SYS-PROOF-004` is a *separate* optional primitive. *Reviewer check: confirm
  the CTO spec's commitment is a hash commitment and not Pedersen/EC for the field-value use.*

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
