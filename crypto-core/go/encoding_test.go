package cryptocore

import (
	"bytes"
	"testing"
)

func TestWriterReaderRoundTrip(t *testing.T) {
	buf := NewWriter().U8(1).U16(258).U32(0xdeadbeef).U64(0x0102030405060708).Bytes([]byte("hi")).Str("héllo").Finish()
	rd := NewReader(buf)
	if rd.U8() != 1 || rd.U16() != 258 || rd.U32() != 0xdeadbeef || rd.U64() != 0x0102030405060708 {
		t.Fatal("primitive mismatch")
	}
	if !bytes.Equal(rd.Bytes(), []byte("hi")) {
		t.Fatal("bytes mismatch")
	}
	if rd.Str() != "héllo" {
		t.Fatal("str mismatch")
	}
	if err := rd.End(); err != nil {
		t.Fatalf("end: %v", err)
	}
}

func TestReaderRejections(t *testing.T) {
	if rd := NewReader([]byte{0x00}); func() bool { rd.U32(); return rd.Err() == nil }() {
		t.Fatal("expected truncation error")
	}
	// trailing bytes
	rd := NewReader([]byte{7, 0xff})
	rd.U8()
	if err := rd.End(); err == nil {
		t.Fatal("expected trailing-bytes error")
	}
	// length prefix overruns buffer
	rd2 := NewReader([]byte{0, 0, 0, 10, 1})
	rd2.Bytes()
	if rd2.Err() == nil {
		t.Fatal("expected truncation on oversized length")
	}
	// invalid utf-8
	rd3 := NewReader([]byte{0, 0, 0, 1, 0xff})
	rd3.Str()
	if rd3.Err() == nil {
		t.Fatal("expected invalid utf-8 error")
	}
}

func TestRecordRoundTripAndRejections(t *testing.T) {
	rec := FieldRecord{
		StreamID:    []byte("ledger.invoices"),
		Seq:         7,
		PrevTxid:    bytes.Repeat([]byte{0xab}, 32),
		ImageKind:   ImageCommitment,
		ChangeImage: bytes.Repeat([]byte{0xcd}, 32),
		Tag:         bytes.Repeat([]byte{0xef}, 32),
	}
	enc, err := EncodeRecord(rec)
	if err != nil {
		t.Fatal(err)
	}
	dec, err := DecodeRecord(enc)
	if err != nil {
		t.Fatal(err)
	}
	if dec.Seq != 7 || dec.ImageKind != ImageCommitment {
		t.Fatal("decoded fields mismatch")
	}
	reenc, _ := EncodeRecord(dec)
	if !bytes.Equal(reenc, enc) {
		t.Fatal("re-encode mismatch")
	}
	// bad magic
	bad := append([]byte{}, enc...)
	bad[0] = 0x58
	if _, err := DecodeRecord(bad); err == nil {
		t.Fatal("expected bad-magic error")
	}
	// trailing bytes
	if _, err := DecodeRecord(append(append([]byte{}, enc...), 0x00)); err == nil {
		t.Fatal("expected trailing-bytes error")
	}
}

func TestEncodeMessageRejectsBadPrevTxid(t *testing.T) {
	_, err := EncodeMessage(ChangeMessage{TableID: "t", ColumnID: "c", Op: OpInsert, Seq: 0, PrevTxid: make([]byte, 5)})
	if err == nil {
		t.Fatal("expected prevTxid length error")
	}
}
