package edi

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
)

// DHT is the off-chain goods/document-body store (US11210372): a key-value table keyed by H2, whose
// value holds the record body, its hash, location, and the registered owner key. Records are
// header+body with the header carrying the body hash (SYS-LOG-010). Integrity is anchored on chain by
// H2 + H4 (GB2558485A): the on-chain Doc.BodyHashH4 commits to the body so later tampering is detected.
type DHT struct {
	store map[string]dhtEntry
}

type dhtEntry struct {
	H4       []byte // hash of the body (= H1), the integrity anchor read back on verify
	Body     []byte
	Location string
	OwnerPub []byte // registered owner/custodian key (US11210372)
}

func NewDHT() *DHT { return &DHT{store: map[string]dhtEntry{}} }

func h(b ...[]byte) []byte {
	s := sha256.New()
	for _, x := range b {
		s.Write(x)
	}
	return s.Sum(nil)
}

// Put stores a record body and returns (H2 key, H4 body-hash). H2 = hash representative of the details
// + location (US11210372); H4 = SHA-256(body) is the on-chain integrity anchor.
func (d *DHT) Put(body []byte, location string, ownerPub []byte) (h2 []byte, h4 []byte) {
	h4 = h(body)
	h2 = h(h4, []byte(location))
	d.store[hex.EncodeToString(h2)] = dhtEntry{H4: h4, Body: append([]byte{}, body...), Location: location, OwnerPub: ownerPub}
	return h2, h4
}

// VerifyIntegrity recomputes H3 over the current body and checks H3 == H4 (GB2558485A method 900).
func (d *DHT) VerifyIntegrity(h2 []byte) (bool, error) {
	e, ok := d.store[hex.EncodeToString(h2)]
	if !ok {
		return false, fmt.Errorf("no DHT entry for H2=%x", h2)
	}
	h3 := h(e.Body)
	return hex.EncodeToString(h3) == hex.EncodeToString(e.H4), nil
}

// UpdateOwner re-registers the owner/custodian after an on-chain custody transfer (SYS-LOG-007).
func (d *DHT) UpdateOwner(h2 []byte, ownerPub []byte) {
	k := hex.EncodeToString(h2)
	if e, ok := d.store[k]; ok {
		e.OwnerPub = ownerPub
		d.store[k] = e
	}
}

// Tamper mutates a stored body (for the adversarial integrity test).
func (d *DHT) Tamper(h2 []byte) {
	k := hex.EncodeToString(h2)
	if e, ok := d.store[k]; ok {
		e.Body = append(e.Body, 0x00)
		d.store[k] = e
	}
}

// VerifyOwnership checks the on-chain controller key matches the DHT-registered owner (PU2 == P2,
// US11210372): a match verifies current ownership/custody from chain + DHT, no intermediary.
func (d *DHT) VerifyOwnership(h2 []byte, onchainControllerPub []byte) (bool, error) {
	e, ok := d.store[hex.EncodeToString(h2)]
	if !ok {
		return false, fmt.Errorf("no DHT entry for H2=%x", h2)
	}
	return hex.EncodeToString(e.OwnerPub) == hex.EncodeToString(onchainControllerPub), nil
}

// AnchorMatches confirms the on-chain anchor (Doc.BodyHashH4) equals the DHT H4 (anchor binds body).
func (d *DHT) AnchorMatches(h2 []byte, onchainH4 []byte) (bool, error) {
	e, ok := d.store[hex.EncodeToString(h2)]
	if !ok {
		return false, fmt.Errorf("no DHT entry for H2=%x", h2)
	}
	return hex.EncodeToString(e.H4) == hex.EncodeToString(onchainH4), nil
}
