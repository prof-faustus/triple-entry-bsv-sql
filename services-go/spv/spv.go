// Package spv implements SPV inclusion proofs and the BURI reference (Phase 6, SYS-PROOF-*),
// grounded in WO2022100946A1 (Merkle proof) and WO2022214264A1 (BURI). A proof shows a target tx
// exists in a block's transaction-Merkle tree, SPV-verifiable against the block-header Merkle root
// WITHOUT the block payload — so the triple-entry log is auditable by anyone holding block headers.
package spv

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
)

// hash256 is Bitcoin's double-SHA-256.
func hash256(b []byte) []byte {
	a := sha256.Sum256(b)
	c := sha256.Sum256(a[:])
	return c[:]
}

// MerkleRoot computes the transaction-Merkle root over leaves (in block order). Odd levels duplicate
// the last node, as in Bitcoin.
func MerkleRoot(leaves [][]byte) []byte {
	if len(leaves) == 0 {
		return nil
	}
	level := make([][]byte, len(leaves))
	copy(level, leaves)
	for len(level) > 1 {
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}
		next := make([][]byte, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			next = append(next, hash256(append(append([]byte{}, level[i]...), level[i+1]...)))
		}
		level = next
	}
	return level[0]
}

// BranchFor returns the Merkle branch (sibling hashes bottom-up) for leaf at index.
func BranchFor(leaves [][]byte, index int) ([][]byte, error) {
	if index < 0 || index >= len(leaves) {
		return nil, fmt.Errorf("index out of range")
	}
	var branch [][]byte
	level := make([][]byte, len(leaves))
	copy(level, leaves)
	idx := index
	for len(level) > 1 {
		if len(level)%2 == 1 {
			level = append(level, level[len(level)-1])
		}
		sib := idx ^ 1
		branch = append(branch, level[sib])
		next := make([][]byte, 0, len(level)/2)
		for i := 0; i < len(level); i += 2 {
			next = append(next, hash256(append(append([]byte{}, level[i]...), level[i+1]...)))
		}
		level = next
		idx /= 2
	}
	return branch, nil
}

// RootFromBranch recomputes the Merkle root from a leaf, its branch, and its index (the SPV check).
func RootFromBranch(leaf []byte, branch [][]byte, index int) []byte {
	h := append([]byte{}, leaf...)
	idx := index
	for _, sib := range branch {
		if idx&1 == 0 {
			h = hash256(append(append([]byte{}, h...), sib...))
		} else {
			h = hash256(append(append([]byte{}, sib...), h...))
		}
		idx /= 2
	}
	return h
}

// BURI is a Blockchain Uniform Resource Indicator (WO2022214264A1): a delimiter-separated string
// carrying a block id, tx id, Merkle index, and proof hashes — SPV-verifiable without the block payload.
type BURI struct {
	BlockID string
	TxID    string
	Index   int
	Branch  []string // hex sibling hashes
}

const buriScheme = "buri:"

func (b BURI) String() string {
	return fmt.Sprintf("%s%s|%s|%d|%s", buriScheme, b.BlockID, b.TxID, b.Index, strings.Join(b.Branch, ","))
}

func ParseBURI(s string) (BURI, error) {
	var b BURI
	if !strings.HasPrefix(s, buriScheme) {
		return b, fmt.Errorf("not a BURI")
	}
	parts := strings.Split(strings.TrimPrefix(s, buriScheme), "|")
	if len(parts) != 4 {
		return b, fmt.Errorf("BURI must have 4 fields")
	}
	idx, err := strconv.Atoi(parts[2])
	if err != nil {
		return b, err
	}
	b.BlockID, b.TxID, b.Index = parts[0], parts[1], idx
	if parts[3] != "" {
		b.Branch = strings.Split(parts[3], ",")
	}
	return b, nil
}

// BuildBURI constructs a BURI for the tx at index in a block's leaves.
func BuildBURI(blockID string, leaves [][]byte, index int) (BURI, error) {
	branch, err := BranchFor(leaves, index)
	if err != nil {
		return BURI{}, err
	}
	hexBranch := make([]string, len(branch))
	for i, h := range branch {
		hexBranch[i] = hex.EncodeToString(h)
	}
	return BURI{BlockID: blockID, TxID: hex.EncodeToString(leaves[index]), Index: index, Branch: hexBranch}, nil
}

// Verify checks the BURI's tx is included under merkleRoot (the block-header Merkle root) — the SPV
// check a third party performs with only block headers (SYS-PROOF-005).
func (b BURI) Verify(merkleRoot []byte) (bool, error) {
	leaf, err := hex.DecodeString(b.TxID)
	if err != nil {
		return false, err
	}
	branch := make([][]byte, len(b.Branch))
	for i, hx := range b.Branch {
		branch[i], err = hex.DecodeString(hx)
		if err != nil {
			return false, err
		}
	}
	got := RootFromBranch(leaf, branch, b.Index)
	return hex.EncodeToString(got) == hex.EncodeToString(merkleRoot), nil
}
