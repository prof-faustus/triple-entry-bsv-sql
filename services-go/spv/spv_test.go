package spv

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func leaf(s string) []byte { h := sha256.Sum256([]byte(s)); return h[:] }

func TestMerkleProofRoundTrip(t *testing.T) {
	for _, n := range []int{1, 2, 3, 5, 8, 13} {
		leaves := make([][]byte, n)
		for i := range leaves {
			leaves[i] = leaf("tx" + string(rune('a'+i)))
		}
		root := MerkleRoot(leaves)
		for idx := 0; idx < n; idx++ {
			branch, err := BranchFor(leaves, idx)
			if err != nil {
				t.Fatal(err)
			}
			got := RootFromBranch(leaves[idx], branch, idx)
			if hex.EncodeToString(got) != hex.EncodeToString(root) {
				t.Fatalf("n=%d idx=%d: branch does not recompute root", n, idx)
			}
		}
	}
}

func TestBURIVerifyAndTamper(t *testing.T) {
	leaves := [][]byte{leaf("coinbase"), leaf("third-entry"), leaf("token"), leaf("doc")}
	root := MerkleRoot(leaves)
	b, err := BuildBURI("block#100", leaves, 1)
	if err != nil {
		t.Fatal(err)
	}
	// round-trip string
	b2, err := ParseBURI(b.String())
	if err != nil {
		t.Fatal(err)
	}
	ok, err := b2.Verify(root)
	if err != nil || !ok {
		t.Fatalf("BURI must verify against the header merkle root: ok=%v err=%v", ok, err)
	}
	// tamper: wrong root must fail
	bad := MerkleRoot([][]byte{leaf("x"), leaf("y")})
	if ok, _ := b2.Verify(bad); ok {
		t.Fatal("BURI must NOT verify against a wrong root")
	}
}

func TestSingleTxBlockTrivialProof(t *testing.T) {
	// A single-tx (coinbase-only) block: merkleroot == the leaf; empty branch verifies.
	cb := leaf("coinbase-only")
	root := MerkleRoot([][]byte{cb})
	b, err := BuildBURI("block#1", [][]byte{cb}, 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(b.Branch) != 0 {
		t.Fatalf("single-leaf branch should be empty, got %d", len(b.Branch))
	}
	if ok, _ := b.Verify(root); !ok {
		t.Fatal("single-leaf BURI must verify")
	}
}
