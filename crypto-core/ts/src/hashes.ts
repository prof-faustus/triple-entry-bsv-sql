// Hash / MAC / KDF / AEAD via Node's built-in crypto (no third-party deps). ALGORITHMS.md §3–§5.
import { createHash, createHmac, hkdfSync, createCipheriv, createDecipheriv } from "node:crypto";

export function sha256(data: Uint8Array): Uint8Array {
  return new Uint8Array(createHash("sha256").update(data).digest());
}

export function hmacSha256(key: Uint8Array, data: Uint8Array): Uint8Array {
  return new Uint8Array(createHmac("sha256", key).update(data).digest());
}

/** RFC-5869 HKDF with SHA-256. */
export function hkdfSha256(salt: Uint8Array, ikm: Uint8Array, info: Uint8Array, length: number): Uint8Array {
  // Node's hkdfSync requires a non-zero-length ikm; salt may be empty.
  const out = hkdfSync("sha256", ikm, salt, info, length);
  return new Uint8Array(out);
}

export interface AeadResult {
  ciphertext: Uint8Array;
  tag: Uint8Array; // 16 bytes
}

/** AES-256-GCM. key=32, nonce=12, tag=16. ALGORITHMS.md §5. */
export function aes256gcmEncrypt(
  key: Uint8Array,
  nonce: Uint8Array,
  aad: Uint8Array,
  plaintext: Uint8Array,
): AeadResult {
  if (key.length !== 32) throw new Error("AES-256-GCM key must be 32 bytes");
  if (nonce.length !== 12) throw new Error("GCM nonce must be 12 bytes");
  const c = createCipheriv("aes-256-gcm", key, nonce);
  c.setAAD(aad);
  const ct = Buffer.concat([c.update(Buffer.from(plaintext)), c.final()]);
  return { ciphertext: new Uint8Array(ct), tag: new Uint8Array(c.getAuthTag()) };
}

export function aes256gcmDecrypt(
  key: Uint8Array,
  nonce: Uint8Array,
  aad: Uint8Array,
  ciphertext: Uint8Array,
  tag: Uint8Array,
): Uint8Array {
  const d = createDecipheriv("aes-256-gcm", key, nonce);
  d.setAAD(aad);
  d.setAuthTag(Buffer.from(tag));
  const pt = Buffer.concat([d.update(Buffer.from(ciphertext)), d.final()]);
  return new Uint8Array(pt);
}
