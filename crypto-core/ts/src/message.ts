// Change Message M(c) and the on-chain field record. ALGORITHMS.md §1.1–§1.2.
import { Writer, Reader, utf8 } from "./bytes.js";

export const MAGIC_M = utf8("TEMC"); // Triple-Entry Message Canonical
export const MAGIC_R = utf8("TER1"); // Triple-Entry Record v1
export const VERSION = 1;

export enum Op {
  INSERT = 1,
  UPDATE = 2,
  DELETE = 3,
}

export enum ImageKind {
  PLAINTEXT = 0,
  COMMITMENT = 1,
}

export interface ChangeMessage {
  tableId: string;
  rowId: Uint8Array;
  columnId: string;
  op: Op;
  seq: bigint;
  prevTxid: Uint8Array; // 32 bytes, or empty at genesis
}

/** encode M(c) — the public change identity hashed to GV. ALGORITHMS.md §1.1 */
export function encodeMessage(m: ChangeMessage): Uint8Array {
  if (m.prevTxid.length !== 0 && m.prevTxid.length !== 32) throw new Error("prevTxid must be empty or 32 bytes");
  return new Writer()
    .raw(MAGIC_M)
    .u8(VERSION)
    .str(m.tableId)
    .bytes(m.rowId)
    .str(m.columnId)
    .u8(m.op)
    .u64(m.seq)
    .bytes(m.prevTxid)
    .finish();
}

export interface FieldRecord {
  streamId: Uint8Array;
  seq: bigint;
  prevTxid: Uint8Array;
  imageKind: ImageKind;
  changeImage: Uint8Array;
  tag: Uint8Array; // 32 bytes
}

/** encode the on-chain field record carried as spendable-script pushdata. ALGORITHMS.md §1.2 */
export function encodeRecord(r: FieldRecord): Uint8Array {
  if (r.prevTxid.length !== 0 && r.prevTxid.length !== 32) throw new Error("prevTxid must be empty or 32 bytes");
  if (r.tag.length !== 32) throw new Error("tag must be 32 bytes");
  return new Writer()
    .raw(MAGIC_R)
    .u8(VERSION)
    .bytes(r.streamId)
    .u64(r.seq)
    .bytes(r.prevTxid)
    .u8(r.imageKind)
    .bytes(r.changeImage)
    .bytes(r.tag)
    .finish();
}

export function decodeRecord(buf: Uint8Array): FieldRecord {
  const rd = new Reader(buf);
  const magic = rd.raw(4);
  for (let i = 0; i < 4; i++) if (magic[i] !== MAGIC_R[i]) throw new Error("bad record magic");
  const version = rd.u8();
  if (version !== VERSION) throw new Error(`unsupported record version ${version}`);
  const streamId = rd.bytes();
  const seq = rd.u64();
  const prevTxid = rd.bytes();
  const imageKind = rd.u8() as ImageKind;
  if (imageKind !== ImageKind.PLAINTEXT && imageKind !== ImageKind.COMMITMENT) throw new Error("bad image_kind");
  const changeImage = rd.bytes();
  const tag = rd.bytes();
  rd.end();
  if (tag.length !== 32) throw new Error("tag must be 32 bytes");
  return { streamId, seq, prevTxid, imageKind, changeImage, tag };
}
