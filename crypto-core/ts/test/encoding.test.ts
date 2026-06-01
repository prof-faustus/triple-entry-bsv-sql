import { test } from "node:test";
import assert from "node:assert/strict";
import { Writer, Reader, bytesToHex, hexToBytes, utf8 } from "../src/bytes.js";
import { encodeRecord, decodeRecord, ImageKind, encodeMessage, Op } from "../src/message.js";

test("Writer/Reader round-trips primitives and length-prefixed fields", () => {
  const buf = new Writer().u8(1).u16(258).u32(0xdeadbeef).u64(0x0102030405060708n).bytes(utf8("hi")).str("héllo").finish();
  const rd = new Reader(buf);
  assert.equal(rd.u8(), 1);
  assert.equal(rd.u16(), 258);
  assert.equal(rd.u32(), 0xdeadbeef);
  assert.equal(rd.u64(), 0x0102030405060708n);
  assert.equal(bytesToHex(rd.bytes()), bytesToHex(utf8("hi")));
  assert.equal(rd.str(), "héllo");
  rd.end();
});

test("Reader rejects truncation, trailing bytes, bad utf8, oversized length", () => {
  assert.throws(() => new Reader(new Uint8Array([0x00])).u32(), /truncated/);
  const w = new Writer().u8(7).finish();
  const rd = new Reader(new Uint8Array([...w, 0xff]));
  assert.equal(rd.u8(), 7);
  assert.throws(() => rd.end(), /trailing/);
  // length prefix says 10 bytes but buffer has 1
  assert.throws(() => new Reader(new Uint8Array([0, 0, 0, 10, 1])).bytes(), /truncated/);
  // invalid utf8 (0xff is not valid)
  assert.throws(() => new Reader(new Uint8Array([0, 0, 0, 1, 0xff])).str(), /./);
});

test("record encode/decode round-trip", () => {
  const rec = {
    streamId: utf8("ledger.invoices"),
    message: { tableId: "ledger.invoices", rowId: hexToBytes("0001"), columnId: "amount", op: Op.UPDATE, seq: 7n, prevTxid: hexToBytes("ab".repeat(32)) },
    imageKind: ImageKind.COMMITMENT,
    changeImage: hexToBytes("cd".repeat(32)),
    tag: hexToBytes("ef".repeat(32)),
  };
  const enc = encodeRecord(rec);
  const dec = decodeRecord(enc);
  assert.equal(bytesToHex(encodeRecord(dec)), bytesToHex(enc));
  assert.equal(dec.message.seq, 7n);
  assert.equal(dec.message.columnId, "amount");
  assert.equal(dec.imageKind, ImageKind.COMMITMENT);
});

test("record decode rejects bad magic and trailing bytes", () => {
  const enc = encodeRecord({
    streamId: utf8("s"),
    message: { tableId: "t", rowId: new Uint8Array(0), columnId: "c", op: Op.INSERT, seq: 0n, prevTxid: new Uint8Array(0) },
    imageKind: ImageKind.PLAINTEXT,
    changeImage: utf8("v"),
    tag: hexToBytes("00".repeat(32)),
  });
  const bad = enc.slice();
  bad[0] = 0x58;
  assert.throws(() => decodeRecord(bad), /magic/);
  assert.throws(() => decodeRecord(new Uint8Array([...enc, 0x00])), /trailing/);
});

test("encodeMessage rejects malformed prevTxid", () => {
  assert.throws(
    () => encodeMessage({ tableId: "t", rowId: new Uint8Array(0), columnId: "c", op: Op.INSERT, seq: 0n, prevTxid: new Uint8Array(5) }),
    /prevTxid/,
  );
});
