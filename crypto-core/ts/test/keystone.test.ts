import { test } from "node:test";
import assert from "node:assert/strict";
import { bytesToHex, hexToBytes, utf8 } from "../src/bytes.js";
import { pubFromPriv, bytesToBigIntBE } from "../src/curve.js";
import { Op } from "../src/message.js";
import {
  generatorValue,
  commonSecretAsWriter,
  commonSecretAsCounterparty,
  deriveHmacKey,
  tag,
  commit,
} from "../src/keystone.js";
import { sha256 } from "../src/hashes.js";
import { encodeMessage } from "../src/message.js";

const writerPriv = bytesToBigIntBE(hexToBytes("e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262"));
const cpPriv = bytesToBigIntBE(hexToBytes("f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181"));
const writerPub = pubFromPriv(writerPriv);
const cpPub = pubFromPriv(cpPriv);

const m = { tableId: "ledger.invoices", rowId: hexToBytes("0001"), columnId: "amount", op: Op.INSERT, seq: 3n, prevTxid: new Uint8Array(0) };

test("GV = SHA-256(encodeMessage(M))", () => {
  const { gvBytes } = generatorValue(m);
  assert.equal(bytesToHex(gvBytes), bytesToHex(sha256(encodeMessage(m))));
});

test("common secret is symmetric (EP3860037A1): writer-side == counterparty-side", () => {
  const { gv } = generatorValue(m);
  const csW = commonSecretAsWriter({ priv: writerPriv, pub: writerPub }, cpPub, gv);
  const csC = commonSecretAsCounterparty({ priv: cpPriv, pub: cpPub }, writerPub, gv);
  assert.equal(bytesToHex(csW), bytesToHex(csC));
  assert.equal(csW.length, 33);
});

test("different message → different GV → different CS and tag", () => {
  const { gv: gv1 } = generatorValue(m);
  const { gv: gv2 } = generatorValue({ ...m, seq: 4n });
  assert.notEqual(gv1, gv2);
  const cs1 = commonSecretAsWriter({ priv: writerPriv, pub: writerPub }, cpPub, gv1);
  const cs2 = commonSecretAsWriter({ priv: writerPriv, pub: writerPub }, cpPub, gv2);
  assert.notEqual(bytesToHex(cs1), bytesToHex(cs2));
});

test("K_hmac is 32 bytes and deterministic; tag binds the image", () => {
  const { gv } = generatorValue(m);
  const cs = commonSecretAsWriter({ priv: writerPriv, pub: writerPub }, cpPub, gv);
  const k = deriveHmacKey(cs, m);
  assert.equal(k.length, 32);
  assert.equal(bytesToHex(deriveHmacKey(cs, m)), bytesToHex(k));
  const t1 = tag(k, utf8("1500.00"));
  const t2 = tag(k, utf8("1500.01"));
  assert.notEqual(bytesToHex(t1), bytesToHex(t2)); // any change to the value breaks the tag (SYS-HMAC-011a)
});

test("commitment: deterministic, binding-sensitive, hidden by blinding", () => {
  const v = utf8("1500.00");
  const r1 = hexToBytes("11".repeat(32));
  const r2 = hexToBytes("22".repeat(32));
  assert.equal(bytesToHex(commit(v, r1)), bytesToHex(commit(v, r1))); // deterministic
  assert.notEqual(bytesToHex(commit(v, r1)), bytesToHex(commit(v, r2))); // different blinding hides
  assert.notEqual(bytesToHex(commit(v, r1)), bytesToHex(commit(utf8("1500.01"), r1))); // binding to value
  assert.equal(commit(v, r1).length, 32);
});
