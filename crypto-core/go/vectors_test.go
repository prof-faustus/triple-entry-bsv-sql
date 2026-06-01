package cryptocore

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"testing"
)

func mustHex(t *testing.T, s string) []byte {
	t.Helper()
	b, err := hex.DecodeString(s)
	if err != nil {
		t.Fatalf("bad hex %q: %v", s, err)
	}
	return b
}

func eqHex(t *testing.T, name string, got []byte, want string) {
	t.Helper()
	if h := hex.EncodeToString(got); h != want {
		t.Errorf("%s: got %s want %s", name, h, want)
	}
}

type partyJSON struct{ Priv, Pub string }
type msgJSON struct {
	TableID  string `json:"tableId"`
	RowID    string `json:"rowId"`
	ColumnID string `json:"columnId"`
	Op       uint8  `json:"op"`
	Seq      string `json:"seq"`
	PrevTxid string `json:"prevTxid"`
}
type expectJSON struct {
	EncodedMessage     string `json:"encodedMessage"`
	GV                 string `json:"gv"`
	SubPubWriter       string `json:"subPubWriter"`
	SubPubCounterparty string `json:"subPubCounterparty"`
	CS                 string `json:"cs"`
	KHmac              string `json:"kHmac"`
	ChangeImage        string `json:"changeImage"`
	Tag                string `json:"tag"`
	Commit             string `json:"commit"`
	EncodedRecord      string `json:"encodedRecord"`
}
type caseJSON struct {
	Name      string     `json:"name"`
	Message   msgJSON    `json:"message"`
	Value     string     `json:"value"`
	Blinding  string     `json:"blinding"`
	ImageKind uint8      `json:"imageKind"`
	Expect    expectJSON `json:"expect"`
}
type aeadJSON struct {
	Name, Key, Nonce, Aad, Plaintext, Ciphertext, Tag string
}
type coreVectors struct {
	Parties struct {
		Writer       partyJSON `json:"writer"`
		Counterparty partyJSON `json:"counterparty"`
	} `json:"parties"`
	Cases []caseJSON `json:"cases"`
	Aead  []aeadJSON `json:"aead"`
}

func loadCore(t *testing.T) coreVectors {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join("..", "vectors", "core_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v coreVectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	return v
}

func TestCoreVectors(t *testing.T) {
	v := loadCore(t)
	wPriv := mustHex(t, v.Parties.Writer.Priv)
	cPriv := mustHex(t, v.Parties.Counterparty.Priv)
	wPub := mustHex(t, v.Parties.Writer.Pub)
	cPub := mustHex(t, v.Parties.Counterparty.Pub)

	eqHex(t, "writer.pub", PubFromPriv(wPriv), v.Parties.Writer.Pub)
	eqHex(t, "counterparty.pub", PubFromPriv(cPriv), v.Parties.Counterparty.Pub)

	for _, tc := range v.Cases {
		seq, err := strconv.ParseUint(tc.Message.Seq, 10, 64)
		if err != nil {
			t.Fatalf("%s: seq: %v", tc.Name, err)
		}
		m := ChangeMessage{
			TableID:  tc.Message.TableID,
			RowID:    mustHex(t, tc.Message.RowID),
			ColumnID: tc.Message.ColumnID,
			Op:       Op(tc.Message.Op),
			Seq:      seq,
			PrevTxid: mustHex(t, tc.Message.PrevTxid),
		}
		value := mustHex(t, tc.Value)
		r := mustHex(t, tc.Blinding)
		kind := ImageKind(tc.ImageKind)

		enc, err := EncodeMessage(m)
		if err != nil {
			t.Fatalf("%s: encode: %v", tc.Name, err)
		}
		eqHex(t, tc.Name+" encodedMessage", enc, tc.Expect.EncodedMessage)

		gvBytes, gv, err := GeneratorValue(m)
		if err != nil {
			t.Fatalf("%s: gv: %v", tc.Name, err)
		}
		eqHex(t, tc.Name+" gv", gvBytes, tc.Expect.GV)

		subW, _ := SubPub(wPub, gv)
		subC, _ := SubPub(cPub, gv)
		eqHex(t, tc.Name+" P2_W", subW, tc.Expect.SubPubWriter)
		eqHex(t, tc.Name+" P2_C", subC, tc.Expect.SubPubCounterparty)

		csW, err := CommonSecretAsWriter(wPriv, cPub, gv)
		if err != nil {
			t.Fatalf("%s: csW: %v", tc.Name, err)
		}
		csC, err := CommonSecretAsCounterparty(cPriv, wPub, gv)
		if err != nil {
			t.Fatalf("%s: csC: %v", tc.Name, err)
		}
		eqHex(t, tc.Name+" cs(writer)", csW, tc.Expect.CS)
		eqHex(t, tc.Name+" cs(counterparty symmetry)", csC, tc.Expect.CS)

		k, err := DeriveHMACKey(csW, m)
		if err != nil {
			t.Fatalf("%s: kHmac: %v", tc.Name, err)
		}
		eqHex(t, tc.Name+" kHmac", k, tc.Expect.KHmac)

		img := ChangeImage(kind, value, r)
		eqHex(t, tc.Name+" changeImage", img, tc.Expect.ChangeImage)
		eqHex(t, tc.Name+" tag", Tag(k, img), tc.Expect.Tag)
		eqHex(t, tc.Name+" commit", Commit(value, r), tc.Expect.Commit)

		rec, err := EncodeRecord(FieldRecord{StreamID: []byte(m.TableID), Message: m, ImageKind: kind, ChangeImage: img, Tag: Tag(k, img)})
		if err != nil {
			t.Fatalf("%s: record: %v", tc.Name, err)
		}
		eqHex(t, tc.Name+" encodedRecord", rec, tc.Expect.EncodedRecord)
	}
}

func TestAEADVectors(t *testing.T) {
	v := loadCore(t)
	for _, a := range v.Aead {
		key, nonce, aad, pt := mustHex(t, a.Key), mustHex(t, a.Nonce), mustHex(t, a.Aad), mustHex(t, a.Plaintext)
		ct, tag, err := AES256GCMEncrypt(key, nonce, aad, pt)
		if err != nil {
			t.Fatalf("%s: encrypt: %v", a.Name, err)
		}
		eqHex(t, a.Name+" ct", ct, a.Ciphertext)
		eqHex(t, a.Name+" tag", tag, a.Tag)
		dec, err := AES256GCMDecrypt(key, nonce, aad, ct, tag)
		if err != nil || !bytes.Equal(dec, pt) {
			t.Fatalf("%s: decrypt round-trip failed: %v", a.Name, err)
		}
		bad := append([]byte{}, ct...)
		bad[0] ^= 0x01
		if _, err := AES256GCMDecrypt(key, nonce, aad, bad, tag); err == nil {
			t.Fatalf("%s: tamper must fail", a.Name)
		}
	}
}

type rfcVectors struct {
	SHA256 []struct {
		Name      string `json:"name"`
		InputUTF8 string `json:"input_utf8"`
		Digest    string `json:"digest"`
	} `json:"sha256"`
	HMAC []struct{ Name, Key, Data, Mac string } `json:"hmac_sha256"`
	HKDF []struct {
		Name, Ikm, Salt, Info, Okm string
		Length                     int
	} `json:"hkdf_sha256"`
}

func TestRFCKATs(t *testing.T) {
	raw, err := os.ReadFile(filepath.Join("..", "vectors", "rfc_vectors.json"))
	if err != nil {
		t.Fatal(err)
	}
	var v rfcVectors
	if err := json.Unmarshal(raw, &v); err != nil {
		t.Fatal(err)
	}
	for _, c := range v.SHA256 {
		eqHex(t, c.Name, SHA256([]byte(c.InputUTF8)), c.Digest)
	}
	for _, c := range v.HMAC {
		eqHex(t, c.Name, HMACSHA256(mustHex(t, c.Key), mustHex(t, c.Data)), c.Mac)
	}
	for _, c := range v.HKDF {
		out, err := HKDFSHA256(mustHex(t, c.Salt), mustHex(t, c.Ikm), mustHex(t, c.Info), c.Length)
		if err != nil {
			t.Fatal(err)
		}
		eqHex(t, c.Name, out, c.Okm)
	}
}
