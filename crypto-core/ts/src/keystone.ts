// The ECDH-HMAC keystone: GV → sub-keys → CS → K_hmac → tag, plus commitment. ALGORITHMS.md §2–§4.
// Grounded in EP3860037A1; CTO primitive assumptions documented in spec/CTO-PRIMITIVES.md.
import { Writer, concat, utf8 } from "./bytes.js";
import { sha256, hmacSha256, hkdfSha256 } from "./hashes.js";
import { addScalarsModN, bytesToBigIntBE, mulPointByScalar, pubAddScalarG } from "./curve.js";
import { encodeMessage, type ChangeMessage } from "./message.js";

const HMAC_DOMAIN = utf8("TE/hmac/v1");
const COMMIT_DOMAIN = utf8("CTO/commit/v1"); // CTO substrate commitment (CTO_BSV_Build_Spec_v1 §6/T4)

/** GV = SHA-256(M(c)), as a big-endian scalar. ALGORITHMS.md §2 */
export function generatorValue(m: ChangeMessage): { gvBytes: Uint8Array; gv: bigint } {
  const gvBytes = sha256(encodeMessage(m));
  return { gvBytes, gv: bytesToBigIntBE(gvBytes) };
}

export interface Party {
  priv: bigint; // master private scalar
  pub: Uint8Array; // master public (compressed, 33 bytes)
}

/** Writer-side common secret CS = compressed( v2_W · P2_C ). ALGORITHMS.md §2 */
export function commonSecretAsWriter(writer: Party, counterpartyPub: Uint8Array, gv: bigint): Uint8Array {
  const v2W = addScalarsModN(writer.priv, gv);
  const P2C = pubAddScalarG(counterpartyPub, gv);
  return mulPointByScalar(P2C, v2W);
}

/** Counterparty-side common secret CS = compressed( v2_C · P2_W ) — must equal the writer's. */
export function commonSecretAsCounterparty(counterparty: Party, writerPub: Uint8Array, gv: bigint): Uint8Array {
  const v2C = addScalarsModN(counterparty.priv, gv);
  const P2W = pubAddScalarG(writerPub, gv);
  return mulPointByScalar(P2W, v2C);
}

/** K_hmac = HKDF(salt=table||row||column, ikm=CS, info="TE/hmac/v1"||u64(seq), L=32). ALGORITHMS.md §3.2 */
export function deriveHmacKey(cs: Uint8Array, m: ChangeMessage): Uint8Array {
  const salt = concat(utf8(m.tableId), m.rowId, utf8(m.columnId));
  const info = new Writer().raw(HMAC_DOMAIN).u64(m.seq).finish();
  return hkdfSha256(salt, cs, info, 32);
}

/** tag(c) = HMAC-SHA256(K_hmac, change_image). ALGORITHMS.md §3.3 */
export function tag(kHmac: Uint8Array, changeImage: Uint8Array): Uint8Array {
  return hmacSha256(kHmac, changeImage);
}

/** CTO blinded commitment: SHA-256(domain ‖ r ‖ value), raw concat, r = 32-byte blinding
 *  (CTO_BSV_Build_Spec_v1 §6 / Step T4; on-chain-openable via OP_CAT). ALGORITHMS.md §4. */
export function commit(value: Uint8Array, r: Uint8Array): Uint8Array {
  return sha256(concat(COMMIT_DOMAIN, r, value));
}

/** Convenience: full keystone from writer's perspective → { gv, cs, kHmac, tag }. */
export function computeTagAsWriter(
  writer: Party,
  counterpartyPub: Uint8Array,
  m: ChangeMessage,
  changeImage: Uint8Array,
): { gvBytes: Uint8Array; cs: Uint8Array; kHmac: Uint8Array; tag: Uint8Array } {
  const { gv, gvBytes } = generatorValue(m);
  const cs = commonSecretAsWriter(writer, counterpartyPub, gv);
  const kHmac = deriveHmacKey(cs, m);
  return { gvBytes, cs, kHmac, tag: tag(kHmac, changeImage) };
}
