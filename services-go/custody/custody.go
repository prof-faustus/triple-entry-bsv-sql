// Package custody implements threshold custody (Phase 6, SYS-CUST-*): loss-resistant key sharing
// (EP3259724B1) with shares transmitted/stored encrypted under the ECDH common secret, and native
// N-of-M authorisation via bare OP_CHECKMULTISIG (SYS-CUST-003, no P2SH). The no-key-reconstruction
// threshold-signing of US11671255 is the production alternative to bare multisig; bare multisig is the
// native realisation provided here (M distinct keys sign; no single private key is ever reconstructed).
package custody

import (
	"crypto/rand"
	"errors"
	"math/big"

	"github.com/bsv-blockchain/go-sdk/script"
	cc "te-bsv/cryptocore"
)

// secp256k1 group order.
var nOrder, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)

// Share is a Shamir share (x, y) of a secret scalar mod n.
type Share struct {
	X *big.Int
	Y *big.Int
}

// Split shares `secret` into `total` shares with threshold `need` (Shamir over GF(n)) — loss-resistant:
// any `need` shares recover the secret, fewer reveal nothing (EP3259724B1).
func Split(secret *big.Int, need, total int) ([]Share, error) {
	if need < 2 || need > total {
		return nil, errors.New("require 2 <= need <= total")
	}
	coeffs := make([]*big.Int, need)
	coeffs[0] = new(big.Int).Mod(secret, nOrder)
	for i := 1; i < need; i++ {
		c, err := rand.Int(rand.Reader, nOrder)
		if err != nil {
			return nil, err
		}
		coeffs[i] = c
	}
	shares := make([]Share, total)
	for i := 0; i < total; i++ {
		x := big.NewInt(int64(i + 1))
		y := new(big.Int).Set(coeffs[need-1])
		for j := need - 2; j >= 0; j-- {
			y.Mul(y, x)
			y.Add(y, coeffs[j])
			y.Mod(y, nOrder)
		}
		shares[i] = Share{X: x, Y: y}
	}
	return shares, nil
}

// Recover reconstructs the secret from `need` shares via Lagrange interpolation at x=0.
func Recover(shares []Share) *big.Int {
	secret := big.NewInt(0)
	for i := range shares {
		num := big.NewInt(1)
		den := big.NewInt(1)
		for j := range shares {
			if i == j {
				continue
			}
			num.Mul(num, new(big.Int).Neg(shares[j].X))
			num.Mod(num, nOrder)
			d := new(big.Int).Sub(shares[i].X, shares[j].X)
			den.Mul(den, d)
			den.Mod(den, nOrder)
		}
		inv := new(big.Int).ModInverse(den, nOrder)
		term := new(big.Int).Mul(shares[i].Y, num)
		term.Mul(term, inv)
		term.Mod(term, nOrder)
		secret.Add(secret, term)
		secret.Mod(secret, nOrder)
	}
	return secret
}

// EncryptShare encrypts a share under a key derived from the ECDH common secret (EP3259724B1: shares
// transmitted encrypted under the common secret). Returns nonce||ct||tag-friendly tuple via cryptocore.
func EncryptShare(cs []byte, location string, s Share) (ct, tag []byte, err error) {
	key, err := cc.HKDFSHA256([]byte(location), cs, []byte("TE/custody/v1"), 32)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, 12)
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, err
	}
	pt := append(s.X.Bytes(), 0x00) // separator
	pt = append(pt, s.Y.Bytes()...)
	ct, tag, err = cc.AES256GCMEncrypt(key, nonce, []byte(location), pt)
	if err != nil {
		return nil, nil, err
	}
	return append(nonce, ct...), tag, nil
}

// StorageLayout records where shares live; EP3259724B1 requires >=3 shares at separate locations,
// at least one in a backup/safe-storage facility.
type StorageLayout struct {
	Locations []string
	Backup    string
}

func (l StorageLayout) Valid() bool {
	if l.Backup == "" || len(l.Locations) < 3 {
		return false
	}
	hasBackup := false
	for _, loc := range l.Locations {
		if loc == l.Backup {
			hasBackup = true
		}
	}
	return hasBackup
}

// BuildBareMultisig builds a native N-of-M locking script: OP_m <pubkeys...> OP_n OP_CHECKMULTISIG
// (SYS-CUST-003; no P2SH wrapper).
func BuildBareMultisig(need int, pubkeys [][]byte) (*script.Script, error) {
	if need < 1 || need > len(pubkeys) || len(pubkeys) > 16 {
		return nil, errors.New("invalid m-of-n")
	}
	s := &script.Script{}
	if err := s.AppendOpcodes(byte(0x50 + need)); err != nil { // OP_1..OP_16
		return nil, err
	}
	for _, pk := range pubkeys {
		if err := s.AppendPushData(pk); err != nil {
			return nil, err
		}
	}
	if err := s.AppendOpcodes(byte(0x50+len(pubkeys)), script.OpCHECKMULTISIG); err != nil {
		return nil, err
	}
	return s, nil
}
