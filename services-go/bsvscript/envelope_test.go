package bsvscript

import (
	"bytes"
	"encoding/hex"
	"testing"

	"github.com/bsv-blockchain/go-sdk/script"
	cc "te-bsv/cryptocore"
)

// buildRealRecord produces an on-chain field record (REC) via the crypto core,
// so the envelope tests carry a genuine ECDH-HMAC third-entry payload.
func buildRealRecord(t *testing.T) []byte {
	t.Helper()
	wPriv, _ := hex.DecodeString("e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262")
	cPriv, _ := hex.DecodeString("f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181")
	cPub := cc.PubFromPriv(cPriv)
	m := cc.ChangeMessage{TableID: "ledger.invoices", RowID: []byte{0x00, 0x01}, ColumnID: "amount", Op: cc.OpInsert, Seq: 0}
	_, gv, err := cc.GeneratorValue(m)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := cc.CommonSecretAsWriter(wPriv, cPub, gv)
	if err != nil {
		t.Fatal(err)
	}
	k, err := cc.DeriveHMACKey(cs, m)
	if err != nil {
		t.Fatal(err)
	}
	img := cc.ChangeImage(cc.ImagePlaintext, []byte("1500.00"), nil)
	rec, err := cc.EncodeRecord(cc.FieldRecord{StreamID: []byte(m.TableID), Seq: m.Seq, PrevTxid: m.PrevTxid, ImageKind: cc.ImagePlaintext, ChangeImage: img, Tag: cc.Tag(k, img)})
	if err != nil {
		t.Fatal(err)
	}
	return rec
}

func pkh20() []byte {
	p := make([]byte, 20)
	for i := range p {
		p[i] = byte(i + 1)
	}
	return p
}

func TestEnvelopeIfRoundTripAndSafety(t *testing.T) {
	rec := buildRealRecord(t)
	pkh := pkh20()
	s, err := BuildEnvelopeIf(rec, pkh)
	if err != nil {
		t.Fatal(err)
	}
	if err := AssertNativeSpendable(s); err != nil {
		t.Fatalf("envelope must be native+spendable: %v", err)
	}
	if s.IsP2SH() {
		t.Fatal("must not be P2SH")
	}
	got, err := ExtractEnvelopeData(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, rec) {
		t.Fatalf("payload round-trip mismatch: got %x want %x", got, rec)
	}
	// the recovered REC must decode back to the same field record
	if _, err := cc.DecodeRecord(got); err != nil {
		t.Fatalf("recovered REC does not decode: %v", err)
	}
	// spend tail is native P2PKH carrying our pkh
	if h, err := EnvelopePubKeyHash(s); err != nil || !bytes.Equal(h, pkh) {
		t.Fatalf("expected P2PKH tail with our pkh; err=%v h=%x", err, h)
	}
}

func TestEnvelopeDropRoundTripAndSafety(t *testing.T) {
	rec := buildRealRecord(t)
	pkh := pkh20()
	s, err := BuildEnvelopeDrop(rec, pkh)
	if err != nil {
		t.Fatal(err)
	}
	if err := AssertNativeSpendable(s); err != nil {
		t.Fatalf("drop envelope must be native+spendable: %v", err)
	}
	got, err := ExtractEnvelopeData(s)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, rec) {
		t.Fatalf("drop payload round-trip mismatch")
	}
}

// Appendix B.5 static check: P2SH and OP_RETURN scripts must be rejected.
func TestAssertRejectsForbidden(t *testing.T) {
	// P2SH: OP_HASH160 <20> OP_EQUAL
	p2sh := &script.Script{}
	_ = p2sh.AppendOpcodes(script.OpHASH160)
	_ = p2sh.AppendPushData(pkh20())
	_ = p2sh.AppendOpcodes(script.OpEQUAL)
	if !p2sh.IsP2SH() {
		t.Fatal("sanity: expected IsP2SH true")
	}
	if err := AssertNativeSpendable(p2sh); err == nil {
		t.Fatal("must reject P2SH")
	}

	// OP_RETURN data output
	ret := &script.Script{}
	_ = ret.AppendOpcodes(script.OpFALSE, script.OpRETURN)
	_ = ret.AppendPushData([]byte("data"))
	if err := AssertNativeSpendable(ret); err == nil {
		t.Fatal("must reject OP_RETURN")
	}

	// data-only (no spend tail) must be rejected as not spendable
	dataOnly := &script.Script{}
	_ = dataOnly.AppendPushData([]byte("just data"))
	_ = dataOnly.AppendOpcodes(script.OpDROP)
	if err := AssertNativeSpendable(dataOnly); err == nil {
		t.Fatal("must reject non-spendable data-only script")
	}
}
