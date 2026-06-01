# ALGORITHMS.md — normative crypto & encoding definitions (Phase 1)

This file is the **byte-exact contract** for the shared crypto core (`crypto-core/`). The C, TypeScript,
and Go implementations MUST all produce identical bytes for identical inputs; the known-answer vectors in
`crypto-core/vectors/` are the enforcement (`SYS-TEST-003`, Appendix B.1). Grounded in **EP3860037A1**
(ECDH common-secret) and the re-derived CTO primitives in `CTO-PRIMITIVES.md` (`SYS-SUB-001`).

All multi-byte integers are **big-endian** unless stated. Curve is **secp256k1**; order
`n = 0xFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFFEBAAEDCE6AF48A03BBFD25E8CD0364141`; generator `G`.

---

## 1. Canonical byte encoding (`SYS-ENC-005`)

Primitive encoders (deterministic, unambiguous, length-prefixed):

| Encoder | Definition |
|---|---|
| `u8(x)` | 1 byte |
| `u16(x)`, `u32(x)`, `u64(x)` | 2/4/8 bytes, big-endian |
| `bytes(b)` | `u32(len(b)) ‖ b` — length-prefixed octet string (length is a 4-byte BE count) |
| `str(s)` | `bytes(utf8(s))` |

`‖` denotes concatenation. Decoding rejects: trailing bytes, truncated length, length exceeding the
remaining buffer, and (for `str`) invalid UTF-8.

### 1.1 The change Message `M(c)` (`SYS-HMAC-001`)

`M(c)` is the **public** identity of a SQL change (carries no secret; may travel insecure).

```
M(c) = MAGIC_M ‖ u8(VERSION) ‖ str(table_id) ‖ bytes(row_id)
            ‖ str(column_id) ‖ u8(op) ‖ u64(seq) ‖ bytes(prev_txid)
```

- `MAGIC_M` = ASCII `"TEMC"` (0x54 0x45 0x4D 0x43) — Triple-Entry Message Canonical.
- `VERSION` = `1`.
- `op`: `INSERT=1`, `UPDATE=2`, `DELETE=3`.
- `row_id`: caller-supplied canonical row-key octets (e.g. the primary-key bytes).
- `prev_txid`: 32-byte big-endian txid of the previous stream entry, or **empty** (length 0) at genesis.

### 1.2 The on-chain field record (`SYS-ENC-005`)

The canonical layout placed as pushdata inside the spendable locking script (`SYS-ENC-001`; never
OP_RETURN). This is what carries the tag + prev-reference on chain.

```
REC = MAGIC_R ‖ u8(VERSION) ‖ bytes(stream_id) ‖ u64(seq) ‖ bytes(prev_txid)
          ‖ u8(image_kind) ‖ bytes(change_image) ‖ bytes(tag)
```

- `MAGIC_R` = ASCII `"TER1"` — Triple-Entry Record v1.
- `stream_id`: the hash-chain/stream identifier (`SYS-HMAC-008`, `SYS-DECIDE-006`).
- `image_kind`: `0 = plaintext`, `1 = commitment` (see §4 / `SYS-HMAC-009`).
- `change_image`: per §3.4.
- `tag`: the 32-byte HMAC tag (§3.3).

Round-trip (`encodeRecord`/`decodeRecord`) and rejection tests are mandatory.

---

## 2. ECDH common secret (EP3860037A1, `SYS-HMAC-002`)

Given the change Message `M`, each party's master key pair `(v, P=v·G)`, writer `W` and counterparty `C`:

```
GV     = SHA-256(M(c))                      # 32 bytes, interpreted as a big-endian integer
v2_W   = (v_W + GV) mod n                    # derived sub-private key (writer)
P2_W   = P_W + GV·G   ( = v2_W·G )           # derived sub-public key  (writer)
v2_C   = (v_C + GV) mod n
P2_C   = P_C + GV·G   ( = v2_C·G )
CS_pt  = v2_W · P2_C  =  v2_C · P2_W          # the common-secret EC point (symmetric)
CS     = compressed(CS_pt)                    # 33 bytes: 0x02/0x03 ‖ X(32)  — the shared-secret octets
```

`compressed(point)` is the SEC1 compressed encoding. **Assumption (documented in `CTO-PRIMITIVES.md`):
the 33-byte compressed common-secret point is used directly as the HKDF IKM**; it is never used as a key
directly (`SYS-HMAC-003`). If `CS_pt` is the point at infinity (degenerate keys) the inputs are rejected.

---

## 3. Key derivation, tag, change image

### 3.1 HKDF-SHA256 (`SYS-HMAC-003`)
Standard RFC-5869 HKDF with SHA-256. `HKDF(salt, ikm, info, L) = Expand(Extract(salt, ikm), info, L)`.

### 3.2 HMAC key
```
salt   = table_id ‖ row_id ‖ column_id           # raw concatenation of the octet strings
info   = utf8("TE/hmac/v1") ‖ u64(seq)            # the domain label is realised as the HKDF info prefix
K_hmac = HKDF-SHA256(salt=salt, ikm=CS, info=info, L=32)
```
Folding the `domain` argument into `info` gives the domain separation `SYS-HMAC-003` requires while
using only the standard HKDF parameter set. (Documented assumption — `CTO-PRIMITIVES.md`.)

### 3.3 Tag (`SYS-HMAC-004`)
```
tag(c) = HMAC-SHA256(K_hmac, change_image)        # 32 bytes
```

### 3.4 Change image (`SYS-HMAC-004`, `SYS-HMAC-009`)
`change_image` is the committed representation of the changed value(s):
- **single field, plaintext:** `change_image = bytes(value)`; `image_kind = 0`.
- **single field, confidential:** `change_image = commit(value, r)` (§4, 32 bytes); `image_kind = 1`.
- **multiple fields:** `change_image = bytes(v1) ‖ bytes(v2) ‖ …` in column order (each element per the
  plaintext/commitment rule), wrapped so the whole is one octet string.

---

## 4. SHA-256 blinded commitment (`SYS-HMAC-009`, `SYS-SUB-001`)

```
commit(value, r) = SHA-256( utf8("TE/commit/v1") ‖ u32(len(r)) ‖ r ‖ u32(len(value)) ‖ value )
```
- `r` = 32-byte uniformly random blinding factor; stored with the plaintext in the confidential DB row.
- Binding: SHA-256 collision resistance. Hiding: random `r` (RO/PRF assumption).
- This is a **hash commitment** — *not* additively homomorphic. The EC/homomorphic commitment of
  `SYS-PROOF-004` (Merkle-node sums) is a separate, optional Phase-6 primitive and is **not** this one.

---

## 5. AEAD for confidential payloads (`SYS-SUB-001`, used by `SYS-LOG-004`)

```
K_aead = HKDF-SHA256(salt=context, ikm=CS, info=utf8("TE/aead/v1") ‖ context_seq, L=32)
ct     = AES-256-GCM(key=K_aead, nonce=N(96-bit), aad=record_header, plaintext)
```
- AES-256-GCM; 96-bit nonce; 128-bit tag. Nonce derivation and AAD = the record header (so the header
  binds the ciphertext). Full rationale and the encrypt-then-anchor flow are in `CTO-PRIMITIVES.md`.

---

## 6. Vectors (`crypto-core/vectors/`)
The vector files are the single source of truth. Each implementation MUST reproduce, byte-for-byte:
`encode(M)`, `GV`, `P2_W`/`P2_C`, `CS`, `K_hmac`, `tag`, `commit(value,r)`, `encodeRecord(REC)`, and the
AES-256-GCM ciphertext/tag, for every vector. Any mismatch is a release blocker.
