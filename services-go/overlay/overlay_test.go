package overlay

import (
	"bytes"
	"encoding/hex"
	"testing"

	cc "te-bsv/cryptocore"
)

func TestCKDDeterministicAndStructured(t *testing.T) {
	seed, _ := hex.DecodeString("000102030405060708090a0b0c0d0e0f")
	m, err := Master(seed)
	if err != nil {
		t.Fatal(err)
	}
	a, err := m.DerivePath(5, 2, 9)
	if err != nil {
		t.Fatal(err)
	}
	b, err := m.DerivePath(5, 2, 9)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(a.Pub, b.Pub) {
		t.Fatal("derivation must be deterministic")
	}
	c, err := m.DerivePath(5, 2, 10)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(a.Pub, c.Pub) {
		t.Fatal("different overlay positions must yield different keys")
	}
	var pb [32]byte
	a.Priv.FillBytes(pb[:])
	if !bytes.Equal(a.Pub, cc.PubFromPriv(pb[:])) {
		t.Fatal("node pub must equal priv*G")
	}
	parent, err := m.DerivePath(5, 2)
	if err != nil {
		t.Fatal(err)
	}
	child, err := parent.Derive(9)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(child.Pub, a.Pub) {
		t.Fatal("parent.Derive(child) must match DerivePath (hierarchy mirrors the graph)")
	}
}
