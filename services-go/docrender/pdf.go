// Package docrender produces deterministic, byte-stable PDF paper copies of commercial/logistics
// documents (Phase 5/9.6, SYS-DOC-001..005). The PDF is assembled from the document's on-chain DFA
// state + its SPV/BURI reference; it embeds object_id, state, controller, BURI, and the body integrity
// hash (so a holder can verify against block headers, SYS-DOC-002 / SYS-LOG-012). For a negotiable bill
// of lading the copy is marked non-negotiable — title stays on-chain (SYS-DOC-004). Rendering is
// deterministic (fixed template, fixed field order, no timestamps) so the same document at the same
// state yields byte-identical output (SYS-DOC-003).
package docrender

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"

	qrcode "github.com/skip2/go-qrcode"
)

// Fields are the inputs to a paper copy, gathered from chain + off-chain record.
type Fields struct {
	ObjectID      string
	DocType       string
	State         string
	ControllerPub string   // hex
	Refs          []string // cross-referenced object_ids
	BURI          string   // SPV reference (spv.BURI string)
	BodyHashH4    string   // hex of the on-chain-anchored body hash (integrity anchor)
}

// bodyLines is the canonical, ordered, timestamp-free body of the paper copy.
func (f Fields) bodyLines() []string {
	lines := []string{
		"TRIPLE-ENTRY BSV — VERIFIABLE DOCUMENT PAPER COPY",
		"Document type : " + f.DocType,
		"Object ID     : " + f.ObjectID,
		"State         : " + f.State,
		"Controller    : " + f.ControllerPub,
		"References    : " + strings.Join(f.Refs, ", "),
		"Body hash H4  : " + f.BodyHashH4,
		"BURI          : " + f.BURI,
		"Verify        : SPV-check the BURI against BSV block headers; recompute H3 over this body and",
		"                confirm H3 == H4 (GB2558485A); the on-chain state is the source of truth.",
	}
	if f.DocType == "bill_of_lading" {
		lines = append(lines,
			"",
			"COPY — title held on-chain; current holder verifiable via the embedded BURI.",
			"THIS PDF IS NOT A NEGOTIABLE ORIGINAL AND DOES NOT TRANSFER TITLE (SYS-DOC-004).")
	}
	return lines
}

// BodyHash is H3 over the canonical body (recomputed by a verifier; must equal the anchored H4).
func (f Fields) BodyHash() string {
	h := sha256.Sum256([]byte(strings.Join(f.bodyLines(), "\n")))
	return hex.EncodeToString(h[:])
}

// qrOps renders a deterministic QR of `content` as filled PDF rectangles (vector, no image embed) so a
// verifier can scan the BURI off the paper copy (SYS-DOC-002). Returns "" if encoding fails.
func qrOps(content string) string {
	q, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return ""
	}
	bm := q.Bitmap() // [][]bool incl. quiet zone; deterministic for fixed content+level
	const cell = 3.0
	const ox, oy = 410.0, 770.0 // top-right; draw downward
	var s strings.Builder
	s.WriteString("0 0 0 rg\n")
	for r := range bm {
		for c := range bm[r] {
			if bm[r][c] {
				x := ox + float64(c)*cell
				y := oy - float64(r)*cell
				fmt.Fprintf(&s, "%.0f %.0f %.0f %.0f re f\n", x, y, cell, cell)
			}
		}
	}
	return s.String()
}

func esc(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "(", "\\(")
	s = strings.ReplaceAll(s, ")", "\\)")
	return s
}

// RenderPDF builds a deterministic single-page PDF. No /CreationDate, fixed object order/offsets, so
// the same Fields always produce byte-identical output (SYS-DOC-003).
func RenderPDF(f Fields) []byte {
	// content stream: draw each body line top-to-bottom in Helvetica 11pt.
	var content strings.Builder
	content.WriteString("BT /F1 11 Tf 50 800 Td 14 TL\n")
	for i, ln := range f.bodyLines() {
		if i == 0 {
			content.WriteString("(" + esc(ln) + ") Tj\n")
		} else {
			content.WriteString("T* (" + esc(ln) + ") Tj\n")
		}
	}
	content.WriteString("ET\n")
	content.WriteString(qrOps(f.BURI)) // scannable QR of the BURI (SYS-DOC-002)
	cs := content.String()

	objs := []string{
		"<</Type/Catalog/Pages 2 0 R>>",
		"<</Type/Pages/Kids[3 0 R]/Count 1>>",
		"<</Type/Page/Parent 2 0 R/MediaBox[0 0 595 842]/Resources<</Font<</F1 4 0 R>>>>/Contents 5 0 R>>",
		"<</Type/Font/Subtype/Type1/BaseFont/Helvetica>>",
		"<</Length " + strconv.Itoa(len(cs)) + ">>\nstream\n" + cs + "endstream",
	}

	var b strings.Builder
	b.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objs)+1)
	for i, o := range objs {
		offsets[i+1] = b.Len()
		b.WriteString(fmt.Sprintf("%d 0 obj\n%s\nendobj\n", i+1, o))
	}
	xref := b.Len()
	b.WriteString(fmt.Sprintf("xref\n0 %d\n", len(objs)+1))
	b.WriteString("0000000000 65535 f \n")
	for i := 1; i <= len(objs); i++ {
		b.WriteString(fmt.Sprintf("%010d 00000 n \n", offsets[i]))
	}
	b.WriteString(fmt.Sprintf("trailer\n<</Size %d/Root 1 0 R>>\nstartxref\n%d\n%%%%EOF\n", len(objs)+1, xref))
	return []byte(b.String())
}
