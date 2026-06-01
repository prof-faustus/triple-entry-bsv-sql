package cryptocore

import "errors"

var (
	MagicM  = []byte("TEMC") // Triple-Entry Message Canonical
	MagicR  = []byte("TER1") // Triple-Entry Record v1
	Version = byte(1)
)

type Op uint8

const (
	OpInsert Op = 1
	OpUpdate Op = 2
	OpDelete Op = 3
)

type ImageKind uint8

const (
	ImagePlaintext  ImageKind = 0
	ImageCommitment ImageKind = 1
)

// ChangeMessage is the public change identity M(c) (ALGORITHMS.md §1.1).
type ChangeMessage struct {
	TableID  string
	RowID    []byte
	ColumnID string
	Op       Op
	Seq      uint64
	PrevTxid []byte // 32 bytes, or empty at genesis
}

// EncodeMessage encodes M(c). ALGORITHMS.md §1.1.
func EncodeMessage(m ChangeMessage) ([]byte, error) {
	if len(m.PrevTxid) != 0 && len(m.PrevTxid) != 32 {
		return nil, errors.New("prevTxid must be empty or 32 bytes")
	}
	return NewWriter().
		Raw(MagicM).
		U8(Version).
		Str(m.TableID).
		Bytes(m.RowID).
		Str(m.ColumnID).
		U8(byte(m.Op)).
		U64(m.Seq).
		Bytes(m.PrevTxid).
		Finish(), nil
}

// FieldRecord is the on-chain field record carried as spendable-script pushdata (ALGORITHMS.md §1.2).
type FieldRecord struct {
	StreamID    []byte
	Seq         uint64
	PrevTxid    []byte
	ImageKind   ImageKind
	ChangeImage []byte
	Tag         []byte // 32 bytes
}

func EncodeRecord(r FieldRecord) ([]byte, error) {
	if len(r.PrevTxid) != 0 && len(r.PrevTxid) != 32 {
		return nil, errors.New("prevTxid must be empty or 32 bytes")
	}
	if len(r.Tag) != 32 {
		return nil, errors.New("tag must be 32 bytes")
	}
	return NewWriter().
		Raw(MagicR).
		U8(Version).
		Bytes(r.StreamID).
		U64(r.Seq).
		Bytes(r.PrevTxid).
		U8(byte(r.ImageKind)).
		Bytes(r.ChangeImage).
		Bytes(r.Tag).
		Finish(), nil
}

func DecodeRecord(buf []byte) (FieldRecord, error) {
	var rec FieldRecord
	rd := NewReader(buf)
	magic := rd.Raw(4)
	if rd.Err() == nil && string(magic) != string(MagicR) {
		return rec, errors.New("bad record magic")
	}
	if v := rd.U8(); rd.Err() == nil && v != Version {
		return rec, errors.New("unsupported record version")
	}
	rec.StreamID = rd.Bytes()
	rec.Seq = rd.U64()
	rec.PrevTxid = rd.Bytes()
	kind := rd.U8()
	if rd.Err() == nil && kind != byte(ImagePlaintext) && kind != byte(ImageCommitment) {
		return rec, errors.New("bad image_kind")
	}
	rec.ImageKind = ImageKind(kind)
	rec.ChangeImage = rd.Bytes()
	rec.Tag = rd.Bytes()
	if err := rd.End(); err != nil {
		return rec, err
	}
	if len(rec.Tag) != 32 {
		return rec, errors.New("tag must be 32 bytes")
	}
	return rec, nil
}
