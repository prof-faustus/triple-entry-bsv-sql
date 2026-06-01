// Generates crypto-core/vectors/core_vectors.json — the single cross-impl source of truth.
// Run: npm run gen-vectors  (from crypto-core/ts). Deterministic; commit the output.
import { writeFileSync } from "node:fs";
import { resolve } from "node:path";
import { bytesToHex, hexToBytes, utf8, Writer } from "../src/bytes.js";
import { pubFromPriv, bytesToBigIntBE } from "../src/curve.js";
import { Op, ImageKind, encodeMessage, encodeRecord, type ChangeMessage } from "../src/message.js";
import {
  generatorValue,
  commonSecretAsWriter,
  commonSecretAsCounterparty,
  deriveHmacKey,
  tag as makeTag,
  commit,
} from "../src/keystone.js";
import { pubAddScalarG } from "../src/curve.js";
import { aes256gcmEncrypt } from "../src/hashes.js";

const WRITER_PRIV = "e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262";
const CP_PRIV = "f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181";

const writerPriv = bytesToBigIntBE(hexToBytes(WRITER_PRIV));
const cpPriv = bytesToBigIntBE(hexToBytes(CP_PRIV));
const writerPub = pubFromPriv(writerPriv);
const cpPub = pubFromPriv(cpPriv);

function changeImage(kind: ImageKind, value: Uint8Array, r: Uint8Array): Uint8Array {
  return kind === ImageKind.PLAINTEXT ? new Writer().bytes(value).finish() : commit(value, r);
}

interface CaseIn {
  name: string;
  m: ChangeMessage;
  value: Uint8Array;
  blinding: Uint8Array;
  kind: ImageKind;
}

const r1 = hexToBytes("11".repeat(32));
const r2 = hexToBytes("22".repeat(32));

const cases: CaseIn[] = [
  {
    name: "genesis-insert-plaintext",
    m: { tableId: "ledger.invoices", rowId: hexToBytes("0001"), columnId: "amount", op: Op.INSERT, seq: 0n, prevTxid: new Uint8Array(0) },
    value: utf8("1500.00"),
    blinding: r1,
    kind: ImageKind.PLAINTEXT,
  },
  {
    name: "update-confidential-commitment",
    m: { tableId: "ledger.invoices", rowId: hexToBytes("0001"), columnId: "amount", op: Op.UPDATE, seq: 1n, prevTxid: hexToBytes("ab".repeat(32)) },
    value: utf8("1499.95"),
    blinding: r2,
    kind: ImageKind.COMMITMENT,
  },
  {
    name: "delete-plaintext-unicode",
    m: { tableId: "ledgér.notes", rowId: hexToBytes("00ff10"), columnId: "memo", op: Op.DELETE, seq: 42n, prevTxid: hexToBytes("cd".repeat(32)) },
    value: utf8("closed — paid in full ✓"),
    blinding: r1,
    kind: ImageKind.PLAINTEXT,
  },
];

const out = {
  algorithm: "TE-BSV crypto core",
  version: 1,
  spec: "spec/ALGORITHMS.md",
  curve: "secp256k1",
  note: "Cross-implementation KAT source of truth (SYS-TEST-003). Hex is lowercase, no 0x prefix.",
  parties: {
    writer: { priv: WRITER_PRIV, pub: bytesToHex(writerPub) },
    counterparty: { priv: CP_PRIV, pub: bytesToHex(cpPub) },
  },
  cases: cases.map((c) => {
    const enc = encodeMessage(c.m);
    const { gv, gvBytes } = generatorValue(c.m);
    const subPubWriter = pubAddScalarG(writerPub, gv);
    const subPubCp = pubAddScalarG(cpPub, gv);
    const csW = commonSecretAsWriter({ priv: writerPriv, pub: writerPub }, cpPub, gv);
    const csC = commonSecretAsCounterparty({ priv: cpPriv, pub: cpPub }, writerPub, gv);
    if (bytesToHex(csW) !== bytesToHex(csC)) throw new Error(`CS asymmetry in ${c.name}`);
    const kHmac = deriveHmacKey(csW, c.m);
    const img = changeImage(c.kind, c.value, c.blinding);
    const t = makeTag(kHmac, img);
    const rec = encodeRecord({ streamId: utf8(c.m.tableId), seq: c.m.seq, prevTxid: c.m.prevTxid, imageKind: c.kind, changeImage: img, tag: t });
    return {
      name: c.name,
      message: {
        tableId: c.m.tableId,
        rowId: bytesToHex(c.m.rowId),
        columnId: c.m.columnId,
        op: c.m.op,
        seq: c.m.seq.toString(),
        prevTxid: bytesToHex(c.m.prevTxid),
      },
      value: bytesToHex(c.value),
      blinding: bytesToHex(c.blinding),
      imageKind: c.kind,
      expect: {
        encodedMessage: bytesToHex(enc),
        gv: bytesToHex(gvBytes),
        subPubWriter: bytesToHex(subPubWriter),
        subPubCounterparty: bytesToHex(subPubCp),
        cs: bytesToHex(csW),
        kHmac: bytesToHex(kHmac),
        changeImage: bytesToHex(img),
        tag: bytesToHex(t),
        commit: bytesToHex(commit(c.value, c.blinding)),
        encodedRecord: bytesToHex(rec),
      },
    };
  }),
  aead: [
    (() => {
      const key = hexToBytes("00112233445566778899aabbccddeeff00112233445566778899aabbccddeeff");
      const nonce = hexToBytes("0123456789ab0123456789ab");
      const aad = utf8("TER1-header");
      const pt = utf8("confidential field plaintext");
      const { ciphertext, tag } = aes256gcmEncrypt(key, nonce, aad, pt);
      return {
        name: "aes-256-gcm-1",
        key: bytesToHex(key),
        nonce: bytesToHex(nonce),
        aad: bytesToHex(aad),
        plaintext: bytesToHex(pt),
        ciphertext: bytesToHex(ciphertext),
        tag: bytesToHex(tag),
      };
    })(),
  ],
};

const path = resolve(process.cwd(), "..", "vectors", "core_vectors.json");
writeFileSync(path, JSON.stringify(out, null, 2) + "\n");
console.log(`wrote ${path} (${out.cases.length} cases, ${out.aead.length} aead)`);
