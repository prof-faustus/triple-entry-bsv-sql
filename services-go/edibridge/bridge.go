// Package edibridge is the OPTIONAL, per-partner X12/EDIFACT <-> on-chain DFA translation layer
// (SYS-EDI-005/006). It is a PURE translation layer: inbound it parses a standard message and resolves
// the corresponding DFA (doc type + transition event); outbound it serialises current on-chain state
// into a standard message. It holds no authority the chain does not confirm, is enabled/configured per
// partner, and is omittable — the core `edi` DFA engine does not import this package.
package edibridge

import (
	"fmt"
	"strings"
)

type Standard string

const (
	X12     Standard = "X12"
	EDIFACT Standard = "EDIFACT"
)

// Partner is the per-partner bridge configuration (SYS-EDI-006).
type Partner struct {
	Name     string
	Standard Standard
	Enabled  bool
	Subset   []string // permitted message types (e.g. "850","810" or "ORDERS","INVOIC")
}

func (p Partner) allows(msgType string) bool {
	for _, m := range p.Subset {
		if m == msgType {
			return true
		}
	}
	return false
}

// Action is the resolved DFA action for an inbound message. Event=="" means originate (create in the
// document's initial state); otherwise it is a transition event.
type Action struct {
	DocType string
	Event   string
}

// mapping: standard message type -> DFA action (SYS-EDI-005 listed message set).
var x12Map = map[string]Action{
	"850": {"purchase_order", ""},        // PO
	"855": {"purchase_order", "acknowledge"}, // PO acknowledgement
	"856": {"despatch_advice", ""},       // ASN / despatch
	"810": {"invoice", ""},               // invoice
	"820": {"payment_note", "settle"},    // payment / remittance
	"214": {"consignment", "depart"},     // transport status
	"990": {"booking_confirmation", "confirm"}, // freight response
	"210": {"invoice", ""},               // freight invoice
}

var edifactMap = map[string]Action{
	"ORDERS": {"purchase_order", ""},
	"ORDRSP": {"order_ack", ""},
	"DESADV": {"despatch_advice", ""},
	"INVOIC": {"invoice", ""},
	"REMADV": {"payment_note", "settle"},
	"IFTMIN": {"booking_confirmation", ""},
	"IFTSTA": {"consignment", "depart"},
	"PAYORD": {"payment_note", "instruct"},
}

// DetectType extracts the message type from a raw X12 or EDIFACT message.
func DetectType(std Standard, raw string) (string, error) {
	switch std {
	case X12:
		// segments end with ~, elements separated by *; type is the ST segment's second element.
		for _, seg := range strings.Split(raw, "~") {
			el := strings.Split(strings.TrimSpace(seg), "*")
			if len(el) >= 2 && el[0] == "ST" {
				return el[1], nil
			}
		}
		return "", fmt.Errorf("X12: no ST segment")
	case EDIFACT:
		// segments end with ', elements by +, components by :; type is in UNH's 3rd element, 1st comp.
		for _, seg := range strings.Split(raw, "'") {
			el := strings.Split(strings.TrimSpace(seg), "+")
			if len(el) >= 3 && el[0] == "UNH" {
				return strings.Split(el[2], ":")[0], nil
			}
		}
		return "", fmt.Errorf("EDIFACT: no UNH segment")
	}
	return "", fmt.Errorf("unknown standard %q", std)
}

// Inbound parses a standard message for a partner and resolves the DFA action to drive (SYS-EDI-006).
func Inbound(p Partner, raw string) (Action, error) {
	if !p.Enabled {
		return Action{}, fmt.Errorf("bridge disabled for partner %s", p.Name)
	}
	mt, err := DetectType(p.Standard, raw)
	if err != nil {
		return Action{}, err
	}
	if !p.allows(mt) {
		return Action{}, fmt.Errorf("message type %s not in partner %s subset", mt, p.Name)
	}
	var m map[string]Action
	if p.Standard == X12 {
		m = x12Map
	} else {
		m = edifactMap
	}
	a, ok := m[mt]
	if !ok {
		return Action{}, fmt.Errorf("no mapping for %s/%s", p.Standard, mt)
	}
	return a, nil
}

// Outbound serialises current on-chain document state into a standard message for the partner
// (SYS-EDI-006 outbound). Representative envelope; real field-level mapping is per message spec.
func Outbound(p Partner, docType, objectID, state string) (string, error) {
	if !p.Enabled {
		return "", fmt.Errorf("bridge disabled for partner %s", p.Name)
	}
	mt := reverseLookup(p.Standard, docType)
	if mt == "" {
		return "", fmt.Errorf("no %s message type for doc %s", p.Standard, docType)
	}
	switch p.Standard {
	case X12:
		return fmt.Sprintf("ST*%s*0001~REF*OI*%s~STATUS*%s~SE*3*0001~", mt, objectID, state), nil
	default:
		return fmt.Sprintf("UNH+1+%s:D:01B:UN'RFF+OI:%s'STS+%s'UNT+3+1'", mt, objectID, state), nil
	}
}

func reverseLookup(std Standard, docType string) string {
	m := x12Map
	if std == EDIFACT {
		m = edifactMap
	}
	best := ""
	for mt, a := range m {
		if a.DocType == docType {
			if best == "" || mt < best {
				best = mt // deterministic pick
			}
		}
	}
	return best
}
