package custody

import (
	"encoding/hex"
	"math/big"
	"testing"

	cc "te-bsv/cryptocore"
)

func TestShamirLossResistant(t *testing.T) {
	secret, _ := new(big.Int).SetString("e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262", 16)
	shares, err := Split(secret, 3, 5)
	if err != nil {
		t.Fatal(err)
	}
	// any 3 recover the secret (loss-resistant: survives 2 missing shares)
	if got := Recover([]Share{shares[0], shares[2], shares[4]}); got.Cmp(secret) != 0 {
		t.Fatalf("3-of-5 recover mismatch")
	}
	if got := Recover([]Share{shares[1], shares[3], shares[0]}); got.Cmp(secret) != 0 {
		t.Fatalf("3-of-5 recover (other subset) mismatch")
	}
	// fewer than threshold must NOT recover the secret
	if got := Recover([]Share{shares[0], shares[1]}); got.Cmp(secret) == 0 {
		t.Fatalf("2 shares must not reconstruct the secret")
	}
}

func TestShareEncryptionUnderCommonSecret(t *testing.T) {
	// CS from the keystone (writer<->counterparty); shares stored encrypted under it (EP3259724B1).
	wPriv, _ := hex.DecodeString("e9873d79c6d87dc0fb6a5778633389f4453213303da61f20bd67fc233aa33262")
	cPriv, _ := hex.DecodeString("f8b8af8ce3c7cca5e300d33939540c10d45ce001b8f252bfbc57ba0342904181")
	cPub := cc.PubFromPriv(cPriv)
	m := cc.ChangeMessage{TableID: "custody", RowID: []byte{1}, ColumnID: "share", Op: cc.OpInsert, Seq: 0}
	_, gv, err := cc.GeneratorValue(m)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := cc.CommonSecretAsWriter(wPriv, cPub, gv)
	if err != nil {
		t.Fatal(err)
	}
	s := Share{X: big.NewInt(1), Y: big.NewInt(123456789)}
	ct, tag, err := EncryptShare(cs, "vault-A", s)
	if err != nil || len(ct) < 12 || len(tag) != 16 {
		t.Fatalf("encrypt share: ct=%d tag=%d err=%v", len(ct), len(tag), err)
	}
}

func TestStorageLayoutAndMultisig(t *testing.T) {
	l := StorageLayout{Locations: []string{"vault-A", "vault-B", "backup-safe"}, Backup: "backup-safe"}
	if !l.Valid() {
		t.Fatal("layout with 3 locations incl backup must be valid")
	}
	if (StorageLayout{Locations: []string{"a", "b"}, Backup: "b"}).Valid() {
		t.Fatal("fewer than 3 locations must be invalid")
	}
	// native 2-of-3 bare multisig (no P2SH)
	pks := [][]byte{
		cc.PubFromPriv(mustHex32(1)), cc.PubFromPriv(mustHex32(2)), cc.PubFromPriv(mustHex32(3)),
	}
	ms, err := BuildBareMultisig(2, pks)
	if err != nil {
		t.Fatal(err)
	}
	if ms.IsP2SH() {
		t.Fatal("bare multisig must not be P2SH")
	}
	if !ms.IsMultiSigOut() {
		t.Fatal("expected a bare multisig output")
	}
}

func mustHex32(b byte) []byte {
	out := make([]byte, 32)
	out[31] = b
	return out
}
