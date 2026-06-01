package cryptocore

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"io"

	"github.com/decred/dcrd/dcrec/secp256k1/v4"
	"golang.org/x/crypto/hkdf"
)

// --- hashes / MAC / KDF (ALGORITHMS.md §3) ---

func SHA256(data []byte) []byte {
	h := sha256.Sum256(data)
	return h[:]
}

func HMACSHA256(key, data []byte) []byte {
	mac := hmac.New(sha256.New, key)
	mac.Write(data)
	return mac.Sum(nil)
}

// HKDFSHA256 is RFC-5869 HKDF with SHA-256.
func HKDFSHA256(salt, ikm, info []byte, length int) ([]byte, error) {
	r := hkdf.New(sha256.New, ikm, salt, info)
	out := make([]byte, length)
	if _, err := io.ReadFull(r, out); err != nil {
		return nil, err
	}
	return out, nil
}

// --- secp256k1 (ALGORITHMS.md §2) ---

func scalarFromBytes(b []byte) (*secp256k1.ModNScalar, error) {
	s := new(secp256k1.ModNScalar)
	s.SetByteSlice(b) // reduces mod n; overflow bool ignored (priv keys are < n, gv is reduced by design)
	if s.IsZero() {
		return nil, errors.New("scalar is zero mod n")
	}
	return s, nil
}

// PubFromPriv returns the compressed public key for a 32-byte private scalar.
func PubFromPriv(privBytes []byte) []byte {
	priv := secp256k1.PrivKeyFromBytes(privBytes)
	return priv.PubKey().SerializeCompressed()
}

func compress(jp *secp256k1.JacobianPoint) []byte {
	cp := *jp
	cp.ToAffine()
	return secp256k1.NewPublicKey(&cp.X, &cp.Y).SerializeCompressed()
}

// subPubJ returns P2 = P1 + gv·G (Jacobian).
func subPubJ(pubBytes []byte, gv *secp256k1.ModNScalar) (*secp256k1.JacobianPoint, error) {
	pk, err := secp256k1.ParsePubKey(pubBytes)
	if err != nil {
		return nil, err
	}
	var P1, gvG, P2 secp256k1.JacobianPoint
	pk.AsJacobian(&P1)
	secp256k1.ScalarBaseMultNonConst(gv, &gvG)
	secp256k1.AddNonConst(&P1, &gvG, &P2)
	if P2.Z.IsZero() {
		return nil, errors.New("derived sub-public key is point at infinity")
	}
	return &P2, nil
}

// SubPub returns the compressed P2 = P1 + gv·G.
func SubPub(pubBytes []byte, gv *secp256k1.ModNScalar) ([]byte, error) {
	p, err := subPubJ(pubBytes, gv)
	if err != nil {
		return nil, err
	}
	return compress(p), nil
}

// GeneratorValue returns (gvBytes = SHA-256(M), gvScalar = gvBytes mod n).
func GeneratorValue(m ChangeMessage) (gvBytes []byte, gv *secp256k1.ModNScalar, err error) {
	enc, err := EncodeMessage(m)
	if err != nil {
		return nil, nil, err
	}
	gvBytes = SHA256(enc)
	gv = new(secp256k1.ModNScalar)
	gv.SetByteSlice(gvBytes)
	if gv.IsZero() {
		return nil, nil, errors.New("GV is zero mod n")
	}
	return gvBytes, gv, nil
}

func commonSecret(privBytes, otherPub []byte, gv *secp256k1.ModNScalar) ([]byte, error) {
	priv, err := scalarFromBytes(privBytes)
	if err != nil {
		return nil, err
	}
	v2 := new(secp256k1.ModNScalar)
	v2.Set(priv)
	v2.Add(gv)
	if v2.IsZero() {
		return nil, errors.New("derived sub-private key is zero mod n")
	}
	P2, err := subPubJ(otherPub, gv)
	if err != nil {
		return nil, err
	}
	var cs secp256k1.JacobianPoint
	secp256k1.ScalarMultNonConst(v2, P2, &cs)
	if cs.Z.IsZero() {
		return nil, errors.New("ECDH result is point at infinity")
	}
	return compress(&cs), nil
}

// CommonSecretAsWriter computes CS = compressed(v2_W · P2_C).
func CommonSecretAsWriter(writerPrivBytes, counterpartyPub []byte, gv *secp256k1.ModNScalar) ([]byte, error) {
	return commonSecret(writerPrivBytes, counterpartyPub, gv)
}

// CommonSecretAsCounterparty computes CS = compressed(v2_C · P2_W).
func CommonSecretAsCounterparty(counterpartyPrivBytes, writerPub []byte, gv *secp256k1.ModNScalar) ([]byte, error) {
	return commonSecret(counterpartyPrivBytes, writerPub, gv)
}

// --- key derivation / tag / commitment (ALGORITHMS.md §3–§4) ---

// DeriveHMACKey: HKDF(salt=table||row||column, ikm=CS, info="TE/hmac/v1"||u64(seq), L=32).
func DeriveHMACKey(cs []byte, m ChangeMessage) ([]byte, error) {
	salt := NewWriter().Raw([]byte(m.TableID)).Raw(m.RowID).Raw([]byte(m.ColumnID)).Finish()
	info := NewWriter().Raw([]byte("TE/hmac/v1")).U64(m.Seq).Finish()
	return HKDFSHA256(salt, cs, info, 32)
}

// Tag: HMAC-SHA256(K_hmac, change_image).
func Tag(kHmac, changeImage []byte) []byte { return HMACSHA256(kHmac, changeImage) }

// Commit is the CTO blinded commitment: SHA-256(domain ‖ r ‖ value), raw concat, r = 32-byte blinding
// (CTO_BSV_Build_Spec_v1 §6 / Step T4; on-chain-openable via OP_CAT). ALGORITHMS.md §4.
func Commit(value, r []byte) []byte {
	pre := NewWriter().Raw([]byte("CTO/commit/v1")).Raw(r).Raw(value).Finish()
	return SHA256(pre)
}

// ChangeImage builds the change image per kind (ALGORITHMS.md §3.4).
func ChangeImage(kind ImageKind, value, r []byte) []byte {
	if kind == ImagePlaintext {
		return NewWriter().Bytes(value).Finish()
	}
	return Commit(value, r)
}
