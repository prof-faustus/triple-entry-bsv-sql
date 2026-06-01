// Canonical byte encoding primitives — ALGORITHMS.md §1.
// Deterministic, big-endian, length-prefixed. Shared contract with the Go (and future C) impls.

export function hexToBytes(hex: string): Uint8Array {
  if (hex.length % 2 !== 0) throw new Error("hex: odd length");
  const out = new Uint8Array(hex.length / 2);
  for (let i = 0; i < out.length; i++) {
    const b = Number.parseInt(hex.slice(i * 2, i * 2 + 2), 16);
    if (Number.isNaN(b)) throw new Error("hex: invalid char");
    out[i] = b;
  }
  return out;
}

export function bytesToHex(b: Uint8Array): string {
  let s = "";
  for (const x of b) s += x.toString(16).padStart(2, "0");
  return s;
}

export function concat(...parts: Uint8Array[]): Uint8Array {
  let n = 0;
  for (const p of parts) n += p.length;
  const out = new Uint8Array(n);
  let off = 0;
  for (const p of parts) {
    out.set(p, off);
    off += p.length;
  }
  return out;
}

export function utf8(s: string): Uint8Array {
  return new TextEncoder().encode(s);
}

/** Append-only canonical writer (big-endian). */
export class Writer {
  private chunks: Uint8Array[] = [];
  u8(x: number): this {
    if (x < 0 || x > 0xff || !Number.isInteger(x)) throw new Error("u8 range");
    this.chunks.push(Uint8Array.of(x));
    return this;
  }
  u16(x: number): this {
    if (x < 0 || x > 0xffff || !Number.isInteger(x)) throw new Error("u16 range");
    this.chunks.push(Uint8Array.of((x >>> 8) & 0xff, x & 0xff));
    return this;
  }
  u32(x: number): this {
    if (x < 0 || x > 0xffffffff || !Number.isInteger(x)) throw new Error("u32 range");
    this.chunks.push(Uint8Array.of((x >>> 24) & 0xff, (x >>> 16) & 0xff, (x >>> 8) & 0xff, x & 0xff));
    return this;
  }
  u64(x: bigint): this {
    if (x < 0n || x > 0xffffffffffffffffn) throw new Error("u64 range");
    const b = new Uint8Array(8);
    let v = x;
    for (let i = 7; i >= 0; i--) {
      b[i] = Number(v & 0xffn);
      v >>= 8n;
    }
    this.chunks.push(b);
    return this;
  }
  raw(b: Uint8Array): this {
    this.chunks.push(b);
    return this;
  }
  /** Length-prefixed octet string: u32(len) ‖ b */
  bytes(b: Uint8Array): this {
    return this.u32(b.length).raw(b);
  }
  /** Length-prefixed UTF-8 string. */
  str(s: string): this {
    return this.bytes(utf8(s));
  }
  finish(): Uint8Array {
    return concat(...this.chunks);
  }
}

/** Strict canonical reader; rejects truncation and trailing bytes (call end()). */
export class Reader {
  private off = 0;
  constructor(private readonly buf: Uint8Array) {}
  private need(n: number): void {
    if (this.off + n > this.buf.length) throw new Error("decode: truncated");
  }
  u8(): number {
    this.need(1);
    return this.buf[this.off++]!;
  }
  u16(): number {
    this.need(2);
    const v = (this.buf[this.off]! << 8) | this.buf[this.off + 1]!;
    this.off += 2;
    return v >>> 0;
  }
  u32(): number {
    this.need(4);
    // top byte multiplied (not shifted) to stay a safe non-negative integer up to 2^32-1
    const v =
      this.buf[this.off]! * 0x1000000 +
      (((this.buf[this.off + 1]! << 16) | (this.buf[this.off + 2]! << 8) | this.buf[this.off + 3]!) >>> 0);
    this.off += 4;
    return v;
  }
  u64(): bigint {
    this.need(8);
    let v = 0n;
    for (let i = 0; i < 8; i++) v = (v << 8n) | BigInt(this.buf[this.off + i]!);
    this.off += 8;
    return v;
  }
  raw(n: number): Uint8Array {
    this.need(n);
    const out = this.buf.slice(this.off, this.off + n);
    this.off += n;
    return out;
  }
  bytes(): Uint8Array {
    const n = this.u32();
    return this.raw(n);
  }
  str(): string {
    const b = this.bytes();
    const dec = new TextDecoder("utf-8", { fatal: true });
    return dec.decode(b);
  }
  /** Assert the whole buffer was consumed. */
  end(): void {
    if (this.off !== this.buf.length) throw new Error("decode: trailing bytes");
  }
}
