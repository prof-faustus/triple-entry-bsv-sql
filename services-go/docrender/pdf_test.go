package docrender

import (
	"bytes"
	"strings"
	"testing"
)

func sample(docType string) Fields {
	return Fields{
		ObjectID:      "BL#1",
		DocType:       docType,
		State:         "ENDORSED",
		ControllerPub: "0292df7b245b81aa637ab4e867c8d511008f79161a97d64f2ac709600352f7acbc",
		Refs:          []string{"CON#1", "PO#1"},
		BURI:          "buri:block#100|9a3c…|1|ab12,cd34",
		BodyHashH4:    "deadbeef",
	}
}

func TestDeterministicByteStable(t *testing.T) {
	f := sample("invoice")
	a := RenderPDF(f)
	b := RenderPDF(f)
	if !bytes.Equal(a, b) {
		t.Fatal("PDF must be byte-stable for the same fields (SYS-DOC-003)")
	}
	if !bytes.HasPrefix(a, []byte("%PDF-1.4")) || !bytes.Contains(a, []byte("%%EOF")) {
		t.Fatal("not a well-formed PDF")
	}
}

func TestEmbedsVerifiableFields(t *testing.T) {
	f := sample("invoice")
	pdf := string(RenderPDF(f))
	for _, want := range []string{f.ObjectID, f.State, "buri:block#100", f.BodyHashH4, f.ControllerPub} {
		if !strings.Contains(pdf, want) {
			t.Fatalf("PDF must embed %q (SYS-DOC-002)", want)
		}
	}
}

func TestBillOfLadingMarkedNonNegotiable(t *testing.T) {
	bl := string(RenderPDF(sample("bill_of_lading")))
	if !strings.Contains(bl, "NOT A NEGOTIABLE ORIGINAL") || !strings.Contains(bl, "title held on-chain") {
		t.Fatal("B/L copy must be marked non-negotiable (SYS-DOC-004)")
	}
	// a non-B/L document must NOT carry the title-transfer disclaimer
	if strings.Contains(string(RenderPDF(sample("invoice"))), "NEGOTIABLE ORIGINAL") {
		t.Fatal("only the B/L should carry the negotiability marking")
	}
}

func TestBodyHashStableAndSelfConsistent(t *testing.T) {
	f := sample("proof_of_delivery")
	if f.BodyHash() != f.BodyHash() {
		t.Fatal("H3 must be deterministic")
	}
	// changing a field changes H3 (binds the rendered body)
	g := f
	g.State = "PENDING"
	if f.BodyHash() == g.BodyHash() {
		t.Fatal("H3 must bind the document state")
	}
}
