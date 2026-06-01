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

/** decode M(c) (inverse of encodeMessage). ALGORITHMS.md §1.1 */
export function decodeMessage(buf: Uint8Array): ChangeMessage {
  const rd = new Reader(buf);
  const magic = rd.raw(4);
  for (let i = 0; i < 4; i++) if (magic[i] !== MAGIC_M[i]) throw new Error("bad message magic");
  const version = rd.u8();
  if (version !== VERSION) throw new Error(`unsupported message version ${version}`);
  const tableId = rd.str();
  const rowId = rd.bytes();
  const columnId = rd.str();
  const op = rd.u8() as Op;
  if (op !== Op.INSERT && op !== Op.UPDATE && op !== Op.DELETE) throw new Error("bad op");
  const seq = rd.u64();
  const prevTxid = rd.bytes();
  rd.end();
  if (prevTxid.length !== 0 && prevTxid.length !== 32) throw new Error("prevTxid must be empty or 32 bytes");
  return { tableId, rowId, columnId, op, seq, prevTxid };
}

export interface FieldRecord {
  streamId: Uint8Array;
  message: ChangeMessage; // carries table/row/column/op/seq/prev_txid (self-describing)
  imageKind: ImageKind;
  changeImage: Uint8Array;
  tag: Uint8Array; // 32 bytes
}

/** encode the on-chain field record carried as spendable-script pushdata. ALGORITHMS.md §1.2 */
export function encodeRecord(r: FieldRecord): Uint8Array {
  if (r.tag.length !== 32) throw new Error("tag must be 32 bytes");
  return new Writer()
    .raw(MAGIC_R)
    .u8(VERSION)
    .bytes(r.streamId)
    .bytes(encodeMessage(r.message))
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
  const message = decodeMessage(rd.bytes());
  const imageKind = rd.u8() as ImageKind;
  if (imageKind !== ImageKind.PLAINTEXT && imageKind !== ImageKind.COMMITMENT) throw new Error("bad image_kind");
  const changeImage = rd.bytes();
  const tag = rd.bytes();
  rd.end();
  if (tag.length !== 32) throw new Error("tag must be 32 bytes");
  return { streamId, message, imageKind, changeImage, tag };
}
