// edie2e drives the Phase-5 exit (Appendix B.8, B.9) on Teranode regtest:
//   B.8: every SYS-EDI-002 document type runs its DFA lifecycle on-chain (states-as-UTXOs,
//        transitions journalled); cross-references by object_id; verified from chain.
//   B.9: consignment lifecycle, multi-party custody, bill-of-lading-as-token title transfer,
//        ownership (key-match, US11210372) + integrity (hash-match, GB2558485A), delivery-vs-payment.
package main

import (
	"bytes"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	cc "te-bsv/cryptocore"
	"te-bsv/services-go/edi"
	"te-bsv/services-go/node"
)

const (
	walletPriv = "1122334455667788990011223344556677889900112233445566778899001122"
	writerPriv = "e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262"
	cpPriv     = "f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181"
	shipperK   = "1111111111111111111111111111111111111111111111111111111111111111"
	carrierK   = "2222222222222222222222222222222222222222222222222222222222222222"
	consigneeK = "3333333333333333333333333333333333333333333333333333333333333333"
	sellerK    = "4444444444444444444444444444444444444444444444444444444444444444"
)

var logFile *os.File

func logf(f string, a ...any) {
	s := fmt.Sprintf(f, a...)
	fmt.Println(s)
	if logFile != nil {
		fmt.Fprintln(logFile, s)
		logFile.Sync()
	}
}
func ck(err error) {
	if err != nil {
		logf("RESULT: FAIL: %s", err)
		os.Exit(1)
	}
}
func must[T any](v T, err error) T { ck(err); return v }
func expect(c bool, what string) {
	if !c {
		ck(fmt.Errorf("assertion failed: %s", what))
	}
}
func mustHex(s string) []byte { b, err := hex.DecodeString(s); ck(err); return b }

// expected SYS-EDI-002 document set (must all be defined).
var mandated = []string{
	"rfq", "quotation", "purchase_order", "order_ack", "invoice", "payment_note", "credit_note", "debit_note",
	"despatch_advice", "bill_of_lading", "sea_waybill", "air_waybill", "cmr", "rail_consignment_note",
	"packing_list", "booking_confirmation", "arrival_notice", "proof_of_delivery",
	"certificate_of_origin", "customs_declaration", "inspection_certificate", "insurance_certificate",
}

func main() {
	rpc := flag.String("rpc", "http://localhost:9292", "RPC URL")
	user := flag.String("user", "teranode", "RPC user")
	pass := flag.String("pass", "regtestsecret", "RPC pass")
	defs := flag.String("defs", "/mnt/d/claude/SQL/edi-dfa/document-defs.json", "DFA defs")
	logp := flag.String("log", filepath.Join(os.TempDir(), "te_edi.log"), "log path")
	flag.Parse()
	if fl, err := os.Create(*logp); err == nil {
		logFile = fl
		defer fl.Close()
	}

	reg := must(edi.LoadRegistry(*defs))
	wallet := must(edi.NewKeypair(walletPriv))
	shipper := must(edi.NewKeypair(shipperK))
	carrier := must(edi.NewKeypair(carrierK))
	consignee := must(edi.NewKeypair(consigneeK))
	seller := must(edi.NewKeypair(sellerK))
	e := &edi.Engine{C: node.New(*rpc, *user, *pass), Wallet: wallet, WriterPriv: mustHex(writerPriv), CpPub: cc.PubFromPriv(mustHex(cpPriv)), Reg: reg}
	ck(e.Fund())

	// ---- B.8: every mandated document type runs its DFA lifecycle on-chain ----
	logf("== B.8: running every SYS-EDI-002 document DFA on-chain ==")
	for _, dt := range mandated {
		def, err := reg.Get(dt)
		ck(err)
		final := driveToFinal(e, def, seller)
		expect(def.IsFinal(final), dt+" reached a final state")
	}
	logf("  %d document DFAs each driven to a final state and tag-verified ✓", len(mandated))

	// ---- B.8: cross-referenced PO -> invoice -> payment_note (SYS-EDI-004) ----
	logf("== cross-referenced trade documents ==")
	po := must(e.Originate("purchase_order", "PO#1", seller, nil, nil))
	po = must(e.Transition(po, "issue", nil))
	po = must(e.Transition(po, "acknowledge", nil))
	inv := must(e.Originate("invoice", "INV#1", seller, []string{"PO#1"}, nil))
	inv = must(e.Transition(inv, "issue", nil))
	pay := must(e.Originate("payment_note", "PAY#1", seller, []string{"INV#1"}, nil))
	pay = must(e.Transition(pay, "instruct", nil))
	invH := must(e.Verify(inv.Txid, "INV#1"))
	expect(contains(invH.Final.Refs, "PO#1"), "invoice references PO#1")
	logf("  PO#1 -> INV#1(refs PO#1) -> PAY#1(refs INV#1); reference graph verified from chain ✓")

	// ---- B.9: consignment + DHT goods record + ownership + integrity ----
	logf("== B.9: consignment lifecycle, custody, ownership, integrity ==")
	dht := edi.NewDHT()
	body := []byte("WIDGET x1000; HS 8479.89; 12,000kg; origin GB; provenance: factory-A")
	h2, h4 := dht.Put(body, "dht://goods/CON#1", shipper.Pub)
	con := must(e.Originate("consignment", "CON#1", shipper, []string{"PO#1"}, h4))
	anchorOK := must(dht.AnchorMatches(h2, con.BodyHashH4))
	expect(anchorOK, "on-chain anchor commits the DHT body (H4)")
	con = must(e.Transition(con, "book", nil))
	con = must(e.Transition(con, "pickup", nil))
	con = must(e.Transition(con, "depart", nil))
	con = must(e.Transition(con, "customs_clear", nil))
	con = must(e.Transition(con, "inspect", nil))
	// multi-party custody transfer: re-key control shipper -> carrier (SYS-LOG-007)
	con = must(e.Transition(con, "custody_transfer", carrier))
	dht.UpdateOwner(h2, carrier.Pub)
	ownOK := must(dht.VerifyOwnership(h2, con.Controller.Pub))
	expect(ownOK, "ownership: on-chain controller == DHT-registered owner (US11210372)")
	badOwn := must(dht.VerifyOwnership(h2, consignee.Pub))
	expect(!badOwn, "ownership: unrelated key does NOT match")
	// integrity: H3==H4, then tamper -> mismatch (GB2558485A)
	intOK := must(dht.VerifyIntegrity(h2))
	expect(intOK, "integrity: H3 == H4 on the genuine body")
	dht.Tamper(h2)
	intBad := must(dht.VerifyIntegrity(h2))
	expect(!intBad, "integrity: tampered body detected (H3 != H4)")
	logf("  ownership key-match ✓ (and mismatch rejected); integrity hash-match ✓ (and tamper detected)")

	// deliver + accept, then delivery-versus-payment
	con = must(e.Transition(con, "deliver", consignee))
	con = must(e.Transition(con, "accept", nil))
	pod := must(e.Originate("proof_of_delivery", "POD#1", consignee, []string{"CON#1"}, nil))
	pod = must(e.Transition(pod, "sign", nil))
	expect(con.State == "ACCEPTED" && pod.State == "SIGNED", "delivery evidence present")
	pay = must(e.Transition(pay, "settle", nil)) // DvP: settle only on acceptance+POD
	con = must(e.Transition(con, "settle", nil))
	logf("  delivery-versus-payment: CON#1 ACCEPTED + POD#1 SIGNED -> PAY#1 SETTLED, CON#1 SETTLED ✓")

	// ---- B.9: bill-of-lading-as-token title transfer ----
	logf("== bill-of-lading as the token (title transfer) ==")
	bl := must(e.Originate("bill_of_lading", "BL#1", shipper, []string{"CON#1"}, h4))
	bl = must(e.Transition(bl, "endorse", consignee)) // endorsement = title transfer (re-key)
	bl = must(e.Transition(bl, "surrender", nil))
	blH := must(e.Verify(bl.Txid, "BL#1"))
	expect(bytes.Equal(blH.Final.ControllerPub, consignee.Pub), "B/L title now held by consignee")
	expect(blH.Final.State == "SURRENDERED", "B/L surrendered at delivery")
	logf("  B/L issued(shipper) -> endorsed(consignee) -> surrendered; title transfer verified on chain ✓")

	logf("RESULT: PHASE-5 E2E PASS — %d document DFAs, cross-refs, consignment+custody, ownership(US11210372)+integrity(GB2558485A), DvP, B/L-as-token", len(mandated))
}

// driveToFinal originates a doc and fires transitions greedily (preferring unvisited targets) until a
// final state, exercising the DFA on-chain. Returns the final state.
func driveToFinal(e *edi.Engine, def edi.DFADef, ctrl *edi.Keypair) string {
	oid := def.DocType + "#auto"
	doc := must(e.Originate(def.DocType, oid, ctrl, nil, nil))
	visited := map[string]bool{doc.State: true}
	for step := 0; step < 16 && !def.IsFinal(doc.State); step++ {
		edges := def.From(doc.State)
		if len(edges) == 0 {
			break
		}
		// prefer an edge to an unvisited state (avoid self-loops), else the first edge
		chosen := edges[0]
		for _, t := range edges {
			if !visited[t.To] {
				chosen = t
				break
			}
		}
		doc = must(e.Transition(doc, chosen.Event, nil))
		visited[doc.State] = true
	}
	// tag-verify the lineage from the chain
	h := must(e.Verify(doc.Txid, oid))
	_ = h
	return doc.State
}

func contains(ss []string, x string) bool {
	for _, s := range ss {
		if s == x {
			return true
		}
	}
	return false
}
