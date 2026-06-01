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

// DecodeMessage is the inverse of EncodeMessage (ALGORITHMS.md §1.1).
func DecodeMessage(buf []byte) (ChangeMessage, error) {
	var m ChangeMessage
	rd := NewReader(buf)
	magic := rd.Raw(4)
	if rd.Err() == nil && string(magic) != string(MagicM) {
		return m, errors.New("bad message magic")
	}
	if v := rd.U8(); rd.Err() == nil && v != Version {
		return m, errors.New("unsupported message version")
	}
	m.TableID = rd.Str()
	m.RowID = rd.Bytes()
	m.ColumnID = rd.Str()
	op := rd.U8()
	if rd.Err() == nil && op != byte(OpInsert) && op != byte(OpUpdate) && op != byte(OpDelete) {
		return m, errors.New("bad op")
	}
	m.Op = Op(op)
	m.Seq = rd.U64()
	m.PrevTxid = rd.Bytes()
	if err := rd.End(); err != nil {
		return m, err
	}
	if len(m.PrevTxid) != 0 && len(m.PrevTxid) != 32 {
		return m, errors.New("prevTxid must be empty or 32 bytes")
	}
	return m, nil
}

// FieldRecord is the on-chain field record carried as spendable-script pushdata (ALGORITHMS.md §1.2).
// It embeds the full change identity M(c) so the record is self-describing for cold-rebuild.
type FieldRecord struct {
	StreamID    []byte
	Message     ChangeMessage
	ImageKind   ImageKind
	ChangeImage []byte
	Tag         []byte // 32 bytes
}

func EncodeRecord(r FieldRecord) ([]byte, error) {
	if len(r.Tag) != 32 {
		return nil, errors.New("tag must be 32 bytes")
	}
	mEnc, err := EncodeMessage(r.Message)
	if err != nil {
		return nil, err
	}
	return NewWriter().
		Raw(MagicR).
		U8(Version).
		Bytes(r.StreamID).
		Bytes(mEnc).
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
	mEnc := rd.Bytes()
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
	m, err := DecodeMessage(mEnc)
	if err != nil {
		return rec, err
	}
	rec.Message = m
	return rec, nil
}
