// Package token implements the definable token primitive (Phase 4), grounded in EP3748903A1,
// re-expressed in native BSV (no P2SH, no OP_RETURN). A token is a UTXO lineage whose state carries
// token metadata + an ECDH-HMAC third-entry record in a SPENDABLE locking script. Cash, CBDC/stablecoin-
// linked tokens, and goods are instances of one schema (SYS-TOK-005). Every event journals (SYS-CASH-002).
package token

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	cc "te-bsv/cryptocore"
)

// Backing models (SYS-CASH-001).
type Backing string

const (
	IssuerBacked Backing = "issuer-backed" // claim on off-chain value; redeem burns
	SatoshiTagged Backing = "satoshi-tagged" // denominates underlying BSV value
	Pegged        Backing = "pegged"        // tracks an external unit via oracle (+ custodian)
)

// SupplyPolicy.
type Supply string

const (
	Fixed       Supply = "fixed"
	MintOnDemand Supply = "mint-on-demand"
	Burnable     Supply = "burnable"
)

// Definition is the issuer-defined token type (SYS-TOK-005). A new type is created purely as data
// (e.g. a JSON row) — no code change. `External` declares an adapter linkage (SYS-TOK-006).
type Definition struct {
	TypeID          string `json:"type_id"`
	Label           string `json:"label"`
	Unit            string `json:"unit"`     // fiat code / CBDC / crypto unit / SKU / point
	Decimals        int    `json:"decimals"` // divisibility
	Supply          Supply `json:"supply"`
	Backing         Backing `json:"backing"`
	PeggingRateMicro uint64 `json:"pegging_rate_micro"` // sat per token minor-unit, ×1e6 (SYS-TOK-003)
	MinSatThreshold uint64 `json:"min_sat_threshold"`   // SYS-TOK-003 claim 10
	Confidential    bool   `json:"confidential"`
	External        *ExternalLink `json:"external,omitempty"` // SYS-TOK-006 adapter linkage (contract only)
}

// ExternalLink declares a peg/back or bridge to an external rail. The rail's own interface is OUT OF
// SCOPE beyond this adapter contract; no real rail is integrated without its actual interface + go-ahead.
type ExternalLink struct {
	Mode      string `json:"mode"`      // "peg" | "bridge"
	Adapter   string `json:"adapter"`   // adapter name (e.g. "cbdc-egbp", "stablecoin-usdx")
	Oracle    string `json:"oracle"`    // named rate oracle (peg)
	Custodian string `json:"custodian"` // named backing custodian (peg, where backed)
}

// SatoshiQuantity binds token value to satoshis: B1 = f(TV1, PR1), with a minimum threshold
// (EP3748903A1 claims 9–10; SYS-TOK-003). value is in token minor units.
func (d Definition) SatoshiQuantity(value uint64) uint64 {
	q := value * d.PeggingRateMicro / 1_000_000
	if q < d.MinSatThreshold {
		q = d.MinSatThreshold
	}
	if q == 0 {
		q = 1 // never an unspendable zero-value output
	}
	return q
}

// EventKind for the journalled token event.
type EventKind uint8

const (
	Mint     EventKind = 1
	Transfer EventKind = 2
	Redeem   EventKind = 3
)

// EncodeEvent is the canonical token-event payload carried as the third-entry change_image.
func EncodeEvent(d Definition, value uint64, controllerPub []byte, kind EventKind) []byte {
	backing := map[Backing]byte{IssuerBacked: 0, SatoshiTagged: 1, Pegged: 2}[d.Backing]
	ext := byte(0)
	if d.External != nil {
		ext = 1
	}
	return cc.NewWriter().
		Str(d.TypeID).
		Str(d.Unit).
		U8(backing).
		U8(byte(d.Decimals)).
		U64(value).
		U64(d.PeggingRateMicro).
		Bytes(controllerPub).
		U8(ext).
		U8(byte(kind)).
		Finish()
}

// Event is a decoded token event (inverse of EncodeEvent).
type Event struct {
	TypeID        string
	Unit          string
	BackingByte   byte
	Decimals      byte
	Value         uint64
	PeggingRate   uint64
	ControllerPub []byte
	HasExternal   bool
	Kind          EventKind
}

// DecodeEvent parses a token-event change_image (used in cold-rebuild / verification).
func DecodeEvent(img []byte) (Event, error) {
	var e Event
	rd := cc.NewReader(img)
	e.TypeID = rd.Str()
	e.Unit = rd.Str()
	e.BackingByte = rd.U8()
	e.Decimals = rd.U8()
	e.Value = rd.U64()
	e.PeggingRate = rd.U64()
	e.ControllerPub = rd.Bytes()
	e.HasExternal = rd.U8() == 1
	e.Kind = EventKind(rd.U8())
	if err := rd.End(); err != nil {
		return e, err
	}
	return e, nil
}

// Op maps a token event to the journalling op (mint=INSERT, transfer=UPDATE, redeem=DELETE).
func (k EventKind) Op() cc.Op {
	switch k {
	case Mint:
		return cc.OpInsert
	case Transfer:
		return cc.OpUpdate
	default:
		return cc.OpDelete
	}
}

// Registry is a set of token definitions loaded from data (no code change to add a type).
type Registry map[string]Definition

func LoadRegistry(path string) (Registry, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var defs []Definition
	if err := json.Unmarshal(b, &defs); err != nil {
		return nil, err
	}
	r := Registry{}
	for _, d := range defs {
		if d.TypeID == "" {
			return nil, errors.New("token def missing type_id")
		}
		r[d.TypeID] = d
	}
	return r, nil
}

func (r Registry) Get(typeID string) (Definition, error) {
	d, ok := r[typeID]
	if !ok {
		return Definition{}, fmt.Errorf("unknown token type %q", typeID)
	}
	return d, nil
}

// ---- External-rail adapter CONTRACT (SYS-TOK-006) — interface only ----

// RailAdapter is the contract an external rail (CBDC/stablecoin/other-coin) must satisfy to be linked.
// Implementations integrating a REAL rail are gated behind explicit go-ahead (STOP-AND-ASK) and are NOT
// provided here. On-chain, the linked token remains the native definable token of SYS-TOK-005.
type RailAdapter interface {
	// RateMicro returns the current peg rate (sat per token minor-unit ×1e6) from the named oracle.
	RateMicro() (uint64, error)
	// Lock reserves `value` on the external side against an on-chain mint (bridge mode); returns a ref.
	Lock(value uint64, ref string) (string, error)
	// Release frees the external-side reservation on redeem/burn.
	Release(extRef string) error
}

// MockAdapter is a test-only stand-in (no real rail). It echoes a fixed oracle rate; Lock/Release are
// no-ops returning deterministic refs. Used only to exercise the on-chain side of SYS-TOK-006.
type MockAdapter struct {
	Name      string
	FixedRate uint64
}

func (m MockAdapter) RateMicro() (uint64, error)           { return m.FixedRate, nil }
func (m MockAdapter) Lock(v uint64, ref string) (string, error) { return "mock:" + m.Name + ":" + ref, nil }
func (m MockAdapter) Release(extRef string) error          { return nil }
