// secp256k1 wrappers (audited @noble/curves). ALGORITHMS.md §2.
import { secp256k1 } from "@noble/curves/secp256k1";
import { bytesToHex } from "./bytes.js";

const Point = secp256k1.Point;
export const N: bigint = secp256k1.CURVE.n;

export function bytesToBigIntBE(b: Uint8Array): bigint {
  let v = 0n;
  for (const x of b) v = (v << 8n) | BigInt(x);
  return v;
}

/** Public key (compressed, 33 bytes) for a private scalar in [1, n-1]. */
export function pubFromPriv(priv: bigint): Uint8Array {
  if (priv <= 0n || priv >= N) throw new Error("priv out of range");
  return Point.BASE.multiply(priv).toRawBytes(true);
}

/** Add the scalar GV·G to a public point P, returning compressed bytes (P2 = P1 + GV·G). */
export function pubAddScalarG(pubCompressed: Uint8Array, gv: bigint): Uint8Array {
  const gvModN = ((gv % N) + N) % N;
  const P1 = Point.fromBytes(pubCompressed);
  const sum = gvModN === 0n ? P1 : P1.add(Point.BASE.multiply(gvModN));
  if (sum.is0()) throw new Error("derived sub-public key is point at infinity");
  return sum.toRawBytes(true);
}

/** ECDH: scalar · point → compressed bytes of the resulting point. */
export function mulPointByScalar(pubCompressed: Uint8Array, scalar: bigint): Uint8Array {
  const s = ((scalar % N) + N) % N;
  if (s === 0n) throw new Error("scalar is zero mod n");
  const R = Point.fromBytes(pubCompressed).multiply(s);
  if (R.is0()) throw new Error("ECDH result is point at infinity");
  return R.toRawBytes(true);
}

/** (v + GV) mod n, rejecting the degenerate zero result. */
export function addScalarsModN(v: bigint, gv: bigint): bigint {
  const r = (((v % N) + N) % N + ((gv % N) + N) % N) % N;
  if (r === 0n) throw new Error("derived sub-private key is zero mod n");
  return r;
}

export const debugPubHex = (p: Uint8Array): string => bytesToHex(p);
