package edibridge

import "testing"

func TestX12InboundMapping(t *testing.T) {
	p := Partner{Name: "acme", Standard: X12, Enabled: true, Subset: []string{"850", "810", "820"}}
	cases := map[string]Action{
		"ST*850*0001~BEG*00*SA*PO123~SE*3*0001~": {"purchase_order", ""},
		"ST*810*0002~BIG*20260601*INV9~SE*3*0002~": {"invoice", ""},
		"ST*820*0003~BPR*C*1500~SE*3*0003~":      {"payment_note", "settle"},
	}
	for raw, want := range cases {
		got, err := Inbound(p, raw)
		if err != nil {
			t.Fatalf("inbound: %v", err)
		}
		if got != want {
			t.Errorf("X12 %q -> %+v, want %+v", raw, got, want)
		}
	}
}

func TestEdifactInboundMapping(t *testing.T) {
	p := Partner{Name: "globex", Standard: EDIFACT, Enabled: true, Subset: []string{"ORDERS", "INVOIC", "REMADV"}}
	cases := map[string]Action{
		"UNH+1+ORDERS:D:01B:UN'BGM+220+PO123'UNT+3+1'": {"purchase_order", ""},
		"UNH+1+INVOIC:D:01B:UN'BGM+380+INV9'UNT+3+1'":  {"invoice", ""},
		"UNH+1+REMADV:D:01B:UN'BGM+481+R1'UNT+3+1'":    {"payment_note", "settle"},
	}
	for raw, want := range cases {
		got, err := Inbound(p, raw)
		if err != nil {
			t.Fatalf("inbound: %v", err)
		}
		if got != want {
			t.Errorf("EDIFACT %q -> %+v, want %+v", raw, got, want)
		}
	}
}

func TestSubsetAndDisabled(t *testing.T) {
	// message type outside the partner subset is rejected
	p := Partner{Name: "acme", Standard: X12, Enabled: true, Subset: []string{"850"}}
	if _, err := Inbound(p, "ST*810*1~SE*1*1~"); err == nil {
		t.Fatal("expected rejection of 810 outside subset")
	}
	// disabled bridge is omittable — no translation occurs
	off := Partner{Name: "acme", Standard: X12, Enabled: false, Subset: []string{"850"}}
	if _, err := Inbound(off, "ST*850*1~SE*1*1~"); err == nil {
		t.Fatal("expected disabled bridge to refuse")
	}
}

func TestOutboundSerialise(t *testing.T) {
	// round-trip: an outbound invoice message must serialise to a type that maps back to "invoice".
	p := Partner{Name: "acme", Standard: X12, Enabled: true, Subset: []string{"810", "210"}}
	msg, err := Outbound(p, "invoice", "INV#1", "ISSUED")
	if err != nil {
		t.Fatal(err)
	}
	got, _ := DetectType(X12, msg)
	if x12Map[got].DocType != "invoice" {
		t.Fatalf("outbound X12 invoice serialised to %q which maps to %q (%s)", got, x12Map[got].DocType, msg)
	}
	pe := Partner{Name: "globex", Standard: EDIFACT, Enabled: true, Subset: []string{"INVOIC"}}
	emsg, err := Outbound(pe, "invoice", "INV#1", "ISSUED")
	if err != nil {
		t.Fatal(err)
	}
	if egot, _ := DetectType(EDIFACT, emsg); edifactMap[egot].DocType != "invoice" {
		t.Fatalf("outbound EDIFACT invoice serialised to %q (%s)", egot, emsg)
	}
}
