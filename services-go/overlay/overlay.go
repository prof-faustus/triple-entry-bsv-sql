// Package overlay implements overlay addressing via a child-key-derivation hierarchy whose key
// structure mirrors the overlay graph (Phase 6, SYS-OVL-*; EP4046048B1). Streams, document graphs, and
// token lineages are nodes; each node carries a key (deriving its children), giving deterministic,
// structure-aligned addressing of every entity from a seed.
package overlay

import (
	"crypto/hmac"
	"crypto/sha512"
	"encoding/binary"
	"errors"
	"math/big"

	cc "te-bsv/cryptocore"
)

var nOrder, _ = new(big.Int).SetString("fffffffffffffffffffffffffffffffebaaedce6af48a03bbfd25e8cd0364141", 16)

// Node is an overlay node's key material (BIP32-style: private scalar + chain code + derived pubkey).
type Node struct {
	Priv  *big.Int
	Chain []byte
	Pub   []byte // compressed
}

// Master derives the root node from a seed.
func Master(seed []byte) (*Node, error) {
	mac := hmac.New(sha512.New, []byte("TE/overlay/v1"))
	mac.Write(seed)
	I := mac.Sum(nil)
	priv := new(big.Int).Mod(new(big.Int).SetBytes(I[:32]), nOrder)
	if priv.Sign() == 0 {
		return nil, errors.New("degenerate seed")
	}
	return node(priv, I[32:])
}

func node(priv *big.Int, chain []byte) (*Node, error) {
	var b [32]byte
	priv.FillBytes(b[:])
	return &Node{Priv: priv, Chain: chain, Pub: cc.PubFromPriv(b[:])}, nil
}

// Derive returns the child node at `index` (non-hardened CKD): the parent key signs/derives the child,
// mirroring an overlay edge (EP4046048B1 claim 1).
func (n *Node) Derive(index uint32) (*Node, error) {
	mac := hmac.New(sha512.New, n.Chain)
	mac.Write(n.Pub)
	var idx [4]byte
	binary.BigEndian.PutUint32(idx[:], index)
	mac.Write(idx[:])
	I := mac.Sum(nil)
	il := new(big.Int).SetBytes(I[:32])
	if il.Cmp(nOrder) >= 0 {
		return nil, errors.New("derive: IL >= n, pick another index")
	}
	childPriv := new(big.Int).Add(n.Priv, il)
	childPriv.Mod(childPriv, nOrder)
	if childPriv.Sign() == 0 {
		return nil, errors.New("derive: zero child key")
	}
	return node(childPriv, I[32:])
}

// DerivePath derives a node by an index path that mirrors the entity's position in the overlay graph
// (e.g. [ledgerStream, document, transition]).
func (n *Node) DerivePath(path ...uint32) (*Node, error) {
	cur := n
	for _, idx := range path {
		c, err := cur.Derive(idx)
		if err != nil {
			return nil, err
		}
		cur = c
	}
	return cur, nil
}
