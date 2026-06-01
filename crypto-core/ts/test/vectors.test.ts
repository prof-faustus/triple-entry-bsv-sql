import { test } from "node:test";
import assert from "node:assert/strict";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { bytesToHex, hexToBytes, utf8, Writer } from "../src/bytes.js";
import { bytesToBigIntBE, pubFromPriv, pubAddScalarG } from "../src/curve.js";
import { ImageKind, encodeMessage, encodeRecord, type ChangeMessage, type Op } from "../src/message.js";
import { generatorValue, commonSecretAsWriter, commonSecretAsCounterparty, deriveHmacKey, tag, commit } from "../src/keystone.js";
import { sha256, hmacSha256, hkdfSha256, aes256gcmEncrypt, aes256gcmDecrypt } from "../src/hashes.js";

const VEC = resolve(process.cwd(), "..", "vectors");
const core = JSON.parse(readFileSync(resolve(VEC, "core_vectors.json"), "utf8"));
const rfc = JSON.parse(readFileSync(resolve(VEC, "rfc_vectors.json"), "utf8"));

function changeImage(kind: ImageKind, value: Uint8Array, r: Uint8Array): Uint8Array {
  return kind === ImageKind.PLAINTEXT ? new Writer().bytes(value).finish() : commit(value, r);
}

test("RFC primitive KATs (sha256 / hmac / hkdf)", () => {
  for (const v of rfc.sha256) assert.equal(bytesToHex(sha256(utf8(v.input_utf8))), v.digest, v.name);
  for (const v of rfc.hmac_sha256) assert.equal(bytesToHex(hmacSha256(hexToBytes(v.key), hexToBytes(v.data))), v.mac, v.name);
  for (const v of rfc.hkdf_sha256)
    assert.equal(bytesToHex(hkdfSha256(hexToBytes(v.salt), hexToBytes(v.ikm), hexToBytes(v.info), v.length)), v.okm, v.name);
});

test("party public keys match vectors", () => {
  const wPriv = bytesToBigIntBE(hexToBytes(core.parties.writer.priv));
  const cPriv = bytesToBigIntBE(hexToBytes(core.parties.counterparty.priv));
  assert.equal(bytesToHex(pubFromPriv(wPriv)), core.parties.writer.pub);
  assert.equal(bytesToHex(pubFromPriv(cPriv)), core.parties.counterparty.pub);
});

test("core vectors reproduce byte-for-byte (encode/GV/subkeys/CS/Khmac/tag/commit/record)", () => {
  const wPriv = bytesToBigIntBE(hexToBytes(core.parties.writer.priv));
  const cPriv = bytesToBigIntBE(hexToBytes(core.parties.counterparty.priv));
  const wPub = hexToBytes(core.parties.writer.pub);
  const cPub = hexToBytes(core.parties.counterparty.pub);

  for (const tc of core.cases) {
    const m: ChangeMessage = {
      tableId: tc.message.tableId,
      rowId: hexToBytes(tc.message.rowId),
      columnId: tc.message.columnId,
      op: tc.message.op as Op,
      seq: BigInt(tc.message.seq),
      prevTxid: hexToBytes(tc.message.prevTxid),
    };
    const value = hexToBytes(tc.value);
    const r = hexToBytes(tc.blinding);
    const kind = tc.imageKind as ImageKind;

    assert.equal(bytesToHex(encodeMessage(m)), tc.expect.encodedMessage, `${tc.name} encodedMessage`);
    const { gv, gvBytes } = generatorValue(m);
    assert.equal(bytesToHex(gvBytes), tc.expect.gv, `${tc.name} gv`);
    assert.equal(bytesToHex(pubAddScalarG(wPub, gv)), tc.expect.subPubWriter, `${tc.name} P2_W`);
    assert.equal(bytesToHex(pubAddScalarG(cPub, gv)), tc.expect.subPubCounterparty, `${tc.name} P2_C`);

    const csW = commonSecretAsWriter({ priv: wPriv, pub: wPub }, cPub, gv);
    const csC = commonSecretAsCounterparty({ priv: cPriv, pub: cPub }, wPub, gv);
    assert.equal(bytesToHex(csW), tc.expect.cs, `${tc.name} cs (writer)`);
    assert.equal(bytesToHex(csC), tc.expect.cs, `${tc.name} cs (counterparty symmetry)`);

    const k = deriveHmacKey(csW, m);
    assert.equal(bytesToHex(k), tc.expect.kHmac, `${tc.name} kHmac`);
    const img = changeImage(kind, value, r);
    assert.equal(bytesToHex(img), tc.expect.changeImage, `${tc.name} changeImage`);
    assert.equal(bytesToHex(tag(k, img)), tc.expect.tag, `${tc.name} tag`);
    assert.equal(bytesToHex(commit(value, r)), tc.expect.commit, `${tc.name} commit`);

    const rec = encodeRecord({ streamId: utf8(m.tableId), message: m, imageKind: kind, changeImage: img, tag: tag(k, img) });
    assert.equal(bytesToHex(rec), tc.expect.encodedRecord, `${tc.name} encodedRecord`);
  }
});

test("AEAD vectors: encrypt matches, decrypt round-trips, tamper fails", () => {
  for (const v of core.aead) {
    const key = hexToBytes(v.key), nonce = hexToBytes(v.nonce), aad = hexToBytes(v.aad), pt = hexToBytes(v.plaintext);
    const { ciphertext, tag: t } = aes256gcmEncrypt(key, nonce, aad, pt);
    assert.equal(bytesToHex(ciphertext), v.ciphertext, `${v.name} ct`);
    assert.equal(bytesToHex(t), v.tag, `${v.name} tag`);
    assert.equal(bytesToHex(aes256gcmDecrypt(key, nonce, aad, ciphertext, t)), bytesToHex(pt), `${v.name} decrypt`);
    const bad = ciphertext.slice();
    bad[0] ^= 0x01;
    assert.throws(() => aes256gcmDecrypt(key, nonce, aad, bad, t), `${v.name} tamper must fail`);
  }
});
